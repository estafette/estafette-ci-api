FROM scratch

LABEL maintainer="estafette.io"

COPY ca-certificates.crt /etc/ssl/certs/
COPY estafette-ci-api /

ENTRYPOINT ["/estafette-ci-api"]
