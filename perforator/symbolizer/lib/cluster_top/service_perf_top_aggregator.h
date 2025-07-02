#pragma once

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>
#include <library/cpp/int128/int128.h>

#include <perforator/symbolizer/lib/gsym/gsym_symbolizer.h>

#include <util/generic/array_ref.h>

#include <vector>
#include <string>

namespace NPerforator::NProto::NPProf {
class ProfileLight;
}

namespace NPerforator::NClusterTop {

class TCachingGSYMSymbolizer final {
public:
    TCachingGSYMSymbolizer(std::string_view gsymPath);

    const std::vector<std::string>& Symbolize(ui64 addr);

private:
    NPerforator::NGsym::TSymbolizer Symbolizer_;

    absl::flat_hash_map<ui64, std::vector<std::string>> SymbolizationCache_;
};

class TServicePerfTopAggregator final {
public:
    TServicePerfTopAggregator();

    void InitializeSymbolizer(TArrayRef<const char> buildId, TArrayRef<const char> gsymPath);

    void AddProfile(TArrayRef<const char> service, TArrayRef<const char> profileBytes);

    void AddProfile(TArrayRef<const char> service, const NPerforator::NProto::NPProf::ProfileLight& profile);

    void MergeAggregator(const TServicePerfTopAggregator& other);

    struct Function final {
        TString Name;
        ui128 SelfCycles;
        ui128 CumulativeCycles;
    };
    struct PerfTop final {
        std::vector<Function> Functions;
        ui128 TotalCycles;
    };
    PerfTop ExtractEntries();

private:
    absl::flat_hash_map<TString, TCachingGSYMSymbolizer> Symbolizers_;

    absl::flat_hash_map<TString, ui128, THash<TString>, TEqualTo<TString>> CumulativeCycles_;
    absl::flat_hash_map<TString, ui128, THash<TString>, TEqualTo<TString>> SelfCycles_;
    ui128 TotalCycles_{0};
};

}
