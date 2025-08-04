GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.6.5)

SRCS(
    longrunning.go
)

GO_TEST_SRCS(
    example_test.go
    longrunning_test.go
)

END()

RECURSE(
    autogen
    gotest
    internal
)
