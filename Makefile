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

$(TARGET): $(SOURCES) Dockerfile .envrc
	GOOS=linux GOARCH=amd64 go build -o ${TARGET}
	docker build -t ${TARGET}:${TAG} -f Dockerfile .

test:
	PURITY_DB_HOST="localhost" go test ./...

down: stop
stop:
	docker-compose down

clean:
	rm ${NAME}
	docker stop purity-pg
