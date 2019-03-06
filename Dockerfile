FROM golang:1.11 AS builder
RUN go version
WORKDIR /src
COPY ./go.mod ./go.sum ./
RUN go mod download

COPY ./ ./
RUN CGO_ENABLED=0 go build \
    -installsuffix 'static' \
    -o /app .

FROM alpine AS final
COPY --from=builder /app /app

EXPOSE 80
ENTRYPOINT ["./app"]