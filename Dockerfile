FROM golang:1.20-alpine3.18 as builder
WORKDIR /code
COPY go.* .
RUN go mod download
COPY main.go .
ARG SKAFFOLD_GO_GCFLAGS
RUN go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -trimpath -o /k8s-leader-example main.go

FROM alpine:3.18
# Define GOTRACEBACK to mark this container as using the Go language runtime
# for `skaffold debug` (https://skaffold.dev/docs/workflows/debug/).
ENV GOTRACEBACK=single
CMD ["./k8s-leader-example"]
COPY --from=builder /k8s-leader-example .
