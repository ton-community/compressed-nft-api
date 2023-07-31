# syntax=docker/dockerfile:1

# Build the application from source
FROM golang:1.20 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /server cmd/server/main.go

FROM debian:12-slim AS build-release-stage

WORKDIR /

COPY --from=build-stage /server /server

ENV PORT 8080
ENV DATA_DIR /apidata

RUN mkdir -p ${DATA_DIR}

ENTRYPOINT ["/server"]