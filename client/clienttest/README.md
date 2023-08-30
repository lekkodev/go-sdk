
# clienttest

This package contains `clienttest`, an in-memory fake Lekko Client. `clienttest` can be used to test code that depends on the Lekko SDK in Golang.

Here is a simple example of how you may use it in a unit test:

```go
func TestLekko(t *testing.T) {
	testClient := NewTestClient().WithBool("namespace", "bool-config", true)

	result, err := testClient.GetBool(context.Background(), "namespace", "bool-config")
	assert.NoError(t, err)
	assert.True(t, result)
}
```
