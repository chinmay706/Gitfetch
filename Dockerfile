FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /gitf .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates
COPY --from=builder /gitf /usr/local/bin/gitf

RUN mkdir -p /tmp/gitf-cache

EXPOSE 7860

CMD ["gitf", "serve", "--port", "7860", "--verbose", "--cache-dir", "/tmp/gitf-cache"]
