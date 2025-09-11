# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache ca-certificates
WORKDIR /src

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/vitalink .

# Runtime stage (ultra minimal)
FROM scratch
WORKDIR /app

# CA certs for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# App binary
COPY --from=builder /out/vitalink /app/vitalink

# Static/templates needed at runtime
COPY templates/ /app/templates/
COPY public/ /app/public/

ENV ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app/vitalink"]
