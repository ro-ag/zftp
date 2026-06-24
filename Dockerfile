FROM gcr.io/distroless/static:nonroot
COPY zftp /usr/bin/zftp
ENTRYPOINT ["/usr/bin/zftp"]
