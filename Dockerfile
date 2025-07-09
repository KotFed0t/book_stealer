FROM golang:1.24.0-alpine as builder

WORKDIR /build
COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux

RUN go mod download
RUN go build -o ./book_stealer ./cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /build/book_stealer /app/
COPY --from=builder /build/.env /app/

CMD ["./book_stealer"]