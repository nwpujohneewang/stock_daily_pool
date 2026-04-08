FROM golang:1.22.4 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/stock-server .

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

COPY --from=builder /bin/stock-server /app/stock-server
COPY config/config.example.yaml /app/config/config.example.yaml

ENV TZ=Asia/Shanghai

EXPOSE 8080

CMD ["/app/stock-server"]
