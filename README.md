# Lekko Go SDK

The Lekko Go SDK lets you add dynamic configuration to your Go applications by handling communication with Lekko and evaluation behind the scenes.

This SDK is intended to be used with Lekko's code transformation tools. Make sure your project is set up with Lekko by following our [Getting Started](https://docs.lekko.com/) guide.

## Prerequisites

- Go version 1.19 or greater
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

The following is an example of a lekko file under the lekko directory specified in your project's `.lekko`.

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

// Level of general log output
func getLogLevel(env string) string {
    if env == "production" {
        return "info"
    } else if env == "test" {
        return "trace"
    }
    return "debug"
}
```

### Notes

A few things to notice about the above example:

- The lekko file contains 2 [lekkos](https://docs.lekko.com/anatomy), `enable-beta` and `log-level`. Remember, lekkos are just simple (but powerful) functions!
- The file is located in `internal/lekko/example/example.go`, in its own directory. This is intentional - and you'll see why in the next step.
- The package name is `lekkoexample`, prefixed with `lekko`. This is to prevent collisions with common package names and language-specific keywords for when lekkos are shared across projects using different languages (e.g. `default`).

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

```go copy filename="main.go"
package main

import (
    "context"
    "fmt"

    "github.com/acme-corp/my-go-project/internal/lekko"
)

func main() {
    // Initialize the Lekko SDK client. This connects to Lekko and sets up periodic refreshing of lekkos and evaluation event handling.
    client := lekko.NewLekkoClient(context.Background())

    // Call a lekko via the initialized client. Note that the package `lekkoexample` is "transformed" to the `Example` field on the generated client.
    // We pass the app's env as a "context variable" for evaluation (which the log-level lekko expects, as defined above)
    appEnv := env.GetEnvOrDefault("APP_ENV", "development")
    logLevel := client.Example.GetLogLevel(appEnv)

    fmt.Printf("The log level is %s", logLevel)
}
```

## Environment variables

The following Lekko environment variables are only required for deployment if you want to connect to Lekko's services. You don't need to connect to Lekko to develop with lekkos.

For the client initialization to successfully connect to Lekko, you must provide the following environment variables when running the project:

- `LEKKO_API_KEY`: A Lekko API key. Can either be generated throughout `lekko init` or on the web UI.
- `LEKKO_REPOSITORY_OWNER`: The GitHub owner of your lekko repository (see `.lekko`)
- `LEKKO_REPOSITORY_NAME`: The name of your lekko repository (see `.lekko`)

You can either use libraries such as [godotenv](https://github.com/joho/godotenv), pass them in directly when executing the project binary, or set them through tools that you use already for building/running your project (e.g. [Docker](https://docs.docker.com/compose/environment-variables/)).

> [!NOTE]
> We're working on simplifying and reducing the number of environment variables needed to connect to Lekko. Stay tuned!

Note that even if the environment variables are missing or malformed, your application will run without issues. However, the SDK will not be connected to Lekko and so the static fallback code will be used instead for evaluation.

## Error handling

Lekko's tools are designed with safety and developer experience in mind.

As you can see in the above examples, initializing the Lekko SDK client and calling lekkos does not require any error handling. This is because there is always a safe default to fall back to - the original lekko code which is included at build time as a [static fallback](https://docs.lekko.com/static-fallback).

If you have any concerns about performance or stability, you can reference our docs [here](https://docs.lekko.com/performance-reliability).
