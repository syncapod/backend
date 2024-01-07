# support for .env file
ifneq (,$(wildcard ./.env))
    include .env
    export
endif 

.PHONY: db migrate run test testv test-db coverage protos

db-start:
	docker run --name syncapod-db -d --rm -ti --network host -e POSTGRES_PASSWORD=secret postgres

db-rm:
	docker rm -f syncapod-db

wait-for-db:
	@until docker exec syncapod-db pg_isready -d "postgresql://postgres:secret@localhost" >/dev/null 2>&1; do \
		echo "Waiting for PostgreSQL database..."; \
		sleep 1; \
    done

migrate:
	migrate -source file://db/migrations \
		-database postgres://postgres:secret@localhost/postgres?sslmode=disable up

migrate-down:
	migrate -source file://db/migrations \
		-database postgres://postgres:secret@localhost/postgres?sslmode=disable down

db: db-start wait-for-db migrate

run:
	go run ./cmd/main.go

build:
	go build -o syncapod ./cmd/main.go

clean:
	rm ./syncapod -f
	go clean -testcache

test:
	go test ./internal/...

testv:
	go test ./internal/... -v

coverage:
	go test ./... -cover

sync:
	rsync -a --exclude config.json --exclude .env . root@syncapod.com:/root/syncapod

protos:
	protoc -I $(PROTO_DIR) \
		-I ${GOOGLE_API_PROTO_DIR} \
		--go_out=internal/gen/ \
		--twirp_out=internal/gen/ \
		$(PROTO_DIR)*

grpc-gateway:
	protoc -I $(PROTO_DIR) \
		-I ${GOOGLE_API_PROTO_DIR} \
		--grpc-gateway_out=internal/protos \
		--grpc-gateway_opt logtostderr=true \
		--grpc-gateway_opt generate_unbound_methods=true \
		$(PROTO_DIR)*

sqlc_generate:
	sqlc generate

sqlc_vet:
	sqlc vet

sqlc_verify:
	sqlc verify
