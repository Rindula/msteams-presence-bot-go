module github.com/rindula/msteams-presence-bot-go

go 1.21
toolchain go1.23.7

require github.com/joho/godotenv v1.5.1 // direct

require (
	github.com/eclipse/paho.mqtt.golang v1.5.0 // direct
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/net v0.36.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
)
