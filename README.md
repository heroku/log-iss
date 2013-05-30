# log-iss

log-iss offers a [logplex HTTP input](https://github.com/heroku/logplex/blob/master/doc/README.http_input.md)-compatible endpoint and forwards received logs to a TCP port, where syslog-ng or similar might be listening.

An example submitter to log-iss is [log-shuttle](http://log-shuttle.io/).
