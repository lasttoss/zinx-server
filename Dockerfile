FROM golang:1.24.5-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main /app/main
COPY --from=builder /app/conf /app/conf

# Copy the .env file
#COPY .env .
COPY config.yaml .

# Command to run the executable
CMD ["/app/main"]