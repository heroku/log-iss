##### Google OAuth HTTP handler

Lovingly based on <https://github.com/kr/githubauth>

See <http://godoc.org/github.com/freeformz/googlegoauth> for documentation.

###### Google Setup

* Log into: https://console.developers.google.com
* Create a project.
* Under "APIs & Auth", click "Credentials"
* Under "OAuth", click "Create new Client ID"
* Leave the "Application Type" set to "Web application"
* Under "Authorized Javascript Origins" enter: `https://<the.host.domain>`
* Under "Authorized Redirect URL" enter: `https://<the.host.domain>/auth/callback/google`
* Click "Create Client ID"
* Under "APIs & Auth", click "Consent screen"
* Enter your/an email address, Product Name, click "Save". What you enter here will appear on the Google OAuth pages when authenticating.

###### Example

see [cmd/example/main.go](https://github.com/freeformz/googlegoauth/blob/master/cmd/example/main.go)

```shell
$ heroku create -b https://github.com/heroku/heroku-buildpack-go.git
$ heroku config:set KEY=$(openssl rand -hex 32) CLIENT_ID=^^ CLIENT_SECRET=^^ REQUIRE_DOMAIN=$your_domain
$ git push heroku master
```

###### To Do

* Merge this with https://github.com/kr/githubauth & https://github.com/heroku/herokugoauth
