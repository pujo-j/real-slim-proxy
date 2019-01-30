FROM alpine
WORKDIR /app
COPY real-slim-proxy /app/
VOLUME ["/config"]
CMD ./real-slim-proxy --config /config/config.yaml

