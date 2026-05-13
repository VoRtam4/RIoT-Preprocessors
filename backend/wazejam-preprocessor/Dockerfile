# @file Dockerfile
# @brief Docker image pro WazeJam preprocesor.
#
# @author Dominik Vondruška
# @author Vojtěch Hubáček
# @ingroup riot_wazejam_preprocessor
#
# @par Autorský podíl
# - Dominik Vondruška: původní Docker základ preprocesoru.
# - Vojtěch Hubáček: úprava build kontextu a runtime podoby pro samostatný repozitář preprocesorů.
# Use an official Go runtime as a parent image
FROM golang:1.22-alpine
# Set the working directory inside the container
WORKDIR /app
# Copy the go.mod file, go.sum file, and src directory from wazejam-preprocessor
COPY ./wazejam-preprocessor/go.mod ./wazejam-preprocessor/go.sum /app/wazejam-preprocessor/
COPY ./wazejam-preprocessor/src /app/wazejam-preprocessor/src
# Copy the go.mod file, go.sum file, and src directory from commons
COPY ./commons/go.mod ./commons/go.sum /app/commons/
COPY ./commons/src /app/commons/src
# Change working directory to where the Go app's main module is located
WORKDIR /app/wazejam-preprocessor
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod tidy
RUN go mod download
# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o RIoT-wazejam-preprocessor ./src
# Run the executable
CMD ["/app/wazejam-preprocessor/RIoT-wazejam-preprocessor"]
