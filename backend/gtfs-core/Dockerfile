# Dockerfile pro GTFS core: 

# --- build stage ---
FROM golang:1.22 as builder

RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates

ENV GODEBUG=x509ignoreCN=0

# Modulový kořen
WORKDIR /app

COPY . .

WORKDIR /app/gtfs-core

# Definice modulů
COPY ../commons/go.mod ../commons/go.sum /app/commons/
COPY ../commons/src /app/commons/src

RUN go mod tidy

# Spuštění
RUN go build -o gtfs-core main.go trip_keys.go stops_handler.go departure_times_handler.go resolve_stop_id_handler.go stable_trip_key_handler.go resolve_trip_id_handler.go resolve_trip_details_handler.go resolve_trip_hash_handler.go resolve_hash_id_handler.go load_static_data.go routes_handler.go update_gtfs_data.go

# --- runtime stage ---
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates

ENV GODEBUG=x509ignoreCN=0

WORKDIR /app

COPY --from=builder /app/gtfs-core .
COPY --from=builder /app/gtfs-core/static_data ./static_data
COPY --from=builder /app/gtfs-core/trip_key_map.csv .

EXPOSE 9100
CMD ["./gtfs-core"]
