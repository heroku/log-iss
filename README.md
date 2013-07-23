# log-iss

log-iss offers a
[logplex HTTP input](https://github.com/heroku/logplex/blob/master/doc/README.http_input.md)-compatible
endpoint and forwards received logs to a TCP port, where syslog-ng or similar
might be listening.

An example submitter to log-iss is [log-shuttle](http://log-shuttle.io/).

Log delivery is synchronous, with a five second timeout. If log-iss is unable to
write `POST`ed messages to the backend TCP connection within the timeout it will
respond with status 504.

Upon receiving `SIGTERM` or `SIGINT` log-iss will stop ingesting logs, respond to
all `POST`s with status 503, wait for pending deliveries (subject to the five
second timeout) to drain, then exit.

log-iss will use four persistent connections per process to the destination
configured in `FORWARD_DEST`.

## Configuration

log-iss is configured via the environment.

* `DEPLOY`: A label naming this instance of log-iss. Used as the `source` value for [l2met](https://github.com/ryandotsmith/l2met#log-conventions)-compatible log lines.
* `PORT`: TCP port number to make the endpoint available on. Given `PORT=5000`, the endpoint will be at `http://<host>:5000/logs`
* `FORWARD_DEST`: TCP host and port to forward received logs to. Example: `FORWARD_DEST=127.0.0.1:5001`
* `FORWARD_DEST_CONNECT_TIMEOUT`: Time in seconds to wait for a connection to `FORWARD_DEST`, default is `10`
* `TOKEN_MAP`: A `,`-separated, `:`-separated list of usernames and tokens to accept. Example: `TOKEN_MAP=dan:logthis,system:islogging`
* `ENFORCE_SSL`: If set to `1`, respond with 400 to any `POST`s where the `X-Forwarded-Proto` request header is not `https`

## Running locally

Assumes working [Go](http://golang.org/doc/install) installation with
`$GOPATH/bin` in `$PATH` as well as something listening at port 5001 (could be
`nc -l -k 5001` for very simple testing).

```bash
$ go get github.com/heroku/log-iss
$ DEPLOY=local PORT=5000 FORWARD_DEST=localhost:5001 TOKEN_MAP=test:token log-iss
# in another shell
$ echo "64 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - - hi" | curl -v -u test:token -H "Content-Type: application/logplex-1" --data-binary @/dev/stdin http://localhost:5000/logs
```

## Running on the platform

```bash
$ DEPLOY=`whoami`
$ heroku create log-iss-$DEPLOY -r $DEPLOY --buildpack https://codon-buildpacks.s3.amazonaws.com/buildpacks/kr/go.tgz
$ heroku config:set -r log-iss-$DEPLOY DEPLOY=$DEPLOY ENFORCE_SSL=1 FORWARD_DEST=my-syslog-host.com:601 TOKEN_MAP=syslog:$(openssl rand -hex 20)
$ heroku labs:enable -r log-iss-$DEPLOY http-request-id
$ git push $DEPLOY master
$ echo "64 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - - hi" | curl -v -u syslog:<generated token> -H "Content-Type: application/logplex-1" --data-binary @/dev/stdin https://log-iss-$DEPLOY.herokuapp.com/logs
```
