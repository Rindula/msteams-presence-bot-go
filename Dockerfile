FROM debian:latest

RUN mkdir /app
WORKDIR /app

# Copy the binary file to the container
COPY msteams-presence /usr/local/bin/msteams-presence

# Set the file permissions
RUN chmod +x /usr/local/bin/msteams-presence

# Set the environment variables
ENV CLIENT_ID= \
    TENANT_ID= \
    AUTH_TENANT=common \
    GRAPH_USER_SCOPES='user.read offline_access' \
    MQTT_USER= \
    MQTT_PASSWORD=

# create empty .env file
RUN touch /app/.env

ENTRYPOINT ["/usr/local/bin/msteams-presence"]
