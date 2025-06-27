#include "service_perf_top_aggregator.h"

#include <perforator/proto/pprofprofile/profile.pb.h>
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

const TString kUnknownLocation = "<UNKNOWN>";
const TString kNoGSYMLocation = "<UNKNOWN (No GSYM)>";

ui64 GetCpuCyclesValue(
    const NPerforator::NProto::NPProf::Profile& profile,
    const NPerforator::NProto::NPProf::Sample& sample) {
    for (std::size_t i = 0; i < profile.sample_typeSize(); ++i) {
        const auto& sampleType = profile.sample_type(i);
        if (profile.string_table(sampleType.type()) == kCPUCyclesType &&
            profile.string_table(sampleType.unit()) == kCPUCyclesUnit) {
            return sample.value(i);
        }
    }

    return 0;
}

}

TCachingGSYMSymbolizer::TCachingGSYMSymbolizer(std::string_view gsymPath) : Symbolizer_{gsymPath} {
    SymbolizationCache_.reserve(128 * 1024);
}

const NPerforator::NGsym::TSmallVector<llvm::DILineInfo>& TCachingGSYMSymbolizer::Symbolize(ui64 addr) {
    auto [it, inserted] = SymbolizationCache_.try_emplace(addr);
    if (inserted) {
        it->second = Symbolizer_.Symbolize(addr);
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

    NPerforator::NProto::NPProf::Profile profile{};
    if (!profile.ParseFromString(std::string_view{profileBytes.data(), profileBytes.size()})) {
        return;
    }

    AddProfile(service, profile);
}

void TServicePerfTopAggregator::AddProfile(TArrayRef<const char> service, const NPerforator::NProto::NPProf::Profile& profile) {
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

    absl::flat_hash_map<ui64, TSmallVector<TString>> symbolizedLocationsById;
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
                    symbolized.push_back(kUnknownLocation);
                    continue;
                }
                const auto& function = *functionByIdMap.at(functionId);
                symbolized.push_back(profile.string_table(function.name()));
            }
        } else {
            [&symbolized, &location, &mappingByIdMap, &profile, this]() {
                const auto mappingId = location.mapping_id();
                if (mappingId == 0) {
                    symbolized.push_back(kUnknownLocation);
                    return;
                }
                const auto& mapping = *mappingByIdMap.at(mappingId);

                if (mapping.build_id() == 0) {
                    symbolized.push_back(kUnknownLocation);
                    return;
                }
                const auto& buildId = profile.string_table(mapping.build_id());

                auto symbolizerIt = Symbolizers_.find(buildId);
                if (symbolizerIt == Symbolizers_.end()) {
                    symbolized.push_back(kNoGSYMLocation);
                    return;
                }
                auto& symbolizer = symbolizerIt->second;

                const auto address = location.address() + mapping.file_offset() - mapping.memory_start();
                const auto& symbolizationResult = symbolizer.Symbolize(address);

                if (symbolizationResult.empty()) {
                    symbolized.push_back(kUnknownLocation);
                    return;
                }

                for (const auto& frame : symbolizationResult) {
                    symbolized.push_back(TString{frame.FunctionName});
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
        SelfCycles_[frame.back()] += value;
    }

    for (const auto& [id, frames] : symbolizedLocationsById) {
        const auto value = cumulativeCyclesCountByLocationId[id];

        for (const auto& frame : frames) {
            CumulativeCycles_[frame] += value;
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
