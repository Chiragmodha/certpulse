# Build stage
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod tidy
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o certpulse-api .

# Run stage
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/certpulse-api .
EXPOSE 5000
CMD ["./certpulse-api"]
