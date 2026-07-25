.PHONY: install/tools install/go-task compose/build compose/up compose/down compose/logs compose/infra

install/tools: install/go-task

install/go-task:
	go install github.com/go-task/task/v3/cmd/task@latest
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

compose/build:
	docker compose -f compose.yml -f compose.dev.yml --profile dev up -d --build

compose/up:
	docker compose -f compose.yml -f compose.dev.yml --profile dev up -d

compose/down:
	docker compose down -v --remove-orphans; docker kill $(docker ps -q); docker rm $(docker ps -aq); docker network prune

compose/logs:
	docker compose -f compose.yml --profile dev logs -f

compose/infra:
	docker compose up -d postgres redis
