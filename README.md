# GopherPit

GopherPit is a tool that allows you to have remote import paths for Go (programming language) packages with custom domains. That way packages are independent of the version control system provider, whether it is GitHub, Bitbucket or a private repository. You can change it whenever you like, and also keep the same import paths. Also, custom domains means better branding of your packages, if you care about it.

For Git HTTP and HTTPS repositories a custom branch or tag can be specified. It allows to have different package paths for different package versions in the same repository that break backward compatibility with master or other branches. This type of Git references change is similar to [https://gopkg.in](gopkg.in) functionality, but it relies on package path configuration rather on versioning convention, and it is available for any Git repository hosting provider with support for HTTP protocol.

This service is meant for on-premises installation. A publicly available web service is hosted on [https://gopherpit.com](https://gopherpit.com) with the same functionalities.

## Installation

The latest GopherPit release can be downloaded from [https://github.com/gopherpit/gopherpit/releases](https://github.com/gopherpit/gopherpit/releases). Just unpack the content of the archive to a directory of your choosing.

Or you can build it yourself:

```text
$ go get -u gopherpit.com/gopherpit
$ cd $GOPATH/src/gopherpit.com/gopherpit
$ make
$ cd dist
```

All files for distribution are located under `dist` directory.

Docker image `gopherpit/gopherpit` is also provided, with a few examples on [Docker Hub](https://hub.docker.com/r/gopherpit/gopherpit/). 

Starting the service with default settings in the foreground is done by executing the binary:

```text
$ ./gopherpit
```

Log messages should appear in terminal. You can open [http://localhost:8080](http://localhost:8080) in browser and checkout the service.

To start the service in the background execute:

```text
$ ./gopherpit daemon
```

Log messages are written in files under configured `log` directory, by default it is a `log` directory next to the `gopherpit` executable.

To check the status of the daemonized process:

```text
$ ./gopherpit status
```

To stop the service:

```text
$ ./gopheprit stop
```

## Configuration introduction

To print all loaded configuration options:

```text
$ ./gopherpit config
```

All default configuration options are optimized for local testing. Each production environment requires its own specific settings, some of which are addressed in this introduction.

Each section of the configuration can be overridden by a YAML or JSON file with the same name under a configuration directory, which is printed at the end of the config command output, by default it is `/etc/gopherpit`.

For example, to change the `log` directory, create a file `/etc/gopherpit/logging.yaml` with the following content:

```yaml
log-dir: /path/to/log
```

Or you can achieve the same effect with a JSON file `/etc/gopherpit/logging.json`:

```json
{
    "log-dir": "/path/to/log"
}
```

Make sure that the directory can be created by the user under which GopherPit daemon is started. After starting the daemon, logs are saved there.

Every configuration parameter has a corresponding environment variable. For example the log directory can also be set with:

```text
$ GOPHERPIT_LOGGING_LOG_DIR=/path/to/log ./gopherpit daemon
```

Beside `log` directory, `storage` directory should be configured, too, but in different file (section). Create `/etc/gopherpit/gopherpit.yaml` file and add the following:

```yaml
storage-dir: /path/to/storage
```

Storage directory is used by GopherPit to store permanent or temporary data and it should be outside of the GopherPit installation directory.

## Serving on ports 80 and 443 on a custom domain

To be able to fully utilize GopherPit, it is required for service to listen to default http and https ports under a specific domain. These are privileged ports, so root user is required for starting the service. This is a simple "one-liner" to accomplish this:

```text
$ GOPHERPIT_DOMAIN=gopherpit.example.com GOPHERPIT_LISTEN=:80 GOPHERPIT_LISTEN_TLS=:443 ./gopherpit daemon
```

Or if you prefer to have an explicit configuration, set this options in `/etc/gopherpit/gopherpit.yaml`:

```yaml
listen: :80
listen-tls: :443
domain: gopherpit.example.com
storage-dir: /path/to/storage
```

And just start:

```text
# ./gopherpit daemon
```

Make sure that configured domain (`gopherpit.example.com` is just an example) has the right DNS record, and you should be able to access the service.

The domain specified in configuration is the domain that GopherPit recognizes as the one that the web interface should be served on. If you mistype or go to a different domain that is pointing to your GopherPit running instance, you will get a text message that no packages could be found, or a TLS error. In that case, make sure that domain in your browser and configuration match.

## TLS certificates and ACME provider

GopherPit is able to obtain TLS certificate from ACME provider, by default Let's Encrypt, and it allows you to register ACME user on a production or staging ACME directory with or without an E-mail address. When you access the service first time on port 80, and TLS listener is configured, it will present a web form to register ACME user.

The first time you access the domain you configured, GopherPit will try to obtain a TLS certificate. The domain must be accessible by the ACME provider, as the only validation method supported is ACME HTTP-01.

To summarize:

 - Make sure that your server is available to the ACME provider (or publicly on the Internet).
 - Make sure that DNS record for your domain points to the right IP address of your server.
 - Configure GopherPit to listen on ports 80 and 433 and to have domain specified.
 - Start GopherPit.
 - Open http://gopherpit.example.com in your browser and register ACME user.
 - Open https://gopherpit.example.com in your browser to obtain certificate.

ACME provider allows automatic obtaining of TLS certificates for domains added through the service, but if you do not need that functionality, it is possible to configure static TLS certificates.

## Static TLS certificates

It is not required to use ACME provider for TLS certificates. If you already have certificates for the domain, just include them in `gopherpit.yaml` configuration:

```yaml
tls-cert: /path/to/certificate-chain.pem
tls-key: /path/to/certificate-key.key
```

## SMTP issues and options

Default configuration for SMTP integration is to verify the certificate of SMTP server and that it listens on localhost port 25. Postfix and other MTAs may not have a valid certificate for localhost. In that case, either change the `smtp-host` in `email.yaml` configuration file, or in the same file set `smtp-skip-verify` to `true`. It is possible to have this type of error in log files `notifier api send email: x509: certificate is valid for gopherpit.example.com, not localhost`.

Some may use Gmail as SMTP server, and example of that `email.yaml` configuration in this case is:

```yaml
smtp-username: me@gmail.com
smtp-password: your password
smtp-host: smtp.gmail.com
smtp-port: 25
```

## Other notable options

Beside other options, the following are important for production deployments.

### gopherpit.yaml

```yaml
listen-internal: :6060
pid-file: /path/to/gopherpit.pid
google-analytics-id: "UA-123456x-1"
contact-recipient-email: gopherpit@localhost
```

Option "listen-internal" defines a listening address for "internal" debug and management server. It used for inspecting debug profiles, server status and handling maintenance mode.

By default `pid-file` is in the same directory as `gopherpit` executable, and  for production installations it should be changed to other location, for example `/var/run/gopherpit.pid`.

If "google-analytics-id" is not empty string, a Google Analytics block of code is added to every HTML page.

E-mails with content from the contact form is sent to address specified under "contact-recipient-email".

### email.yaml

```yaml
default-from: gopherpit@localhost
```

All e-mail messages to GopherPit users have "From" field set from value defined under "default-from".

## Configuration directory

By default all configuration files are read from `/etc/gopherpit`. It can be changed either with `--config-dir` optional argument:

```text
# ./gopherpit --config-dir /path/co/configurations daemon
```

Or with `GOPHERPIT_CONFIGDIR` environment variable:

```text
# GOPHERPIT_CONFIGDIR=/path/to/configurations ./gopherpit daemon
```

## Contributing

Please report bugs or feature requests in the [issue tracker at GitHub](https://github.com/gopherpit/gopherpit/issues).

## License

Unless otherwise noted, the GopherPit source files are distributed under the BSD-style license found in the LICENSE file.
