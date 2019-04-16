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

# final image
FROM ubuntu
RUN apt-get update
RUN apt-get install git -y
COPY --from=tekctl_binary /tekctl /usr/bin
ENTRYPOINT ["tekctl"]
