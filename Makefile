up:
	docker compose up --build

down:
	docker compose down --remove-orphans -v

test-perf:
	docker compose -f docker-compose.yml -f docker-compose.perf.yml run --rm perf-test
