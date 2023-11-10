.PHONY: build

build: main.go token.go
	GOOS=linux GOARCH=amd64 go build
	GOOS=windows GOARCH=amd64 go build
