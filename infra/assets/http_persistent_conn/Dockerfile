FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod ./
# Copy local pb directory if it exists, otherwise it will fail. 
# Better to run 'make gen' before building or include proto files.
COPY . .
RUN go mod tidy && go build -o main .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/server.crt .
COPY --from=builder /app/server.key .
EXPOSE 8080 8443 8444 50051
CMD ["./main"]
