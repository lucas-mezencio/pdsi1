.PHONY: install/tools install/go-task compose/build compose/up compose/down compose/logs compose/infra

install/tools: install/go-task

install/go-task:
	go install github.com/go-task/task/v3/cmd/task@latest
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

compose/build:
	docker compose --profile dev up -d --build

compose/up:
	docker compose --profile dev up -d

compose/down:
	docker compose --profile dev down

compose/logs:
	docker compose logs -f

compose/infra:
	docker compose up -d postgres redis
