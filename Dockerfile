FROM golang:latest AS builder

ARG APP_VERSION=0.0.0

COPY . .

RUN make msteams-presence APP_VERSION=${APP_VERSION} \
    && cp msteams-presence /usr/local/bin/msteams-presence

FROM debian:latest

RUN mkdir /app
WORKDIR /app

# Set the environment variables
ENV CLIENT_ID= \
    AUTH_TENANT=common \
    GRAPH_USER_SCOPES='user.read offline_access' \
    MQTT_USER= \
    MQTT_PASSWORD=

# create empty .env file
RUN touch /app/.env

# Update the image
RUN apt-get update && apt-get upgrade -y

# Ensure root certificates are up to date
RUN apt-get install -y ca-certificates

# clear the apt cache
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/local/bin/msteams-presence /usr/local/bin/msteams-presence

ENTRYPOINT ["/usr/local/bin/msteams-presence"]
