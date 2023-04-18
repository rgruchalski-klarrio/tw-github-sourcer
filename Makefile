.PHONY: kafka-clean kafka-up
kafka-clean:
	docker compose -f compose-kafka.yaml stop
	docker compose -f compose-kafka.yaml rm
	docker volume rm tw-github-sourcer_kafka_data tw-github-sourcer_zookeeper_data

kafka-up:
	docker compose -f compose-kafka.yaml up

.PHONY: kafka-consumer-github-data
kafka-consumer-github-data:
	docker exec -ti tw-github-sourcer-kafka-1 /bin/bash -c 'kafka-console-consumer.sh --bootstrap-server=kafka:9092 --group github-data-makefile-consumer --from-beginning --topic github-data'

.PHONY: kafka-topics-create kafka-topics-describe kafka-topics-list
kafka-topics-create:
	docker exec -ti tw-github-sourcer-kafka-1 /bin/bash -c 'kafka-topics.sh --bootstrap-server=kafka:9092 --create --topic github-data'

kafka-topics-describe:
	docker exec -ti tw-github-sourcer-kafka-1 /bin/bash -c 'kafka-topics.sh --bootstrap-server=kafka:9092 --describe --topic github-data'

kafka-topics-list:
	docker exec -ti tw-github-sourcer-kafka-1 /bin/bash -c 'kafka-topics.sh --bootstrap-server=kafka:9092 --list'
