msteams-presence: main.go go.mod go.sum token.go device_class.go
	CGO_ENABLED=0 go build -ldflags "-X main.version=${APP_VERSION}" -o msteams-presence
	chmod +x msteams-presence
