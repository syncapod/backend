FROM golang:alpine AS build
RUN mkdir /app
WORKDIR /app
# Dependency management
COPY go.* /app/
RUN go mod download
COPY . /app
RUN go build -o /app/main cmd/main.go

FROM alpine:latest AS prod
WORKDIR /syncapod
COPY --from=0 /app/main /syncapod
COPY ./config.json /syncapod
COPY ./migrations /syncapod/migrations
COPY ./templates /syncapod/templates
CMD ["/syncapod/main"]
