# Intentionally uses an old base image (for Trivy scan demo)
# In real projects, always use the latest stable version
FROM alpine:3.12

RUN apk add --no-cache openssl

WORKDIR /app
COPY go.mod go.sum ./
COPY main.go .

CMD ["echo", "Run 'trivy image' to scan this image for vulnerabilities"]
