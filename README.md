# go-sdk
Golang SDK for reading feature flags

## Example

This repo includes an example go program that retrieves a feature flag from Lekko backend. To test it, run
```
go run cmd/example/main.go --lekko-apikey=<replace-with-your-api-key>
```
Make sure to replace the API key with the one given to your organization. Also, make sure to change the owner and repo name to a repository under your organization.
Alternatively, you can build and run with docker, which may resemble how you might containerize your application and run it remotely. 
```
make dockerbuild
```
This will build a docker image locally. You can then run it with
```
docker run --name example lekko/example:latest --lekko-apikey=<replace-with-your-api-key>
```
