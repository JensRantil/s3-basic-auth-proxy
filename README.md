S3 Basic Auth HTTP Proxy
========================

This small application proxies Basic Auth HTTP requests to an S3 bucket.

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

You can generate a sample documentation using `./s3-basic-auth-proxy serve generate`.

The file can be reloaded by sending `SIGHUP` to a running process.
