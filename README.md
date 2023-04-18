# GitHub sourcer

This repository contains a work-together exercise code for the following scenario:

## assumptions

This is an example serving as a background to the concepts outlined in the problem description. Because of that, this program is intended to run against one Kafka broker.

## make commands

- `kafka-clean`: stop running Kafka infrastructure containers, remove them, and clean up volumes
- `kafka-up`: start Kafka infrastructure with Docker Compose

- `kafka-consumer-rollups`: start Kafka console consumer for the `rollups` topic
- `kafka-topics-create`: create all topics required for this work-together exercise
- `kafka-topics-describe`: describe existing Kafka topics
- `kafka-topics-list`: list existing Kafka topics

- `run-consumer`: runs the HTTP server consumer command
- `run-producer`: runs the producer command

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

3. In another terminal window: Run the main program. Assumes that you have the Go programming language configured correctly.

```sh
make run-producer
```
