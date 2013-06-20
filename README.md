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

## Configuration

log-iss is configured via the environment.

* `DEPLOY`: A label naming this instance of log-iss. Used as the `source` value for [l2met](https://github.com/ryandotsmith/l2met#log-conventions)-compatible log lines.
* `PORT`: TCP port number to make the endpoint available on. Given `PORT=5000`, the endpoint will be at `http://<host>:5000/logs`
* `FORWARD_DEST`: TCP host and port to forward received logs to. Example: `FORWARD_DEST=127.0.0.1:5001`
* `FORWARD_DEST_CONNECT_TIMEOUT`: Time in seconds to wait for a connection to `FORWARD_DEST`, default is `10`
* `TOKEN_MAP`: A `,`-separated, `:`-separated list of usernames and tokens to accept. Example: `TOKEN_MAP=dan:logthis,system:islogging`
