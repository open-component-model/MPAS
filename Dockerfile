FROM alpine:3.18

RUN apk add --no-cache ca-certificates

COPY --chmod=755 mpas /usr/local/bin/

USER 65534:65534
ENTRYPOINT [ "mpas" ]
