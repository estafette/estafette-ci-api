# FROM ubuntu:latest
# RUN useradd -u 10001 scratchuser

FROM scratch

LABEL maintainer="estafette.io" \
      description="The ${ESTAFETTE_GIT_NAME} is the component that handles api requests and starts build jobs using the estafette-ci-builder"

COPY ca-certificates.crt /etc/ssl/certs/
COPY ${ESTAFETTE_GIT_NAME} /

# # run as non-root user
# COPY --from=0 /etc/passwd /etc/passwd
# USER scratchuser

ENV GRACEFUL_SHUTDOWN_DELAY_SECONDS="20" \
    ESTAFETTE_LOG_FORMAT="json"

ENTRYPOINT ["/${ESTAFETTE_GIT_NAME}"]