FROM golang:1.10 AS build

# This works as long as log-iss is public.
RUN go get github.com/heroku/log-iss/cmd/forwarder


FROM ubuntu:18.04

RUN mkdir -p /src/log-iss/bin
COPY --from=build /go/bin/forwarder /src/log-iss/bin/forwarder
CMD [ "/src/log-iss/bin/forwarder" ]
