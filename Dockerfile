FROM golang:1.10 AS build

COPY . /go/src/github.com/heroku/log-iss
WORKDIR /go/src/github.com/heroku/log-iss
RUN make clean bin/forwarder

FROM ubuntu:18.04

RUN mkdir -p /src/log-iss/bin
COPY --from=build /go/src/github.com/heroku/log-iss/bin/forwarder /src/log-iss/bin/forwarder
CMD [ "/src/log-iss/bin/forwarder" ]
