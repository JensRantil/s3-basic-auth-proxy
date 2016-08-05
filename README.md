S3 Basic Auth HTTP Proxy
========================

A minimalistic reverse HTTP proxy that can be used to add Basic Auth to an S3 bucket.

Currently, only `GET`s of files are supported.

Usage
-----
First add the following to `~/.aws/credentials`:

    [default]
    aws_access_key_id = XXX
    aws_secret_access_key = YYY

(Set `AWS_PROFILE` environment variable if you would like to use something else
than `default`.) You can then execute:

    ./s3-basic-auth-proxy serve my-auth-file.txt

where `my-auth-file.txt` contains

    aws:
      region: eu-west-1
      bucket: my-bucket
    users:
      arnold:
        hash:
          salt: ksdfkdsj
          sha256: fd853dc703b2b67b0bcaffdf357685fb6480837c3e6e537526e71b858d6a38f8
      peter:
        hash:
          salt: ffkfkdsj
          sha256: fd853dc703b2b67b0bcaffdf357685fb6480837c3e6e537526e71b858d6a38f8

You can generate a sample documentation using `./s3-basic-auth-proxy generate`.
You can also generate hash using `./s3-basic-auth-proxy hash`. Command line
usage and help:

```bash
$ ./s3-basic-auth-proxy --help-long                                                                                          [15:18:14]
usage: s3-basic-auth-proxy [<flags>] <command> [<args> ...]

S3 Basic Auth proxy.

Flags:
  --help  Show context-sensitive help (also try --help-long and --help-man).

Commands:
  help [<command>...]
    Show help.


  generate
    Generate an example configuration.


  serve [<flags>] <auth-file>
    Run the proxy server.

    --addr=":80"  HTTP Server listen address.

  hash
    Generate a hash and a random salt.
```

Alternatives
------------
https://github.com/yegor256/s3auth - also hosted on s3auth.com. Written in Java. Not minimalistic. Requires JVM.
