FROM golang:1.20 as builder

WORKDIR /go/src/
COPY . .

RUN export GOPROXY=https://goproxy.cn && \
    go build -o pizza-crd-webhook cmd/pizza-crd-webhook/main.go

FROM centos:7

COPY --from=builder /go/src/pizza-crd-webhook /pizza-crd-webhook

EXPOSE 8081

CMD ["/pizza-crd-webhook"]