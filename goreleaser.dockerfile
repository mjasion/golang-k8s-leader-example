FROM alpine:3.20
ENTRYPOINT ["/k8s-leader-example"]
COPY k8s-leader-example /
