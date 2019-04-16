# build image
FROM golang:latest AS tekctl_binary
# create the project directory
RUN mkdir -p /tektoncd.dev/experimental
WORKDIR /tektoncd.dev/experimental
# cache go dependencies
ADD go.mod /tektoncd.dev/experimental
ADD go.sum /tektoncd.dev/experimental
RUN go mod download
RUN go get github.com/google/wire/cmd/wire
# prepare for build
ADD . /tektoncd.dev/experimental
# build tekctl
ENV CGO_ENABLED 0
ENV GO111MODULE on
RUN go generate .
RUN go build -o /tekctl .

# kubectl image
FROM ubuntu:latest AS kubectl_binary
RUN apt-get update
RUN apt-get install curl -y
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl

# final image
FROM ubuntu
COPY --from=kubectl_binary /kubectl /usr/bin
COPY --from=tekctl_binary /tekctl /usr/bin
ENTRYPOINT ["tekctl"]
