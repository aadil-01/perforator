GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.23.0)

SRCS(
    all.go
    gbk.go
    hzgb2312.go
    tables.go
)

GO_TEST_SRCS(all_test.go)

END()

RECURSE(
    gotest
)
