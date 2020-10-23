.PHONY: db migrate run test

db:
	docker run -d --rm -ti --network host -e POSTGRES_PASSWORD=secret postgres

migrate:
	migrate -source file://migrations \
		-database postgres://postgres:secret@localhost/postgres?sslmode=disable up

run:
	go run ./cmd/main.go

build:
	go build ./cmd/main.go

clean:
	rm main

test:
	go test ./... -race
