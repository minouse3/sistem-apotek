# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/sistem-apotek .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Jakarta
WORKDIR /app

COPY --from=builder /out/sistem-apotek /usr/local/bin/sistem-apotek

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/sistem-apotek"]
