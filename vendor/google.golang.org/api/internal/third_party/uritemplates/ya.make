GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.228.0)

SRCS(
    uritemplates.go
    utils.go
)

GO_TEST_SRCS(uritemplates_test.go)

END()

RECURSE(
    gotest
)
