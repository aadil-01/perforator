# yo ignore:file
GO_LIBRARY()

USE_UTIL()

IF (CGO_ENABLED AND NOT SANDBOXING)
    PEERDIR(
        perforator/lib/profile/c
    )

    CGO_SRCS(
        merge_cgo.go
        error_cgo.go
    )
ELSE()
    SRCS(
        merge_nocgo.go
    )
ENDIF()

END()

RECURSE(cmd)
