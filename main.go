package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app = kingpin.New("s3-basic-auth-proxy", "S3 Basic Auth proxy.")

	generateCmd = app.Command("generate", "Generate an example configuration.")

	serveCmd      = app.Command("serve", "Run the proxy server.")
	httpInterface = serveCmd.Flag("addr", "HTTP Server listen address.").Default(":80").String()
	configFile    = serveCmd.Arg("auth-file", "Authentication file.").Required().File()
)

type Config struct {
	Aws struct {
		Region string `yaml:"region"`
		Bucket string `yaml:"bucket"`
	}
	Users map[string]struct {
		Password string `yaml:"password"`
	}
}

func main() {
	flagCommand := kingpin.MustParse(app.Parse(os.Args[1:]))

	switch flagCommand {
	case "generate":
		generate()
	case "serve":
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, *configFile); err != nil {
			fmt.Println("Could not read configuration file:", err)
			os.Exit(1)
		}

		c := Config{}
		if err := yaml.Unmarshal(buf.Bytes(), &c); err != nil {
			fmt.Println("Could not parse configuration file:", err)
			os.Exit(1)
		}
		serve(c)
	}
}

func generate() {
	fmt.Println(strings.Join([]string{
		"aws:",
		"  region: eu-west-1",
		"  bucket: my-bucket",
		"users:",
		"  erik:",
		"    password: \"my%secret%password\"",
		"  peter:",
		"    # If you want to obfuscate a password, you can put it in here base64-encoded.",
		"    password: !!binary |",
		"      aGVqCg==",
	}, "\n"))
}

func checkCredentials(c Config, inputUsername, inputPassword string) bool {
	for username, userdata := range c.Users {
		if username == inputUsername && inputPassword == userdata.Password {
			return true
		}
	}
	return false
}

func serve(c Config) {
	awsConfig := aws.Config{
		Region: aws.String(c.Aws.Region),
	}
	s := session.New(&awsConfig)

	if _, err := s.Config.Credentials.Get(); err != nil {
		fmt.Println("Could not find credentials. Please create: ~/.aws/credentials")
		os.Exit(1)
	}

	service := s3.New(s)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var err error

		// Authenticate

		var username, password string
		var ok bool

		if username, password, ok = r.BasicAuth(); ok {
			ok = checkCredentials(c, username, password)
		}
		if !ok {
			w.Header()["WWW-Authenticate"] = []string{"Basic realm=\"Please enter your username, followed by password.\""}

			status := http.StatusUnauthorized
			w.WriteHeader(status)
			logRequest(r, status, "")

			return
		}

		// Build the S3 request.

		request := s3.GetObjectInput{
			Bucket: &c.Aws.Bucket,
			Key:    &r.URL.Path,
		}

		// TODO: Set cache header on request.

		// Execute the S3 request.

		var response *s3.GetObjectOutput
		if response, err = service.GetObject(&request); err != nil {
			status := http.StatusBadGateway
			logRequest(r, status, err.Error())
			http.Error(w, "Error talking to S3.", status)

			return
		}

		// Return the response.

		defer response.Body.Close()
		if _, err = io.Copy(w, response.Body); err != nil {
			status := http.StatusBadGateway
			logRequest(r, status, err.Error())
			http.Error(w, "Error fetching from S3.", status)

			return
		}

		logRequest(r, http.StatusOK, "")

	})

	log.Fatal(http.ListenAndServe(*httpInterface, nil))
}

func logRequest(r *http.Request, outcome int, msg string) {
	if msg == "" {
		msg = "-"
	}
	log.Println(r.URL.Path, r.RemoteAddr, outcome, msg)
}
