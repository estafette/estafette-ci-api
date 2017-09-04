FROM scratch

LABEL maintainer="estafette.io" \
      description="The estafette-ci-api is the component that handles api requests and starts build jobs using the estafette-ci-builder"

COPY ca-certificates.crt /etc/ssl/certs/
COPY estafette-ci-api /
COPY db-migrations /

ENTRYPOINT ["/estafette-ci-api"]
