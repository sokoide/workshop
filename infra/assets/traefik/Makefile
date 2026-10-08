.PHONY: up down logs clean

up:
	podman compose up -d

down:
	podman compose down

logs:
	podman compose logs -f --tail=100

clean:
	podman compose down -v --remove-orphans
