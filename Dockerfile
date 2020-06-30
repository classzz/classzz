FROM golang AS builder 

RUN go get -u github.com/golang/dep/cmd/dep
RUN go get github.com/classzz/classzz \
    && cd /go/src/github.com/classzz/classzz \
    && dep ensure \
    && CGO_ENABLED=0 go install .\
    && CGO_ENABLED=0 go install ./cmd/czzctl 

FROM scratch
MAINTAINER Josh Ellithorpe <quest@mac.com>
WORKDIR /app/

VOLUME ["/data"]
EXPOSE 8333 8334
ENTRYPOINT ["/app/classzz", "--addrindex", "--txindex","-b", "/data"]

COPY --from=builder /go/bin/classzz /app/classzz
COPY --from=builder /go/bin/czzctl /app/czzctl
COPY --from=builder /go/src/github.com/classzz/classzz/csatable.bin /app/csatable.bin
