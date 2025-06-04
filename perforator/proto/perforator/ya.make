PROTO_LIBRARY()

INCLUDE_TAGS(GO_PROTO)

IF (OPENSOURCE)
    EXCLUDE_TAGS(JAVA_PROTO)
ENDIF()

GRPC()

SRCS(
    perforator.proto
    microscope_service.proto
    record_remote.proto
    task_service.proto
)

GO_GRPC_GATEWAY_V2_SRCS(
    perforator.proto
    task_service.proto
)

IF (NOT GO_PROTO)
    PEERDIR(
        contrib/libs/googleapis-common-protos
    )
ENDIF()

USE_COMMON_GOOGLE_APIS(api/annotations)

END()
