.PHONY: install/tools install/go-task compose/build compose/up compose/down compose/logs compose/infra

install/tools: install/go-task

install/go-task:
	go install github.com/go-task/task/v3/cmd/task@latest

compose/build:
	docker compose up -d --build careconnect

compose/up:
	docker compose up -d

compose/down:
	docker compose down

compose/logs:
	docker compose logs -f

compose/infra:
	docker compose up -d postgres redis