FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o wex .

FROM alpine:latest

# Install basic tools and essential programming languages
RUN apk --no-cache add ca-certificates git bash curl wget build-base \
    python3 python3-dev py3-pip \
    nodejs npm \
    && ln -sf python3 /usr/bin/python

# Install Rust (lightweight installation)
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain stable --profile minimal

# Set up environment variables
ENV PATH="/root/.cargo/bin:$PATH"

WORKDIR /root/

COPY --from=builder /app/wex .
COPY --from=builder /app/system_prompt.txt .

ENV WORKSPACE=/workspace
ENV OLLAMA_URL=http://192.168.0.63:11434

VOLUME ["/workspace"]

ENTRYPOINT ["./wex"]