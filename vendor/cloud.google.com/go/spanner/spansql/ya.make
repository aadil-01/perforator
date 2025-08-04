GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.76.1)

SRCS(
    keywords.go
    parser.go
    sql.go
    types.go
)

GO_TEST_SRCS(
    parser_test.go
    sql_test.go
)

END()

RECURSE(
    gotest
)
