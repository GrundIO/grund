FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /user-service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /user-service /user-service
EXPOSE 8080
CMD ["/user-service"]
