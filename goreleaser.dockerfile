FROM alpine:3.19
ENTRYPOINT ["/k8s-leader-example"]
COPY k8s-leader-example /
