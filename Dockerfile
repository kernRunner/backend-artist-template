# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o server .

FROM alpine

WORKDIR /app

# Correct path here:
COPY --from=build /app/server /app/server

RUN chmod +x /app/server

EXPOSE 8080

CMD ["/app/server"]
