ARG GO_VERSION=1.20.10

# STAGE 1: building the executable
FROM golang:${GO_VERSION}-alpine AS build
RUN apk add --no-cache git
RUN apk --no-cache add ca-certificates

# add a user here because addgroup and adduser are not available in scratch
RUN addgroup -S century \
    && adduser -S -u 10000 -g century century

WORKDIR /src
COPY . ./
RUN go mod download

# Run tests
# RUN CGO_ENABLED=0 go test -timeout 30s -v github.com/???

# Build the executable
RUN CGO_ENABLED=0 go build \
    -installsuffix 'static' \
    -o /app .

# STAGE 2: build the container to run
FROM scratch AS final
LABEL maintainer="century"
COPY --from=build /app /app

# copy ca certs
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# copy users from builder
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group

USER century

ENTRYPOINT ["/app"]
