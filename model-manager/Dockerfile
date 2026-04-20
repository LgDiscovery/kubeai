FROM golang:1.22-alpine AS Builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/model-manager ./modelmanager.go

FROM alpine:3.19
WORKDIR /app
COPY --from=Builder /bin/model-manager /app/model-manager
COPY etc/model-manager.yaml /app/etc/model-manager.yaml
EXPOSE 58080
ENTRYPOINT ["/app/model-manager", "-f", "/app/etc/model-manager.yaml"]
