FROM golang:alpine
WORKDIR /
ADD . .
RUN go build rtsp-proxy.go

FROM alpine:latest
COPY --from=0 /rtsp-proxy .
ENTRYPOINT ["/rtsp-proxy"]
