# log-iss

log-iss offers a [logplex HTTP input](https://github.com/heroku/logplex/blob/master/doc/README.http_input.md)-compatible endpoint and forwards received logs to a TCP port, where syslog-ng or similar might be listening.

An example submitter to log-iss is [log-shuttle](http://log-shuttle.io/).

## Configuration

log-iss is configured via the environment.

* `PORT`: TCP port number to make the endpoint available on. Given `PORT=5000`, the endpoint will be at `http://<host>:5000/logs`
* `FORWARD_DEST`: TCP host and port to forward received logs to. Example: `FORWARD_DEST=127.0.0.1:5001`
* `TOKEN_MAP`: A `,`-separated, `:`-separated list of usernames and tokens to accept. Example: `TOKEN_MAP=dan:logthis,system:islogging`
