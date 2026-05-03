FROM golang:1.22-alpine AS Builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api-gateway ./gateway.go

FROM alpine:3.19
WORKDIR /app
COPY --from=Builder /bin/api-gateway /app/api-gateway
COPY etc/gateway.yaml /app/etc/gateway.yaml
EXPOSE 8080
ENTRYPOINT ["/app/api-gateway", "-f", "/app/etc/gateway.yaml"]