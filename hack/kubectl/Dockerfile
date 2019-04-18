# kubectl image
FROM ubuntu:latest AS kubectl_binary
RUN apt-get update
RUN apt-get install curl -y
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl

# final image
FROM ubuntu
COPY --from=kubectl_binary /kubectl /usr/bin
ENTRYPOINT ["kubectl"]
