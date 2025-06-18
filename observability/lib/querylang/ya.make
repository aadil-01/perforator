GO_LIBRARY()

SRCS(
    ast_repr.go
    ast.go
    ast_iter.go
    helpers.go
    parser.go
    tools.go
)

END()

RECURSE(
    operator
    parser
    template
)
