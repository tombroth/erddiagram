FROM golang:1.20 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o go-db-visualizer ./cmd/server/main.go

FROM gcr.io/distroless/base

COPY --from=builder /app/go-db-visualizer /go-db-visualizer

EXPOSE 8080

ENTRYPOINT ["/go-db-visualizer"]