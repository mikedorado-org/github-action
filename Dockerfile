# syntax=docker/dockerfile:1

######################
# Builder Stage
######################
FROM ghcr.io/github/gh-base-image/go-builder-focal:20230619-230510-gfc77d00cd@sha256:eb6fa65559cf97f288792706ae860dcf01c3e902c88c8f8b5441a18c74d8053e AS builder
WORKDIR /go/src/app
ARG SKAFFOLD_GO_GCFLAGS

# Module Download
ADD go.* /go/src/app
ENV GOPROXY=https://goproxy.githubapp.com/mod,https://proxy.golang.org,direct
ENV GOPRIVATE=''
ENV GONOPROXY=''
ENV GONOSUMDB=github.com/github/*

COPY .goproxytoken .
RUN echo "machine goproxy.githubapp.com login nobody password $(cat .goproxytoken)" > $HOME/.netrc
RUN go mod download

# Build
ADD . /go/src/app/
RUN --mount=type=cache,target=/root/.cache/go-build,sharing=locked go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /go/bin/app ./cmd

######################
# Base Runtime Stage
######################
FROM ghcr.io/github/gh-base-image/gh-fips-base-focal:20230616-220302-g87da16c6a@sha256:0f96611bffc35a6a30a26e488bca6c638711f1fb848916e73b22e97aaff6ddbe
LABEL org.opencontainers.image.source=https://github.com/github/actions-example-go
ENV GOTRACEBACK=single
COPY --from=builder /go/bin/app /
USER github
CMD ["/app"]
