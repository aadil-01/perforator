GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.15.0)

GO_SKIP_TESTS(TestEnterpriseCertificateProxySource_GetClientCertificateSuccess)

SRCS(
    default_cert.go
    enterprise_cert.go
    secureconnect_cert.go
    workload_cert.go
)

GO_TEST_SRCS(
    enterprise_cert_test.go
    # secureconnect_cert_test.go
    workload_cert_test.go
)

END()

RECURSE(
    cmd
    gotest
)
