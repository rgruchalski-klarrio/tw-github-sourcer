CURRENT_DIR=$(dir $(realpath $(firstword $(MAKEFILE_LIST))))
CURRENT_DIR_NAME=$(shell basename $(dir ${CURRENT_DIR}))

GOARCH      ?= amd64
GOOS        ?= linux

KAFKA_TOPIC_ROLLUPS=rollups

build.consumer:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(CURRENT_DIR_NAME)-linux-amd64 ./cmd/consumer/command.go
build.producer:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(CURRENT_DIR_NAME)-linux-amd64 ./cmd/producer/main.go

docker.build.consumer:
	docker build --build-arg GOOS=$(GOOS) --build-arg GOARCH=$(GOARCH) --build-arg BUILD_TYPE=consumer -t localhost/$(CURRENT_DIR_NAME)-consumer:latest -f Dockerfile .
docker.build.producer:
	docker build --build-arg GOOS=$(GOOS) --build-arg GOARCH=$(GOARCH) --build-arg BUILD_TYPE=producer -t localhost/$(CURRENT_DIR_NAME)-producer:latest -f Dockerfile .

.PHONY: kafka-clean kafka-up programs-clean programs-up
kafka-clean:
	docker compose -f ${CURRENT_DIR}/compose.yaml stop kafka zookeeper
	docker compose -f ${CURRENT_DIR}/compose.yaml rm kafka zookeeper
	docker volume rm ${CURRENT_DIR_NAME}_kafka_data ${CURRENT_DIR_NAME}_zookeeper_data

kafka-up:
	docker compose -f ${CURRENT_DIR}/compose.yaml up kafka zookeeper

programs-clean:
	docker compose -f ${CURRENT_DIR}/compose.yaml stop producer consumer-1 consumer-2
	docker compose -f ${CURRENT_DIR}/compose.yaml rm producer consumer-1 consumer-2

programs-up:
	docker compose -f ${CURRENT_DIR}/compose.yaml up producer consumer-1 consumer-2

.PHONY: kafka-consumer-rollups
kafka-consumer-rollups:
	docker exec -ti ${CURRENT_DIR_NAME}-kafka-1 \
		/bin/bash -c 'kafka-console-consumer.sh --bootstrap-server=kafka:9092 --group ${CURRENT_DIR_NAME}-consumer --from-beginning --topic ${KAFKA_TOPIC_ROLLUPS}'

.PHONY: kafka-topics-create kafka-topics-describe kafka-topics-list
kafka-topics-create:
	docker exec -ti ${CURRENT_DIR_NAME}-kafka-1 /bin/bash -c 'kafka-topics.sh --bootstrap-server=kafka:9092 --create --topic ${KAFKA_TOPIC_ROLLUPS}'

kafka-topics-describe:
	docker exec -ti ${CURRENT_DIR_NAME}-kafka-1 /bin/bash -c 'kafka-topics.sh --bootstrap-server=kafka:9092 --describe --topic ${KAFKA_TOPIC_ROLLUPS}'

kafka-topics-list:
	docker exec -ti ${CURRENT_DIR_NAME}-kafka-1 /bin/bash -c 'kafka-topics.sh --bootstrap-server=kafka:9092 --list'

.PHONY: run-consumer-1 run-consumer-2 run-producer
run-consumer-1:
	go run cmd/consumer/command.go -server-bind-host-port=:8081 -consumer-group=''

run-consumer-2:
	go run cmd/consumer/command.go -server-bind-host-port=:8082 -consumer-group=''

run-producer:
	go run cmd/producer/main.go
