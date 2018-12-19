FROM golang:1.11-alpine as build
WORKDIR /go/src/github.com/go-ocf/coap-gateway
RUN apk add --no-cache curl git && \
	curl -SL -o /usr/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && \
    chmod +x /usr/bin/dep
COPY ./ ./
RUN dep ensure -v --vendor-only

ENV CGO_ENABLED 0
ENV GOOS linux

RUN go build -a -installsuffix nocgo -o coap-gateway-service ./cmd/coap-gateway-service

FROM scratch
WORKDIR /root/
COPY --from=build /go/src/github.com/go-ocf/coap-gateway/coap-gateway-service .
CMD ["./coap-gateway-service"]
