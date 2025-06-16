PROTO_LIBRARY()

GRPC()

INCLUDE_TAGS(GO_PROTO)

IF (OPENSOURCE)
    EXCLUDE_TAGS(JAVA_PROTO)
ENDIF()

PEERDIR(
    perforator/proto/perforator
    perforator/proto/pprofprofile
)

SRCS(
    perforator_storage.proto
    task_relayer.proto
)

END()
