# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

LABEL maintainer="Josh Ellithorpe <quest@mac.com>"

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/bourbaki-czz/classzz

# Switch to the correct working directory.
WORKDIR /go/src/github.com/bourbaki-czz/classzz

# Restore vendored packages.
RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure

# Build the code and the cli client.
RUN go install .
RUN go install ./cmd/czzctl

# Symlink the config to /root/.classzz/classzz.conf
# so czzctl requires fewer flags.
RUN mkdir -p /root/.classzz
RUN ln -s /data/classzz.conf /root/.classzz/classzz.conf

# Create the data volume.
VOLUME ["/data"]

# Set the start command. This starts classzz with
# flags to save the blockchain data and the
# config on a docker volume.
ENTRYPOINT ["classzz", "--addrindex", "--txindex", "-b", "/data", "-C", "/data/classzz.conf"]

# Document that the service listens on port 8333.
EXPOSE 8333
