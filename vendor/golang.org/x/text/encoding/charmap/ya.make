GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.25.0)

SRCS(
    charmap.go
    tables.go
)

GO_TEST_SRCS(charmap_test.go)

END()

RECURSE(
    gotest
)
