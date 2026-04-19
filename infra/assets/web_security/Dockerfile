FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN cd victim && go build -o /app/victim .
RUN cd attacker && go build -o /app/attacker .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/victim .
COPY --from=builder /app/attacker .
EXPOSE 8080 8081
CMD ["sh", "-c", "./victim & ./attacker & wait"]
