FROM ubuntu:latest

RUN mkdir /app
ADD msteams-presence /app/msteams-presence
RUN chmod +x /app/msteams-presence

ENV CLIENT_ID= \
    TENANT_ID= \
    AUTH_TENANT=common \
    GRAPH_USER_SCOPES='user.read offline_access' \
    MQTT_USER= \
    MQTT_PASSWORD=

# create empty .env file
RUN touch /app/.env

WORKDIR /app

ENTRYPOINT ["/app/msteams-presence"]
