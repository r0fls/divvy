FROM golang:stretch

RUN mkdir -p /go/src/github.com/r0fls/divvy
WORKDIR /go/src/github.com/r0fls/divvy
COPY . .
RUN go build
CMD ["./divvy"]
