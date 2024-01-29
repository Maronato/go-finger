# Finger

Webfinger handler / standalone server written in Go.

## Features
- 🍰  Easy YAML configuration
- 🪶  Single 8MB binary / 0% idle CPU / 4MB idle RAM
- ⚡️   Sub millisecond responses at 10,000 request per second
- 🐳  10MB Docker image

## In your existing server

To use Finger in your existing server, download the package as a dependency:

```bash
go get git.maronato.dev/maronato/finger@latest
```

Then, use it as a regular `http.Handler`:

```go
package main

import (
	"log"
	"net/http"

	"git.maronato.dev/maronato/finger/handler"
	"git.maronato.dev/maronato/finger/webfingers"
)

func main() {
  // Create the webfingers map that will be served by the handler
  fingers, err := webfingers.NewWebFingers(
    // Pass a map of your resources (Subject key followed by it's properties and links)
    // the syntax is the same as the fingers.yml file (see below)
    webfingers.Resources{
      "user@example.com": {
        "name": "Example User",
      },
    },
    // Optionally, pass a map of URN aliases (see urns.yml for more)
    // If nil is provided, no aliases will be used
    webfingers.URNAliases{
      "name": "http://schema.org/name",
    },
  )
  if err != nil {
    log.Fatal(err)
  }

  mux := http.NewServeMux()
  // Then use the handler as a regular http.Handler
  mux.Handle("/.well-known/webfinger", handler.WebfingerHandler(fingers))

  log.Fatal(http.ListenAndServe("localhost:8080", mux))
}
```

## As a standalone server

If you don't have a server, Finger can also serve itself. You can install it via `go install` or use the Docker image.

Via `go install`:

```bash
go install git.maronato.dev/maronato/finger@latest
```

Via Docker:

```bash
docker run \
    --name finger \
    -p 8080:8080 \
    -v ${PWD}/fingers.yml:/app/fingers.yml \
    git.maronato.dev/maronato/finger
```

## Usage

If you installed it using `go install`, run
```bash
finger serve
```
To start the server on port `8080`. Your resources will be queryable via `locahost:8080/.well-known/webfinger?resource=<your-resource>`

If you're using Docker, the use the same command in the install section.

By default, no resources will be exposed. You can create resources via a `fingers.yml` file. It should contain a collection of resources as keys and their attributes as their objects.

Some default URN aliases are provided via the built-in mapping ([`urns.yml`](./urns.yml)). You can replace that with your own or use URNs directly in the `fingers.yml` file.

Here's an example:
```yaml
# fingers.yml

# Resources go in the root of the file. Email address will have the acct: 
# prefix added automatically.
alice@example.com:
  # "avatar" is an alias of "http://webfinger.net/rel/avatar"
  # (see urns.yml for more)
  avatar: "https://example.com/alice-pic"

  # If the value is a URI, it'll be exposed as a webfinger link
  openid: "https://sso.example.com/"

  # If the value of the attribute is not a URI, it will be exposed as a
  # webfinger property
  name: "Alice Doe"

  # You can also specify URN's directly instead of the aliases
  http://webfinger.net/rel/profile-page: "https://example.com/user/alice"

bob@example.com:
  name: Bob Foo
  openid: "https://sso.example.com/"

# Resources can also be URIs
https://example.com/user/charlie:
  name: Charlie Baz
  profile: https://example.com/user/charlie
```

### Example queries
<details>
<summary><b>Query Alice</b><pre>GET http://localhost:8080/.well-known/webfinger?resource=acct:alice@example.com</pre></summary>

```json
{
  "subject": "acct:alice@example.com",
  "links": [
    {
      "rel": "avatar",
      "href": "https://example.com/alice-pic"
    },
    {
      "rel": "openid",
      "href": "https://sso.example.com/"
    },
    {
      "rel": "http://webfinger.net/rel/profile-page",
      "href": "https://example.com/user/alice"
    }
  ],
  "properties": {
    "name": "Alice Doe"
  }
}
```
</details>


<details>
<summary><b>Query Bob</b><pre>GET http://localhost:8080/.well-known/webfinger?resource=acct:bob@example.com</pre></summary>

```json
{
  "subject": "acct:bob@example.com",
  "links": [
    {
      "rel": "http://openid.net/specs/connect/1.0/issuer",
      "href": "https://sso.example.com/"
    }
  ],
  "properties": {
    "http://schema.org/name": "Bob Foo"
  }
}
```
</details>


<details>
<summary><b>Query Charlie</b><pre>GET http://localhost:8080/.well-known/webfinger?resource=https://example.com/user/charlie</pre></summary>

```JSON
{
  "subject": "https://example.com/user/charlie",
  "links": [
    {
      "rel": "http://webfinger.net/rel/profile-page",
      "href": "https://example.com/user/charlie"
    }
  ],
  "properties": {
    "http://schema.org/name": "Charlie Baz"
  }
}
```
</details>

## Commands

Finger exposes two commands: `serve` and `healthcheck`. `serve` is the default command and starts the server. `healthcheck` is used by the Docker healthcheck to check if the server is up.

## Configs
Here are the config options available. You can change them via command line flags or environment variables:

| CLI flag            | Env variable     | Default                                | Description                            |
| ------------------- | ---------------- | -------------------------------------- | -------------------------------------- |
| `-p, --port`        | `WF_PORT`        | `8080`                                 | Port where the server listens to       |
| `-h, --host`        | `WF_HOST`        | `localhost` (`0.0.0.0` when in Docker) | Host where the server listens to       |
| `-f, --finger-file` | `WF_FINGER_FILE` | `fingers.yml`                          | Path to the webfingers definition file |
| `-u, --urn-file`    | `WF_URN_FILE`    | `urns.yml`                             | Path to the URNs alias file            |
| `-d, --debug`       | `WF_DEBUG`       | `false`                                | Enable debug logging                   |

### Docker config
If you're using the Docker image, you can mount your `fingers.yml` file to `/app/fingers.yml` and the `urns.yml` to `/app/urns.yml`.

To run the docker image with flags or a different command, specify the command followed by the flags:
```bash
# Start the server on port 3030 in debug mode with a different fingers file
docker run git.maronato.dev/maronato/finger serve --port 3030 --debug --finger-file /app/my-fingers.yml

# or run a healthcheck on a different finger container
docker run git.maronato.dev/maronato/finger healthcheck --host otherhost --port 3030
```

## Development

You need to have [Go](https://golang.org/) installed to build the project.

Clone the repo and run `make build` to build the binary. You can then run `./finger serve` to start the server.

A few other commands are:
 - `make run` to run the server
 - `make test` to run the tests
 - `make lint` to run the linter
 - `make clean` to clean the build files

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
