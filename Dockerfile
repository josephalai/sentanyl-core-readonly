FROM golang:1.25-alpine AS builder
WORKDIR /workspace
COPY go.work go.work.sum ./
COPY pkg/ pkg/
COPY core-service/ core-service/
COPY lms-service/ lms-service/
COPY marketing-service/ marketing-service/
COPY video-service/ video-service/
COPY coaching-service/ coaching-service/
RUN cd core-service && go build -o /app/core-service ./cmd

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/core-service .
EXPOSE 8081
CMD ["./core-service"]
