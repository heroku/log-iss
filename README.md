# log-iss

log-iss offers a
[logplex HTTP input](https://github.com/heroku/logplex/blob/master/doc/README.http_input.md)-compatible
endpoint and forwards received logs to a TCP port, where syslog-ng or similar
might be listening.

An example submitter to log-iss is [log-shuttle](http://log-shuttle.io/).

Currently log delivery is best-effort. It can be reasonably assumed that if log-iss
receives a `POST` with log data and the backend TCP connection is healthy that the
`POST`ed message(s) will be delivered. There is a finite in-memory buffer that is
used when the backend TCP connection is unavailable.

## Configuration

log-iss is configured via the environment.

* `DEPLOY`: A label naming this instance of log-iss. Used as the `source` value for [l2met](https://github.com/ryandotsmith/l2met#log-conventions)-compatible log lines.
* `PORT`: TCP port number to make the endpoint available on. Given `PORT=5000`, the endpoint will be at `http://<host>:5000/logs`
* `FORWARD_DEST`: TCP host and port to forward received logs to. Example: `FORWARD_DEST=127.0.0.1:5001`
* `TOKEN_MAP`: A `,`-separated, `:`-separated list of usernames and tokens to accept. Example: `TOKEN_MAP=dan:logthis,system:islogging`
