.PHONY: mq-up mq-down

mq-up:
	podman run -d --replace --name workshop-mq -p 5672:5672 -p 15672:15672 docker.io/library/rabbitmq:3-management

mq-down:
	podman stop workshop-mq
	podman rm workshop-mq
