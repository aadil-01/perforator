GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.26.0)

SRCS(
    big5.go
    tables.go
)

GO_TEST_SRCS(all_test.go)

END()

RECURSE(
    gotest
)
