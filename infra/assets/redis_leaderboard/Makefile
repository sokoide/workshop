.PHONY: redis-up redis-down

redis-up:
	podman run -d --replace --name workshop-redis -p 6379:6379 docker.io/library/redis:alpine

redis-down:
	podman stop workshop-redis
	podman rm workshop-redis
