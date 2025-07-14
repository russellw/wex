FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o wex .

FROM alpine:latest

RUN apk --no-cache add ca-certificates git bash curl
WORKDIR /root/

COPY --from=builder /app/wex .
COPY --from=builder /app/system_prompt.txt .

ENV WORKSPACE=/workspace
ENV OLLAMA_URL=http://192.168.0.63:11434
ENV OLLAMA_MODEL=llama3.2

VOLUME ["/workspace"]

ENTRYPOINT ["./wex"]