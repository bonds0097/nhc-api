FROM golang:latest

# Get Glide
RUN curl https://glide.sh/get | sh

ENV APP_DIR /etc/nhc-api
RUN mkdir $APP_DIR

COPY init/commitments.json $APP_DIR
COPY init/organizations.json $APP_DIR
COPY init/faqs.json $APP_DIR

# Copy the local package files to the container's workspace.
COPY . /go/src/github.com/bonds0097/nhc-api
WORKDIR /go/src/github.com/bonds0097/nhc-api

# Install dependencies.
RUN glide install

RUN go install github.com/bonds0097/nhc-api
ENTRYPOINT /go/bin/nhc-api

EXPOSE 8443