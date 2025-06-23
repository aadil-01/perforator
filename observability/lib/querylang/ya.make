GO_LIBRARY()

SRCS(
    ast_repr.go
    ast.go
    ast_iter.go
    helpers.go
    parser.go
    tools.go
)

GO_XTEST_SRCS(
    ast_iter_test.go
)

END()

RECURSE(
    gotest
    operator
    parser
    template
)
