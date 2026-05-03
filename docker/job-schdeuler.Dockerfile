FROM golang:1.22-alpine AS Builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/job-scheduler ./jobscheduler.go

FROM alpine:3.19
WORKDIR /app
COPY --from=Builder /bin/job-scheduler /app/job-scheduler
COPY etc/job-scheduler.yaml /app/etc/job-scheduler.yaml
EXPOSE 58081
ENTRYPOINT ["/app/job-scheduler","-f","/app/etc/job-scheduler.yaml"]