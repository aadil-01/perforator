LIBRARY()

SRCS(
    builder.cpp
    merge.cpp
    parallel_merge.cpp
    pprof.cpp
    profile.cpp
    validate.cpp
    visitor.cpp
)

PEERDIR(
    perforator/proto/pprofprofile
    perforator/proto/profile

    library/cpp/containers/absl_flat_hash
    library/cpp/containers/stack_vector
    library/cpp/introspection
    library/cpp/threading/future
    library/cpp/json
    library/cpp/yt/compact_containers
)

END()

RECURSE_FOR_TESTS(ut)
