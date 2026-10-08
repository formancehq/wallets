FROM ghcr.io/formancehq/base:22.04
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/wallets /usr/bin/wallets
ENV OTEL_SERVICE_NAME wallets
ENTRYPOINT ["/usr/bin/wallets"]
CMD ["server"]
