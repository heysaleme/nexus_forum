.PHONY: run test build clean

run:
	go run cmd/api/main.go

test:
	go test ./... -v

build:
	go build -o server cmd/api/main.go

clean:
	rm -f server nexus_forum.db
