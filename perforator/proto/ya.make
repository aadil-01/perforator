RECURSE(
    custom_profiling_operation
    lib
    perforator
    pprofprofile
    profile
    storage
)

IF(NOT OPENSOURCE)
    RECURSE(
        yt
   )
ENDIF()
