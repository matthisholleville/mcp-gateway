FROM golang:1.24.5-alpine as builder

WORKDIR /mcp-gateway

RUN apk update && apk add --no-cache git

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 go build -ldflags "-s -w \
    -X github.com/matthisholleville/mcp-gateway/pkg/version.REVISION=${REVISION}" \
    -a -o bin/mcp-gateway

FROM alpine:3.20@sha256:77726ef6b57ddf65bb551896826ec38bc3e53f75cdde31354fbffb4f25238ebd

ARG BUILD_DATE
ARG VERSION
ARG REVISION

LABEL maintainer="matthis.holleville"

RUN addgroup -S app \
    && adduser -S -G app app \
    && apk --no-cache add \
    curl netcat-openbsd

WORKDIR /home/app

COPY --from=builder /mcp-gateway/bin/mcp-gateway .
COPY --from=builder /mcp-gateway/config/config.yaml ./config.yaml
COPY --from=builder /mcp-gateway/assets/migrations/postgres/ ./assets/migrations/postgres/

RUN chown -R app:app ./

USER app

EXPOSE 8080

ENTRYPOINT ["./mcp-gateway"]