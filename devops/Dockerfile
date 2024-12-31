FROM golang:1.23.4-alpine AS base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

FROM base AS dev

RUN go install github.com/air-verse/air@latest

CMD ["air", "-c", ".air.toml"]
