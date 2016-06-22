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
      erik:
        password: "my%secret%password"
      peter:
        # If you want to obfuscate a password, you can put it in here base64-encoded.
        password: !!binary |
          aGVqCg==
      arnold:
        hash:
          salt: ksdfkdsj
          sha256: fd853dc703b2b67b0bcaffdf357685fb6480837c3e6e537526e71b858d6a38f8

You can generate a sample documentation using `./s3-basic-auth-proxy serve generate`.

Alternatives
------------
https://github.com/yegor256/s3auth - also hosted on s3auth.com. Written in Java. Not minimalistic. Requires JVM.
