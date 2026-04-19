# Dockerfile pro TMC core: 

# --- build stage ---
    FROM golang:1.22 AS builder
    RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates && rm -rf /var/lib/apt/lists/*
    ENV GODEBUG=x509ignoreCN=0
    
    # Modulový kořen
    WORKDIR /app
    
    # Definice modulů
    COPY tmc-core/go.mod tmc-core/go.sum ./
    RUN go mod download
    
    # Kopírovat zdrojové kódy (včetně src/data)
    COPY tmc-core/src ./src
    
    # Kopírovat statická data (pro výstup)
    COPY tmc-core/static_data ./static_data
    
    # Kopírovat pro statické linkování
    ENV CGO_ENABLED=0
    ENV GOOS=linux
    
    # Spuštění
    RUN go build -o tmc-core ./src/main.go
    
    # --- runtime stage ---
    FROM debian:bookworm-slim
    RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates && rm -rf /var/lib/apt/lists/*
    ENV GODEBUG=x509ignoreCN=0
    
    WORKDIR /app
    COPY --from=builder /app/tmc-core ./tmc-core
    COPY --from=builder /app/static_data ./static_data
    
    ENV TMC_TMC_DIR=/app/static_data/tmc
    ENV TMC_PREDEFINED_DIR=/app/static_data/predefined_positions
    
    EXPOSE 9200
    ENTRYPOINT ["./tmc-core"]
    