# Build the manager binary
FROM golang:1.22.0-alpine3.18 as builder

COPY . /go/src/github.com/uselagoon/lagoon/services/insights-handler/
WORKDIR /go/src/github.com/uselagoon/lagoon/services/insights-handler/

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o insights-handler main.go

# we pull the trivy binary from aquasec's alpine based image
FROM aquasec/trivy:0.49.0 as trivy

# Use distroless as minimal base image to package the insights-handler binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
#FROM gcr.io/distroless/static:nonroot

FROM alpine:3.18

COPY --from=trivy /usr/local/bin/trivy /usr/local/bin/trivy

WORKDIR /
COPY --from=builder /go/src/github.com/uselagoon/lagoon/services/insights-handler/insights-handler .

COPY default_filter_transformers.yaml /default_filter_transformers.yaml
USER 65532:65532

ENTRYPOINT ["/insights-handler"]