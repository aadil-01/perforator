GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.26.0)

SRCS(
    format.go
    parser.go
)

GO_TEST_SRCS(parser_test.go)

END()

RECURSE(
    gotest
)
