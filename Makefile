# support for .env file
ifneq (,$(wildcard ./.env))
    include .env
    export
endif 

.PHONY: db migrate run test testv test-db coverage protos

db:
	docker run -d --rm -ti --network host -e POSTGRES_PASSWORD=secret postgres

migrate:
	migrate -source file://migrations \
		-database postgres://postgres:secret@localhost/postgres?sslmode=disable up

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
