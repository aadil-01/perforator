#pragma once

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>
#include <library/cpp/int128/int128.h>

#include <perforator/symbolizer/lib/gsym/gsym_symbolizer.h>

#include <util/generic/array_ref.h>

namespace NPerforator::NProto::NPProf {
class Profile;
}

namespace NPerforator::NClusterTop {

class TCachingGSYMSymbolizer final {
public:
    TCachingGSYMSymbolizer(std::string_view gsymPath);

    const NPerforator::NGsym::TSmallVector<llvm::DILineInfo>& Symbolize(ui64 addr);

private:
    NPerforator::NGsym::TSymbolizer Symbolizer_;

    absl::flat_hash_map<ui64, NPerforator::NGsym::TSmallVector<llvm::DILineInfo>>
        SymbolizationCache_;
};

class TServicePerfTopAggregator final {
public:
    TServicePerfTopAggregator();

    void InitializeSymbolizer(TArrayRef<const char> buildId, TArrayRef<const char> gsymPath);

    void AddProfile(TArrayRef<const char> service, TArrayRef<const char> profileBytes);

    void AddProfile(TArrayRef<const char> service, const NPerforator::NProto::NPProf::Profile& profile);

    void MergeAggregator(const TServicePerfTopAggregator& other);

    void DebugPrint() const;

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

    absl::flat_hash_map<TString, ui128> CumulativeCycles_;
    absl::flat_hash_map<TString, ui128> SelfCycles_;
    ui128 TotalCycles_{0};
};

}
