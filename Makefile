SOURCES := $(shell find . -type f -name '*.go')
TARGET = purity-vision
.DEFAULT_GOAL: $(TARGET)
TAG = latest

.PHONY: docker-run run test docker-stop clean down

docker-run: $(TARGET)
	docker compose up --detach

run: $(TARGET)
	./scripts/start-db.sh
	./${TARGET}

build:
	docker build -t purity-vision-api .

$(TARGET): $(SOURCES) Dockerfile .envrc
	GOOS=linux GOARCH=amd64 go build -o ${TARGET}
	docker build -t ${TARGET}:${TAG} -f Dockerfile .

local:
	PURITY_DB_HOST="localhost" go run main.go

test:
	PURITY_DB_HOST="localhost" go test ./...

down: stop
stop:
	docker-compose ./.env down

clean:
	rm ${NAME}
	docker stop purity-pg
