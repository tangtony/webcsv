###########################################################
## BUILD STAGE
FROM golang:alpine as build

# Enable Go modules for Go 1.11
ARG GO111MODULE=on

# Install build dependencies
RUN apk add --no-cache build-base git

# Set the working directory
WORKDIR /go/src

# Fetch dependencies
COPY ./go.mod ./go.sum ./
RUN go mod download

# Build the application
COPY ./ ./
RUN go build -o /app .

###########################################################
## IMAGE STAGE
FROM alpine:latest as image

# Copy the compiled application from the build stage
COPY --from=build /app .

# Expose the default port of the HTTP server
EXPOSE 8080

# Start the application
CMD ["sh", "-c", "./app"]
