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

RUN ls -lah

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cron cmd/cron/cron.go

FROM ubuntu

RUN apt-get update && apt-get -y install cron

RUN ls -lah

COPY --from=builder ./app/cron .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/


#
## Copy cron-job file to the cron.d directory
#COPY --from=builder ./app/cron/cron-job /etc/cron.d/cron-job

# Give execution rights on the cron job
RUN #chmod 0644 /etc/cron.d/cron-job

# Apply cron job
RUN #crontab /etc/cron.d/cron-job


ENTRYPOINT ["./cron"]