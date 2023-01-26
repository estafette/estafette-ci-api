FROM scratch

LABEL maintainer="estafette.io" \
      description="The ${ESTAFETTE_GIT_NAME} is the component that handles api requests and starts build jobs using the estafette-ci-builder"

ENV GRACEFUL_SHUTDOWN_DELAY_SECONDS="20" \
    ESTAFETTE_LOG_FORMAT="json"

COPY ca-certificates.crt /etc/ssl/certs/
COPY publish/${ESTAFETTE_GIT_NAME} /
COPY publish/gcs-migrator /

ENTRYPOINT ["/${ESTAFETTE_GIT_NAME}"]
