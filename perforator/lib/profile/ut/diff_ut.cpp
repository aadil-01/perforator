#include <perforator/lib/profile/flat_diffable.h>
#include <perforator/lib/profile/pprof.h>
#include <perforator/lib/profile/profile.h>

#include <library/cpp/testing/gtest/gtest.h>
#include <library/cpp/testing/common/env.h>

#include <util/folder/path.h>
#include <util/stream/file.h>
#include <util/stream/zlib.h>


static TVector<TFsPath> ListGoldenProfiles() {
    TFsPath dir = SRC_("testprofiles");
    TVector<TFsPath> children;
    dir.List(children);
    Y_ENSURE(!children.empty());
    return children;
}

static TString SerializeFlatProfile(const NPerforator::NProfile::TFlatDiffableProfile& profile) {
    TStringStream out;
    profile.WriteTo(out);
    return out.Str();
}

static TMap<TString, ui64> CountFlatProfileEvents(const NPerforator::NProfile::TFlatDiffableProfile& profile) {
    TMap<TString, ui64> sum;

    profile.IterateSamples([&](TStringBuf, const TMap<TString, ui64>& values) {
        for (auto& [type, value] : values) {
            sum[type] += value;
        }
    });

    return sum;
}

template <typename L, typename R>
static void CompareFlatProfiles(L&& lhs, R&& rhs) {
    // Our profiles a somewhat malformed
    NPerforator::NProfile::TFlatDiffableProfileOptions options{
        .LabelBlacklist = {"comm"},
    };
    NPerforator::NProfile::TFlatDiffableProfile left{lhs, options};
    NPerforator::NProfile::TFlatDiffableProfile right{rhs, options};

    auto lhsEvents = CountFlatProfileEvents(left);
    auto rhsEvents = CountFlatProfileEvents(right);
    EXPECT_EQ(lhsEvents, rhsEvents);

    TString pprofString = SerializeFlatProfile(left);
    TString protoString = SerializeFlatProfile(right);
    EXPECT_EQ(pprofString, protoString);
}

static NPerforator::NProto::NPProf::Profile ParsePprof(const TFsPath& path) {
    TFileInput serialized{path};
    TZLibDecompress uncompressed{&serialized};
    NPerforator::NProto::NPProf::Profile proto;
    Y_ENSURE(proto.ParseFromArcadiaStream(&uncompressed));
    return proto;
}

// NOLINTNEXTLINE(readability-identifier-naming)
struct GoldenProfileTest : testing::TestWithParam<TFsPath> {};

TEST_P(GoldenProfileTest, ConvertPprofCanon) {
    NPerforator::NProto::NPProf::Profile pprofProto = ParsePprof(GetParam());

    // Make new protobuf profile from pprof
    NPerforator::NProto::NProfile::Profile profileProto;
    NPerforator::NProfile::ConvertFromPProf(pprofProto, &profileProto);

    CompareFlatProfiles(pprofProto, profileProto);
}

TEST_P(GoldenProfileTest, ConvertPprofRoundTripCanon) {
    NPerforator::NProto::NPProf::Profile pprofOriginalProto = ParsePprof(GetParam());

    // Make new protobuf profile from pprof
    NPerforator::NProto::NProfile::Profile profileProto;
    NPerforator::NProfile::ConvertFromPProf(pprofOriginalProto, &profileProto);

    NPerforator::NProto::NPProf::Profile pprofConvertedProto;
    NPerforator::NProfile::ConvertToPProf(profileProto, &pprofConvertedProto);

    CompareFlatProfiles(pprofOriginalProto, pprofConvertedProto);
}

INSTANTIATE_TEST_SUITE_P(
    GoldenProfiles,
    GoldenProfileTest,
    testing::ValuesIn(ListGoldenProfiles())
);
