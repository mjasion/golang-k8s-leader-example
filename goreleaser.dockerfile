FROM alpine:3.18
ENTRYPOINT ["/k8s-leader-example"]
COPY k8s-leader-example /
