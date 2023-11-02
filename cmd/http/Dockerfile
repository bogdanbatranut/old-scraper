# Start from the latest golang base image
FROM golang:alpine as builder

# Install Essentials
RUN apk update \
    && apk add -U --no-cache ca-certificates \
    && update-ca-certificates
# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the source from the current directory to the Working Directory inside the container
COPY ../.. .

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o http cmd/http/http.go

FROM scratch

COPY --from=builder ./app/http .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["./http"]