# Lekko Go SDK

The Lekko Go SDK lets you add dynamic configuration to your Go applications by handling communication with Lekko and evaluation behind the scenes.

This SDK is intended to be used with Lekko’s code transformation tools. Make sure your project is set up with Lekko by following our [Getting Started](https://docs.lekko.com) guide.

## Prerequisites

- Go version 1.21 or greater
- After running the Lekko Getting Started guide, you should have:
  - The Lekko CLI installed
  - A `.lekko` file at the project root, pointing to a lekko directory in the project (e.g. `internal/lekko`) and a lekko repository
  - This SDK, installed as a library
  - An available Lekko API key

## Manual installation

```bash copy
go get github.com/lekkodev/go-sdk@latest
```

## Example

The following is an example of a lekko file under the lekko directory specified in your project’s `.lekko`.

```go copy filename="internal/lekko/example/example.go"
package lekkoexample

import (
    "strings"
)

// Whether to enable beta features
func getEnableBeta(segment string) bool {
    if strings.HasPrefix(segment, "enterprise") {
        return true
    }
    return false
}

// Return text based on environment
func getReturnText(env string) string {
    if env == "production" {
        return "foobar"
    } else if env == "development" {
        return "foo"
    }
    return "bar"
}
```

### Notes

A few things to notice about the above example:

- The lekko file contains 2 [lekkos](/anatomy), `enable-beta` and `log-level`. Remember, lekkos are just simple (but powerful) functions!
- The file is located in `internal/lekko/example/example.go`, in its own directory. This is intentional - and you'll see why in the next step.
- The package name is `lekkoexample`, prefixed with `lekko`. This is to prevent collisions with common package names and language-specific keywords when lekkos are shared across projects using different languages (e.g. `default`).

### Code transformation

We can generate transformed code using the Lekko CLI in your terminal. This command should be run at the project root.

```bash copy
lekko bisync
```

With the generated code, your project should now look like this:

```
.lekko
go.mod
go.sum
internal
└── lekko
    ├── client_gen.go
    └── example
        ├── example.go
        └── example_gen.go
...
```

In `example_gen.go`, you'll see the transformed functions that call out to the Lekko Go SDK as necessary. Changes to generated files will be overwritten.

### Usage

To call your lekkos, you'll use the generated SDK client code (in `internal/lekko/client_gen.go`).

```go showLineNumbers copy filename="main.go"
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"example/lekko"
)

func main() {
	env := os.Getenv("ENV")
    // Initialize the Lekko SDK client. This connects to Lekko and sets up periodic refreshing of lekkos and evaluation event handling.
	client := lekko.NewLekkoClient(context.TODO())
	// Start a simple HTTP server on localhost:8080
	// You can send requests using `curl localhost:8080`
	addr := ":8080"
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // Call a lekko via the initialized client. Note that the package `lekkoexample` is "transformed" to the `Example` field on the generated client.
        // We pass the app’s env as a "context variable" for evaluation (which the lekko expects, as defined above)
		w.Write([]byte(client.Example.GetReturnText(env)))
	})
	log.Println("Listening on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

```

## Environment variables

For the SDK client to connect to Lekko's services, you must provide the environment variable `LEKKO_API_KEY` when running the project.

Example:

```bash
LEKKO_API_KEY=<YOUR KEY> go run main.go
```

You can either use libraries such as [godotenv](https://github.com/joho/godotenv), pass them in directly when executing the project binary, or set them through tools that you use already for building/running your project (e.g. [Docker](https://docs.docker.com/compose/environment-variables/)).

Note that even if the API key is missing or malformed, your application will run without issues. However, the SDK will not be connected to Lekko and so the static fallback code will be used instead for evaluation. Most of the time during local development, this is the behavior you want - by not setting an API key, your local setup doesn't have to rely on an external dependency but can still behave the same using the local function definitions.

## Debug mode

You can set the `LEKKO_DEBUG` environment to `true` to enable debug logging of evaluations in your project.

Example:

```bash
LEKKO_DEBUG=true go run main.go
```

This allows you to see whenever lekkos are evaluated in your code and what contexts they are evaluated with.

```
2024/07/09 18:02:14 INFO LEKKO_DEBUG=true, running with additional logging
2024/07/09 18:02:15 INFO Connected to Lekko repository=lekkodev/lekko-configs opts=[lekko_a3fdbf*******************************************************************]
2024/07/09 18:02:27 INFO Lekko evaluation name=default/can-access context="{\"is_user_admin\":false}" result=value:true
2024/07/09 18:02:28 INFO Lekko evaluation name=default/can-access context="{\"is_user_admin\":true}" result=value:false
```

This logging is performed through [slog](https://pkg.go.dev/log/slog)'s default logger, so it can easily be customized to suit your desired formatting.

## Error handling

Lekko's tools are designed with safety and developer experience in mind.

As you can see in the above examples, initializing the Lekko SDK client and calling lekkos does not require any error handling. This is because there is always a safe default to fall back to - the original lekko code which is included at build time as a [static fallback](https://docs.lekko.com/static-fallback).

If you have any concerns about performance or stability, you can reference our related docs [here](https://docs.lekko.com/performance-reliability).
