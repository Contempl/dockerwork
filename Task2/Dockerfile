FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -o server .

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/server .

CMD ["./server"]