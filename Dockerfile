FROM alpine
# Pretend we have glibc, it actually works !
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

RUN addgroup -g 110 -S slim && adduser -h /app -u 110 -G slim -D slim
USER 110:110
WORKDIR /app
COPY real-slim-proxy /app/
VOLUME ["/config"]
CMD ./real-slim-proxy --config /config/config.yaml
