GO_LIBRARY()

SRCS(
    kubelet.go
    pod.go
    prometheus.go
    transport.go
)

GO_TEST_SRCS(kubelet_test.go)

GO_TEST_EMBED_PATTERN(kubelet-configz-response.json)

END()

RECURSE(
    gotest
)
