FROM golang:1.22@sha256:682162dd646e9de40bce24104c291f2f8830b9c32a3ac21eec3bef9b8418dfbd AS builder

COPY . /build

RUN cd /build && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' ./cmd/graph-smtpd

FROM gcr.io/distroless/base-debian12:nonroot@sha256:e5260be292def77bc70d03003f788f3d32c0796972ea1412d72cc0c843ab139a

COPY --from=builder /build/graph-smtpd /app/graph-smtpd

ENV SMTPD_ADDR=":2525"

EXPOSE 2525

ENTRYPOINT [ "/app/graph-smtpd" ]
