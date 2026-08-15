FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

COPY . .

RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o promo-engine ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/promo-engine .

EXPOSE 8080

CMD ["./promo-engine"]
