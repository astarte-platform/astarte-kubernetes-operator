FROM golang:1.13 as builder

WORKDIR /operator

# copy-in our module info (to ensure layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the Operator statically
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags '-w -s -extldflags "-static"' -o astarte-operator github.com/astarte-platform/astarte-kubernetes-operator/cmd/manager

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /operator/astarte-operator /astarte-operator

CMD ["/astarte-operator"]
