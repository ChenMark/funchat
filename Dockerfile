# FunChat 后端服务

FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 go build -o server ./cmd/server/

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/server .
COPY backend/.env.example .env

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s \
    CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["./server"]
