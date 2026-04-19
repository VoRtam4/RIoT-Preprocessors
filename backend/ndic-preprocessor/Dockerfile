# Převzato a upraveno z projektu RIOT:

# Use an official Go runtime as a parent image
FROM golang:1.22-alpine
# Set the working directory inside the container
WORKDIR /app
# Copy the go.mod file, go.sum file, and src directory from ndic-preprocessor
COPY ./ndic-preprocessor/go.mod ./ndic-preprocessor/go.sum /app/ndic-preprocessor/
COPY ./ndic-preprocessor/src /app/ndic-preprocessor/src
# Copy the go.mod file, go.sum file, and src directory from commons
COPY ./commons/go.mod ./commons/go.sum /app/commons/
COPY ./commons/src /app/commons/src
# Change working directory to where the Go app's main module is located
WORKDIR /app/ndic-preprocessor
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod tidy
RUN go mod download
# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o RIoT-ndic-preprocessor ./src
# Run the executable
CMD ["/app/ndic-preprocessor/RIoT-ndic-preprocessor"]
