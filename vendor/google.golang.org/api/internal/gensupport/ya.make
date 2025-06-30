GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.224.0)

SRCS(
    buffer.go
    doc.go
    error.go
    json.go
    jsonfloat.go
    media.go
    params.go
    resumable.go
    retry.go
    send.go
    version.go
)

GO_TEST_SRCS(
    buffer_test.go
    error_test.go
    json_test.go
    jsonfloat_test.go
    media_test.go
    params_test.go
    resumable_test.go
    retry_test.go
    send_test.go
    util_test.go
    version_test.go
)

END()

RECURSE(
    gotest
)
