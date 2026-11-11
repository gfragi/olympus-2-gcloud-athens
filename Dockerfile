# Stage 1: Build the WebAssembly game and minimal Cloud Run server
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# 1. Compile single main.go into WebAssembly binary
RUN mkdir -p public && \
    GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o public/game.wasm .

# 2. Copy wasm_exec.js into public directory
RUN if [ -f "$(go env GOROOT)/lib/wasm/wasm_exec.js" ]; then \
        cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" public/; \
    elif [ -f "$(go env GOROOT)/misc/wasm/wasm_exec.js" ]; then \
        cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" public/; \
    fi

# 3. Build minimal static HTTP server for Cloud Run
RUN printf 'package main\n\nimport (\n\t"log"\n\t"net/http"\n\t"os"\n)\n\nfunc main() {\n\tport := os.Getenv("PORT")\n\tif port == "" {\n\t\tport = "8080"\n\t}\n\thttp.Handle("/", http.FileServer(http.Dir("/app/public")))\n\thttp.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {\n\t\tw.WriteHeader(http.StatusOK)\n\t\t_, _ = w.Write([]byte("ok"))\n\t})\n\tlog.Printf("Olympus 2.0 server running on :%%s", port)\n\tif err := http.ListenAndServe(":"+port, nil); err != nil {\n\t\tlog.Fatal(err)\n\t}\n}\n' > /app/server_embed.go && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server /app/server_embed.go

# Stage 2: Minimal runtime container
FROM alpine:3.20

WORKDIR /app
COPY --from=builder /server /server
COPY --from=builder /app/public /app/public

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/server"]
