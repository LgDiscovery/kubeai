FROM golang:1.22-alpine AS Builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/inference-gateway ./inferencegateway.go

FROM alpine:3.19
WORKDIR /app
COPY --from=Builder /bin/inference-gateway /app/inference-gateway
COPY etc/inference-gateway.yaml /app/etc/inference-gateway.yaml
EXPOSE 58082
ENTRYPOINT ["/app/inference-gateway","-f","/app/etc/inference-gateway.yaml"]