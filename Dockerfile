FROM golang:1.23.6-alpine
RUN apk --no-cache add make git gcc libtool musl-dev ca-certificates dumb-init 

WORKDIR /go/src
COPY . .

RUN go build -o app .

CMD ["./app"]
