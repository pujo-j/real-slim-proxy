# build stage
FROM golang:alpine AS build-env
RUN apk add git
ADD . /src
RUN cd /src && go build -o rsp

# final stage
FROM alpine
WORKDIR /app
COPY --from=build-env /src/rsp /app/
VOLUME ["/config"]
CMD ./rsp --config /config/config.yaml

