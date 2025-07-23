FROM golang:1.24.0-alpine as builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux

RUN go build -o ./book_stealer ./cmd/main.go

FROM alpine:latest

WORKDIR /app

# добавление базы mime типов
RUN apk add --no-cache mailcap

COPY --from=builder /build/book_stealer /app/
COPY --from=builder /build/.env /app/
COPY --from=builder /build/migrations /app/migrations
COPY --from=builder /build/googleCredentials.json /app/

CMD ["./book_stealer"]