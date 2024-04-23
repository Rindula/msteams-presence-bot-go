FROM golang:latest

RUN mkdir /app
WORKDIR /app

RUN CGO_ENABLED=0 go build -o /usr/local/bin/msteams-presence

# Set the file permissions
RUN chmod +x /usr/local/bin/msteams-presence

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

ENTRYPOINT ["/usr/local/bin/msteams-presence"]
