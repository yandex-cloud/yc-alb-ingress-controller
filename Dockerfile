# Build the manager binary
FROM golang:1.21 as builder

ARG COMMIT
ARG CREATED_AT

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# copying source and vendor so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
COPY vendor vendor

# Copy the go source
COPY main.go main.go

COPY controllers/ controllers/
COPY pkg/ pkg/
COPY api/ api/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot

ARG COMMIT
ARG CREATED_AT

LABEL org.opencontainers.image.created=${CREATED_AT}
LABEL org.opencontainers.image.revision=${COMMIT}
LABEL org.opencontainers.image.authors="YC L7 Team"

WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
