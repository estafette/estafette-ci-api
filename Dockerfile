# FROM ubuntu:latest
# RUN useradd -u 10001 scratchuser

FROM alpine:3.17

LABEL maintainer="estafette.io" \
      description="The ${ESTAFETTE_GIT_NAME} is the component that handles api requests and starts build jobs using the estafette-ci-builder"

RUN apk --no-cache update && apk upgrade && apk add bash procps && rm -rf /var/cache/apk/*

COPY ca-certificates.crt /etc/ssl/certs/
COPY publish/${ESTAFETTE_GIT_NAME} /
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
