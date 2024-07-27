msteams-presence: main.go go.mod go.sum token.go
	CGO_ENABLED=0 go build -o msteams-presence
	chmod +x msteams-presence
