FROM alpine
COPY bin/balancer /bin
COPY config.yaml .
EXPOSE 80
ENTRYPOINT ["balancer","config.yaml"]