# go-sdk
Golang sdk for reading feature flags

## Example

This repo includes an example go program that retrieves a feature flag from the Lekko backend. In order to test it, run
```
go run cmd/example/main.go --lekko-apikey=<replace-with-your-api-key>
```
Make sure to replace the api key with the one given to your organization.
Alternatively, you can build and run with docker, which may resemble how you might containerize your application and run it remotely. 
```
make dockerbuild
```
This will build a docker image locally. You can then run it with
```
docker run --name example lekko/example:latest --lekko-apikey=<replace-with-your-api-key>
```
