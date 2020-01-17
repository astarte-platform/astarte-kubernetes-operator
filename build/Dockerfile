FROM golang:1.13 as builder

WORKDIR /operator

# Copy stuff
COPY . .

# Build the Operator
RUN go build -o astarte-operator github.com/astarte-platform/astarte-kubernetes-operator/cmd/manager

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/astarte-operator \
    USER_UID=1001 \
    USER_NAME=astarte-operator

ARG BUILT_BINARY_NAME=astarte-operator

# install operator binary
COPY --from=builder /operator/astarte-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
