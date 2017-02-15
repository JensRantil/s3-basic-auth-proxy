package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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

	hashCmd = app.Command("hash", "Generate a hash and a random salt.")
)

type Config struct {
	BufferSize int
	Aws        struct {
		Region string
		Bucket string
	}
	Users map[string]struct {
		Hash struct {
			Salt   string
			Sha256 string
		}
	}
}

const DefaultBufferSize = 1024 * 1024 // 1 MB

func main() {
	flagCommand := kingpin.MustParse(app.Parse(os.Args[1:]))

	switch flagCommand {
	case "generate":
		generate()
	case "hash":
		generateHash()
	case "serve":
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, *configFile); err != nil {
			fmt.Println("Could not read configuration file:", err)
			os.Exit(1)
		}

		c := Config{BufferSize: DefaultBufferSize}
		if err := yaml.Unmarshal(buf.Bytes(), &c); err != nil {
			fmt.Println("Could not parse configuration file:", err)
			os.Exit(1)
		}
		serve(c)
	}
}

const SALTLEN = 6

func generateHash() {
	var err error

	var text string
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter password: ")
	if text, err = reader.ReadString('\n'); err != nil {
		fmt.Println("Could not read the password:", err)
		os.Exit(1)
	}

	text = strings.TrimRightFunc(text, func(r rune) bool { return r == '\n' })

	data := make([]byte, 6)
	if _, err = rand.Read(data); err != nil {
		fmt.Println("Could not generate full salt:", err)
		os.Exit(1)
	}

	salt := hex.EncodeToString(data)

	fmt.Println("salt:", salt)
	fmt.Println("sha256:", calculateSha256(salt, text))
}

func calculateSha256(salt, data string) string {
	sum := sha256.Sum256([]byte(salt + data))

	a := make([]byte, 32)
	copy(a[:], sum[:])

	return hex.EncodeToString(a)
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
		"  arnold:",
		"    hash:",
		"      salt: abcdefg",
		"      sha256: 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
	}, "\n"))
}

func checkCredentials(c Config, inputUsername, inputPassword string) bool {
	for username, userdata := range c.Users {
		if username != inputUsername {
			continue
		}

		if calculateSha256(userdata.Hash.Salt, inputPassword) == userdata.Hash.Sha256 {
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

		// TODO: Set cache headers on request.

		// Execute the S3 request.

		var response *s3.GetObjectOutput
		if response, err = service.GetObject(&request); err != nil {
			status := http.StatusBadGateway
			if awsErr, ok := err.(awserr.RequestFailure); ok {
				status = awsErr.StatusCode()
			}
			logRequest(r, status, err.Error())
			http.Error(w, "Error talking to S3.", status)

			return
		}
		defer response.Body.Close()

		// Return the response.

		// TODO: Write all headers from response to w.

		if _, err = io.Copy(bufio.NewWriterSize(w, c.BufferSize), bufio.NewReaderSize(response.Body, c.BufferSize)); err != nil {
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
