include .env 


build:
	@go build -o bin/potoc ./cmd/main.go

run: build 
	@./bin/potoc


clean:
	@rm -rf bin/


tidy:
	@go mod tidy -v 



buildWindows:
	@go build -o bin/potoc.exe ./cmd/main.go



runWindows: buildWindows
	@./bin/potoc.exe



migration:
	@migrate create -ext sql -dir ./internal/migrations -seq $(name)


migrate-up:
	@migrate -path ./internal/migrations -database $(POSTGRES_URL) up

migration-down:
	@migrate -database $(POSTGRESQL_URL) -path ./internal/migrations down

lint:
	@golangci-lint run --timeout 10m

migration-force:
	@migrate -database $(POSTGRESQL_URL) -path ./internal/migrations force $(version)

migration-version:
	@migrate -database $(POSTGRESQL_URL) -path ./internal/migrations version




# if proto is up to date use make -B 
proto:
	@protoc -I ./pkg/proto ./pkg/proto/data_transfer.proto --go_out=./pkg/proto/ --go_opt=paths=source_relative --go-grpc_out=./pkg/proto/ --go-grpc_opt=paths=source_relative
	@echo "protoc done"


installGrpc:
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

