# GitHub sourcer

This repository contains a work-together exercise code for the following scenario:

## assumptions

This is an example serving as a background to the concepts outlined in the problem description. Because of that, this program is intended to run against one Kafka broker.

## make commands

- `kafka-clean`: stop running Kafka infrastructure containers, remove them, and clean up volumes
- `kafka-up`: start Kafka infrastructure with Docker Compose

- `lb-clean`: stop running Envoy load balancer container, remove it
- `lb-up`: start Envoy load balancer with Docker Compose

- `programs-clean`: stop running producer and consumer program containers, and remove them
- `programs-up`: start producer and consumer containers with Docker Compose

- `kafka-consumer-rollups`: start Kafka console consumer for the `rollups` topic
- `kafka-topics-create`: create all topics required for this work-together exercise
- `kafka-topics-describe`: describe existing Kafka topics
- `kafka-topics-list`: list existing Kafka topics

- `run-consumer-1`: runs the HTTP server consumer command with port 8081
- `run-consumer-2`: runs the HTTP server consumer command with port 8082
- `run-producer`: runs the producer command

Build targets:

- `build.consumer`: builds the consumer
- `build.producer`: builds the producer
- `docker.build.consumer`: builds the consumer Docker image
- `docker.build.producer`: builds the producer Docker image

## dependencies

- Docker
- Docker Compose
- make
- Go 1.19+ programming language

## running

1. Start Kafka in _Terminal 1_. This command will run in foreground. Give it about a minute to settle if you're running from scratch.

```sh
make kafka-up
```

2. In another terminal window: create Kafka topics. You can skip this between restarts if you haven't run `kafka-clean` command.

```sh
make kafka-topics-create
```

3. Build Docker images:

```sh
make docker.build.consumer
make docker.build.producer
```

4. Run programs:

```sh
make programs-up
```
