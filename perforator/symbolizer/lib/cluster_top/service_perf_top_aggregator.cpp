#include "service_perf_top_aggregator.h"

#include <perforator/proto/pprofprofile/lightweightprofile.pb.h>
#include <perforator/symbolizer/lib/symbolize/symbolizer.h>

#include <library/cpp/yt/compact_containers/compact_vector.h>

#include <algorithm>
#include <string_view>

namespace NPerforator::NClusterTop {

namespace {

template<typename T>
using TSmallVector = NYT::TCompactVector<T, 4>;

constexpr std::size_t kMaxEntriesToPrint = 10'000;

constexpr std::string_view kCPUCyclesType = "cpu";
constexpr std::string_view kCPUCyclesUnit = "cycles";

// TODO : PERFORATOR-886
constexpr std::string_view kUnknownLocation = "<UNKNOWN>";
constexpr std::string_view kNoGSYMLocation = "<UNKNOWN (No GSYM)>";

ui64 GetCpuCyclesValue(
    const NPerforator::NProto::NPProf::ProfileLight& profile,
    const NPerforator::NProto::NPProf::SampleLight& sample) {
    for (std::size_t i = 0; i < profile.sample_typeSize(); ++i) {
        const auto& sampleType = profile.sample_type(i);
        if (profile.string_table(sampleType.type()) == kCPUCyclesType &&
            profile.string_table(sampleType.unit()) == kCPUCyclesUnit) {
            return sample.value(i);
        }
    }

    return 0;
}

struct TLifetimeSoundnessReason final {
    explicit constexpr TLifetimeSoundnessReason(std::string_view) {}
};

// A string_view-like class, *implicitly* convertible to TString.
// Used for try_emplace-ing a string_view into a HashMap<TString, ...>
class TStringViewConvertibleToString final {
public:
    TStringViewConvertibleToString(const TString&) = delete;
    TStringViewConvertibleToString(const std::string&) = delete;

    explicit constexpr TStringViewConvertibleToString(TStringBuf data) : Data_{data} {}
    explicit constexpr TStringViewConvertibleToString(std::string_view data) : Data_{data} {}

    TStringViewConvertibleToString(const TString& data, TLifetimeSoundnessReason) : Data_{data} {}
    TStringViewConvertibleToString(const std::string& data, TLifetimeSoundnessReason) : Data_{data} {}
    TStringViewConvertibleToString(std::string_view data, TLifetimeSoundnessReason) : Data_{data} {}

    constexpr operator TStringBuf() const {
        return Data_;
    }

    operator TString() const {
        return TString{Data_};
    }
private:
    TStringBuf Data_;
};

}

TCachingGSYMSymbolizer::TCachingGSYMSymbolizer(std::string_view gsymPath) : Symbolizer_{gsymPath} {
    SymbolizationCache_.reserve(128 * 1024);
}

const std::vector<std::string>& TCachingGSYMSymbolizer::Symbolize(ui64 addr) {
    auto [it, inserted] = SymbolizationCache_.try_emplace(addr);
    if (inserted) {
        auto& frames = it->second;

        auto symbolizationResult = Symbolizer_.Symbolize(addr);
        frames.reserve(symbolizationResult.size());
        for (auto& frame : symbolizationResult) {
            frames.push_back(std::move(frame.FunctionName));
        }
    }

    return it->second;
}

TServicePerfTopAggregator::TServicePerfTopAggregator() {}

void TServicePerfTopAggregator::InitializeSymbolizer(
    TArrayRef<const char> buildId,
    TArrayRef<const char> gsymPath
) {
    Symbolizers_.emplace(
        TString{buildId.data(), buildId.size()},
        std::string_view{gsymPath.data(), gsymPath.size()}
    );
}

void TServicePerfTopAggregator::AddProfile(TArrayRef<const char> service, TArrayRef<const char> profileBytes) {
    if (profileBytes.data() == nullptr || profileBytes.size() == 0) {
        return;
    }

    NPerforator::NProto::NPProf::ProfileLight profile{};
    if (!profile.ParseFromString(std::string_view{profileBytes.data(), profileBytes.size()})) {
        return;
    }

    AddProfile(service, profile);
}

void TServicePerfTopAggregator::AddProfile(TArrayRef<const char> service, const NPerforator::NProto::NPProf::ProfileLight& profile) {
    absl::flat_hash_map<ui64, const NPerforator::NProto::NPProf::Function*> functionByIdMap;
    for (const auto& function : profile.Getfunction()) {
        functionByIdMap[function.id()] = &function;
    }

    absl::flat_hash_map<ui64, const NPerforator::NProto::NPProf::Mapping*> mappingByIdMap;
    for (const auto& mapping : profile.Getmapping()) {
        mappingByIdMap[mapping.id()] = &mapping;
    }

    absl::flat_hash_map<ui64, const NPerforator::NProto::NPProf::Location*> locationByIdMap;
    for (const auto& location : profile.Getlocation()) {
        locationByIdMap[location.id()] = &location;
    }

    // Every string, for which the views in this map are stored, outlives the map:
    // * some strings are static/constexpr
    // * some strings belong to the profile
    // * some strings belong to the TServicePerfTopAggregator (i.e. symbolization cache)
    absl::flat_hash_map<ui64, TSmallVector<TStringViewConvertibleToString>> symbolizedLocationsById;
    symbolizedLocationsById.reserve(profile.locationSize());

    for (const auto& location : profile.Getlocation()) {
        const auto locationId = location.id();

        if (symbolizedLocationsById.count(locationId) != 0) {
            continue;
        }

        auto& symbolized = symbolizedLocationsById.try_emplace(locationId).first->second;
        if (location.lineSize() > 0) {
            // already symbolized, probably kernel function
            for (const auto& line : location.Getline()) {
                const auto functionId = line.function_id();
                if (functionId == 0) {
                    symbolized.emplace_back(kUnknownLocation, TLifetimeSoundnessReason{"kUnknownLocation is static"});
                    continue;
                }
                const auto& function = *functionByIdMap.at(functionId);
                symbolized.emplace_back(
                    profile.string_table(function.name()),
                    TLifetimeSoundnessReason{"profile outlives everything in this function"}
                );
            }
        } else {
            [&symbolized, &location, &mappingByIdMap, &profile, this]() {
                const auto mappingId = location.mapping_id();
                if (mappingId == 0) {
                    symbolized.emplace_back(kUnknownLocation, TLifetimeSoundnessReason{"kUnknownLocation is static"});
                    return;
                }
                const auto& mapping = *mappingByIdMap.at(mappingId);

                if (mapping.build_id() == 0) {
                    symbolized.emplace_back(kUnknownLocation, TLifetimeSoundnessReason{"kUnknownLocation is static"});
                    return;
                }
                const auto& buildId = profile.string_table(mapping.build_id());

                auto symbolizerIt = Symbolizers_.find(buildId);
                if (symbolizerIt == Symbolizers_.end()) {
                    symbolized.emplace_back(kNoGSYMLocation, TLifetimeSoundnessReason{"kNoGSYMLocation is static"});
                    return;
                }
                auto& symbolizer = symbolizerIt->second;

                const auto address = location.address() + mapping.file_offset() - mapping.memory_start();
                const auto& symbolizationResult = symbolizer.Symbolize(address);

                if (symbolizationResult.empty()) {
                    symbolized.emplace_back(kUnknownLocation, TLifetimeSoundnessReason{"kUnknownLocation is static"});
                    return;
                }

                for (const auto& frame : symbolizationResult) {
                    symbolized.emplace_back(
                        frame,
                        TLifetimeSoundnessReason{"symbolizationResult is cached in symbolizer, thus its lifetime is tied to the aggregator"}
                    );
                }
            }();
        }
    }

    absl::flat_hash_map<ui64, ui64> cumulativeCyclesCountByLocationId;
    cumulativeCyclesCountByLocationId.reserve(profile.locationSize());

    TString serviceStr{service.data(), service.size()};

    for (const auto& sample : profile.Getsample()) {
        const auto value = GetCpuCyclesValue(profile, sample);

        // TODO : PERFORATOR-856, count every unique location only once here,
        // otherwise recursive functions and "UNKNOWN" functions get the wrong count.
        // Note that "UNKNOWN" functions are harder to distinguish, as different locations might
        // be UNKNOWN. So not just unique location, but rather unique function/its inlined callchain.
        for (const auto& locationId : sample.Getlocation_id()) {
            cumulativeCyclesCountByLocationId[locationId] += value;
        }

        TotalCycles_ += value;

        if (sample.location_idSize() == 0) {
            continue;
        }

        const auto leafLocationId = sample.location_id(0);
        const auto& frame = symbolizedLocationsById[leafLocationId];
        if (frame.empty()) {
            continue;
        }
        SelfCycles_.try_emplace<TStringViewConvertibleToString>(frame.back(), 0).first->second += value;
    }

    for (const auto& [id, frames] : symbolizedLocationsById) {
        const auto value = cumulativeCyclesCountByLocationId[id];

        for (const auto& frame : frames) {
            CumulativeCycles_.try_emplace<TStringViewConvertibleToString>(frame, 0).first->second += value;
        }
    }
}

void TServicePerfTopAggregator::MergeAggregator(const TServicePerfTopAggregator& other) {
    for (const auto& [k, v] : other.CumulativeCycles_) {
        CumulativeCycles_[k] += v;
    }

    for (const auto& [k, v] : other.SelfCycles_) {
        SelfCycles_[k] += v;
    }

    TotalCycles_ += other.TotalCycles_;
}

TServicePerfTopAggregator::PerfTop TServicePerfTopAggregator::ExtractEntries() {
    const auto sortAndDemangle = [](const auto& cyclesByFunction) {
        std::vector<std::pair<TString, ui128>> total{cyclesByFunction.begin(), cyclesByFunction.end()};
        std::sort(total.begin(), total.end(), [](const auto& lhs, const auto& rhs) {
            return lhs.second > rhs.second;
        });

        if (total.size() > kMaxEntriesToPrint) {
            total.resize(kMaxEntriesToPrint);
        }

        for (auto& [name, _] : total) {
            name = NPerforator::NSymbolize::CleanupFunctionName(
                NPerforator::NSymbolize::DemangleFunctionName(name)
            );
        }

        return total;
    };

    auto selfCycles = sortAndDemangle(SelfCycles_);
    auto cumulativeCycles = sortAndDemangle(CumulativeCycles_);

    struct CyclesCount final {
        ui128 SelfCycles = 0;
        ui128 CumulativeCycles = 0;
    };
    absl::flat_hash_map<TStringBuf, CyclesCount> cyclesCount;
    for (const auto& [name, cycles] : selfCycles) {
        cyclesCount[name].SelfCycles += cycles;
    }
    for (const auto& [name, cycles] : cumulativeCycles) {
        cyclesCount[name].CumulativeCycles += cycles;
    }

    std::vector<Function> functions;
    functions.reserve(cyclesCount.size());
    for (const auto& [name, cycles] : cyclesCount) {
        functions.push_back(Function{
            .Name = TString{name},
            .SelfCycles = cycles.SelfCycles,
            .CumulativeCycles = cycles.CumulativeCycles,
        });
    }

    return PerfTop{
        .Functions = std::move(functions),
        .TotalCycles = TotalCycles_
    };
}

}
