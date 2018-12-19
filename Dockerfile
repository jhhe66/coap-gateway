FROM golang:latest as builder
WORKDIR /go/src/github.com/go-ocf/coap-gateway
RUN wget --no-check-certificate https://raw.githubusercontent.com/golang/dep/master/install.sh
RUN sh install.sh
COPY ./ ./
RUN dep ensure
RUN dep status
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o coap-gateway-service ./cmd/coap-gateway-service

FROM scratch
WORKDIR /root/
COPY --from=builder /go/src/github.com/go-ocf/coap-gateway/coap-gateway-service .
CMD ["./coap-gateway-service"]
