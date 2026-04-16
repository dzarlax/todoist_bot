FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /todoist-bot ./cmd/todoist-bot

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /todoist-bot /todoist-bot

EXPOSE 3000
ENTRYPOINT ["/todoist-bot"]
