# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:latest

# Generate RSA keys for token signing.

# Get Glide
RUN curl https://glide.sh/get | sh

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/bonds0097/nhc-api
WORKDIR /go/src/github.com/bonds0097/nhc-api

# Install dependencies.
RUN glide install

# Build the outyet command inside the container.
# (You may fetch or manage dependencies here,
# either manually or with a tool like "godep".)
RUN go install github.com/bonds0097/nhc-api

# Run the outyet command by default when the container starts.
ENTRYPOINT /go/bin/nhc-api

# Document that the service listens on port 8080.
EXPOSE 8443