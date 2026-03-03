FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/donrat-bot ./cmd/bot

FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/donrat-bot /app/donrat-bot

ENTRYPOINT ["/app/donrat-bot"]
