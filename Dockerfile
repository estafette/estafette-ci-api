# FROM ubuntu:latest
# RUN useradd -u 10001 scratchuser

FROM apline:3.17

LABEL maintainer="estafette.io" \
      description="The ${ESTAFETTE_GIT_NAME} is the component that handles api requests and starts build jobs using the estafette-ci-builder"

RUN apk add bash procps

COPY ca-certificates.crt /etc/ssl/certs/
COPY ${ESTAFETTE_GIT_NAME} /
COPY gcs-migrator/dist/gcs-migrator /
COPY entrypoint.sh /
COPY healthcheck.sh /

# # run as non-root user
# COPY --from=0 /etc/passwd /etc/passwd
# USER scratchuser

ENV GRACEFUL_SHUTDOWN_DELAY_SECONDS="20" \
    ESTAFETTE_LOG_FORMAT="json"

HEALTHCHECK --interval=10s --retries=5 CMD /healthcheck.sh

ENTRYPOINT ["/entrypoint.sh"]
