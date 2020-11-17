.PHONY: db migrate run test

db:
	docker run -d --rm -ti --network host -e POSTGRES_PASSWORD=secret postgres

testdb:
	docker run -d --rm -ti --name pg_test --network host -e POSTGRES_PASSWORD=secret postgres
	sleep 1.5 # wait enough time to run migrations
	migrate  -source file://migrations \
		-database postgres://postgres:secret@localhost/postgres?sslmode=disable up

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
	docker run -d --rm -ti --name pg_test --network host -e POSTGRES_PASSWORD=secret postgres
	sleep 1.5 # wait enough time to run migrations
	migrate  -source file://migrations \
		-database postgres://postgres:secret@localhost/postgres?sslmode=disable up
	go test ./...; docker stop pg_test -t 1

testv:
	docker run -d --rm -ti --name pg_test --network host -e POSTGRES_PASSWORD=secret postgres
	sleep 1.75 # wait enough time to run migrations
	migrate  -source file://migrations \
		-database postgres://postgres:secret@localhost/postgres?sslmode=disable up
	richgo test ./... -v; docker stop pg_test -t 1

test-db:
	docker run -d --rm -ti --name pg_test --network host -e POSTGRES_PASSWORD=secret postgres
	sleep 1.75 # wait enough time to run migrations
	migrate  -source file://migrations \
		-database postgres://postgres:secret@localhost/postgres?sslmode=disable up
	richgo test ./internal/db -v; docker stop pg_test -t 1

test-podcast:
	docker run -d --rm -ti --name pg_test --network host -e POSTGRES_PASSWORD=secret postgres
	sleep 1.75 # wait enough time to run migrations
	migrate  -source file://migrations \
		-database postgres://postgres:secret@localhost/postgres?sslmode=disable up
	richgo test ./internal/podcast -v; docker stop pg_test -t 1

coverage:
	docker run -d --rm -ti --name pg_test --network host -e POSTGRES_PASSWORD=secret postgres
	sleep 1.25 # wait enough time to run migrations
	migrate  -source file://migrations \
		-database postgres://postgres:secret@localhost/postgres?sslmode=disable up
	go test ./... -race -cover; docker stop pg_test -t 1

protos:
	protoc -I=/home/sam/projects/syncapod/syncapod-protos/ \
		--go_out=internal/protos/ \
		--go-grpc_out=internal/protos/ \
		/home/sam/projects/syncapod/syncapod-protos/*
