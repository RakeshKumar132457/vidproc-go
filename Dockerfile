FROM golang:1.23.4-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY go.mod ./
COPY go.sum* ./

RUN go mod download

COPY . . 

RUN CGO_ENABLED=1 GOOS=linux go build -o /app/server ./cmd/server

FROM alpine:latest

RUN apk add --no-cache ffmpeg sqlite

WORKDIR /app

COPY --from=builder /app/server .

RUN mkdir -p /app/data/videos /app/data/db

EXPOSE 8080

CMD ["./server"]
