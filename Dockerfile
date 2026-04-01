FROM golang:1.25@sha256:5a79b94c34c299ac0361fbb7c7fca6dc552e166b42341050323fa3ab137d7be9 AS builder

COPY . /build

RUN cd /build && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' ./cmd/office365-smtp-proxy

FROM gcr.io/distroless/base-debian12:nonroot@sha256:8b9f2e503e55aff85b79d6b22c7a63a65170e8698ae80de680e3f5ea600977bf

COPY --from=builder /build/office365-smtp-proxy /app/office365-smtp-proxy

ENV OFFICE365_SMTP_PROXY_ADDR=":2525" \
    OFFICE365_SMTP_PROXY_METRICS=":8080"

EXPOSE 2525 8080

ENTRYPOINT [ "/app/office365-smtp-proxy" ]
