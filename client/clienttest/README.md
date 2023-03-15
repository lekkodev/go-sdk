
# clienttest

This package contains `clienttest`, an in-memory fake Lekko Client. `clienttest` can be used to test code that depends on the Lekko SDK in Golang.

Here is a simple example of how you may use it in a unit test:

```go
func TestLekko(t *testing.T) {
	testClient := NewTestClient().WithBool("bool-feature", true)

	result, err := testClient.GetBool(context.Background(), "bool-feature")
	assert.NoError(t, err)
	assert.True(t, result)
}
```
