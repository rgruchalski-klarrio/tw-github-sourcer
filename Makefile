CURRENT_DIR=$(dir $(realpath $(firstword $(MAKEFILE_LIST))))
CURRENT_DIR_NAME=$(shell basename $(dir ${CURRENT_DIR}))

KAFKA_TOPIC_ROLLUPS=rollups

.PHONY: kafka-clean kafka-up
kafka-clean:
	docker compose -f ${CURRENT_DIR}/compose-kafka.yaml stop
	docker compose -f ${CURRENT_DIR}/compose-kafka.yaml rm
	docker volume rm ${CURRENT_DIR_NAME}_kafka_data ${CURRENT_DIR_NAME}_zookeeper_data

kafka-up:
	docker compose -f ${CURRENT_DIR}/compose-kafka.yaml up

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
