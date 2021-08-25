# Accept the Go version for the image to be set as a build argument.
# Default to Go 1.12
ARG GO_VERSION=1.12

# First stage: code and flags.
# This stage is used for running the tests with a valid Go environment, with
# `/vendor` directory support enabled.
FROM golang:${GO_VERSION} AS code

# Set the environment variables for the commands passed to the stage when using
# `docker build --target code`. Leave CGO available for the race detector.
ENV GOFLAGS=-mod=vendor

# Set the working directory outside $GOPATH to enable the support for modules.
WORKDIR /terraform-ui

# Import the code from the context.
COPY service ./service

# Second stage: build the executable
FROM golang:${GO_VERSION}-alpine AS builder

# Create the user and group files that will be used in the running container to
# run the process an unprivileged user.
RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534:' > /user/group

# Import the Certificate-Authority certificates for the app to be able to send
# requests to HTTPS endpoints.
RUN apk add --no-cache ca-certificates

# Accept the version of the app that will be injected into the compiled
# executable.
ARG APP_VERSION=undefined

# Set the environment variables for the build command.
ENV CGO_ENABLED=0 GOFLAGS=-mod=vendor

#create storage folder for the uploaded plans
RUN mkdir /plans

# Set the working directory outside $GOPATH to enable the support for modules.
WORKDIR /terraform-ui/

# Import the code from the first stage.
COPY --from=code /terraform-ui ./

# Build the executable to `/app`. Mark the build as statically linked and
# inject the version as a global variable.
RUN cd service && \
    go build \
    -installsuffix 'static' \
    -ldflags "-X main.Version=${APP_VERSION}" \
    -o /app \
    ./main.go

# Final stage: the running container
FROM scratch AS final

ARG GO_VERSION=undefined
ARG APP_VERSION=undefined
ARG GIT_COMMIT=undefined
ARG BUILD_DATE=undefined

LABEL org.label-schema.build-date="$BUILD_DATE"
LABEL org.label-schema.name="terraform-ui"
LABEL org.label-schema.description="Simple ui to show terraform plan diffs."
LABEL org.label-schema.vcs-url=""
LABEL org.label-schema.vcs-ref="$GIT_COMMIT"
LABEL org.label-schema.vendor="fr123k"
LABEL org.label-schema.version="$APP_VERSION"
LABEL org.label-schema.schema-version="1.0"
LABEL go-version="$GO_VERSION"

# Declare the port on which the application will be run.
EXPOSE 8080

# Import the user and group files.
COPY --from=builder /user/group /user/passwd /etc/

# Import the Certificate-Authority certificates for enabling HTTPS.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Import the compiled executable frmo the second stage.
COPY --from=builder /app /app

COPY --from=builder --chown=nobody /plans /plans

COPY ./dist /dist

# Run the container as an unprivileged user.
USER nobody:nobody

ENTRYPOINT ["/app"]
