FROM gcr.io/distroless/static:nonroot
# GoReleaser dockers_v2 lays the build context out as <os>/<arch>/<binary>,
# so the per-platform binary must be copied via the buildx $TARGETPLATFORM arg.
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/zftp /usr/bin/zftp
ENTRYPOINT ["/usr/bin/zftp"]
