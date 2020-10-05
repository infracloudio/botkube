# Copyright (c) 2019 InfraCloud Technologies
#
# Permission is hereby granted, free of charge, to any person obtaining a copy of
# this software and associated documentation files (the "Software"), to deal in
# the Software without restriction, including without limitation the rights to
# use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
# the Software, and to permit persons to whom the Software is furnished to do so,
# subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
# FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
# COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
# IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
# CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

# Development image
FROM golang:1.12-alpine3.10 AS build

ARG TARGETPLATFORM

WORKDIR /app

ENV GO111MODULE=on \
    CGO_ENABLED=0

# Add git
RUN apk update && apk add git curl

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN export GOOS=$(echo ${TARGETPLATFORM} | cut -d / -f1) && \
    export GOARCH=$(echo ${TARGETPLATFORM} | cut -d / -f2) && \
    GOARM=$(echo ${TARGETPLATFORM} | cut -d / -f3 | cut -c2-) && \
    go build -o /go/bin/botkube ./cmd/botkube

# Install kubectl binary
RUN export GOARCH=$(echo ${TARGETPLATFORM} | cut -d / -f2) && \
    apk add --no-cache ca-certificates git \
    && wget -q https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/${GOARCH}/kubectl -O /usr/local/bin/kubectl \
    && chmod +x /usr/local/bin/kubectl

# Production image
FROM alpine:3.10

# Create Non Privilaged user
RUN addgroup --gid 101 botkube && \
    adduser -S --uid 101 --ingroup botkube botkube

# Run as Non Privilaged user
USER botkube

COPY --from=build /go/bin/botkube /go/bin/botkube
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /usr/local/bin/kubectl /usr/local/bin/kubectl

ENTRYPOINT /go/bin/botkube
