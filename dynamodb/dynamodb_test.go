package dynamodb_test

import (
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	awsdynamodb "github.com/aws/aws-sdk-go/service/dynamodb"

	"github.com/philippgille/gokv/dynamodb"
	"github.com/philippgille/gokv/encoding"
	"github.com/philippgille/gokv/test"
)

// For "DynamoDB local" Docker container.
// See https://hub.docker.com/r/amazon/dynamodb-local/.
var customEndpoint = "http://localhost:8000"

// TestConnection only tests the connection to the local DynamoDB, allowing to work on connection options with `go test -run TestConnection .` for example.
func TestConnection(t *testing.T) {
	if !checkConnection() {
		t.Skip("No connection to DynamoDB could be established. Probably not running in a proper test environment.")
	}
}

// TestClient tests if reading from, writing to and deleting from the store works properly.
// A struct is used as value. See TestTypes() for a test that is simpler but tests all types.
func TestClient(t *testing.T) {
	// Test with JSON
	t.Run("JSON", func(t *testing.T) {
		client := createClient(t, encoding.JSON)
		test.TestStore(client, t)
	})

	// Test with gob
	t.Run("gob", func(t *testing.T) {
		client := createClient(t, encoding.Gob)
		test.TestStore(client, t)
	})
}

// TestTypes tests if setting and getting values works with all Go types.
func TestTypes(t *testing.T) {
	// Test with JSON
	t.Run("JSON", func(t *testing.T) {
		client := createClient(t, encoding.JSON)
		test.TestTypes(client, t)
	})

	// Test with gob
	t.Run("gob", func(t *testing.T) {
		client := createClient(t, encoding.Gob)
		test.TestTypes(client, t)
	})
}

// TestClientConcurrent launches a bunch of goroutines that concurrently work with the DynamoDB client.
func TestClientConcurrent(t *testing.T) {
	client := createClient(t, encoding.JSON)

	goroutineCount := 1000

	test.TestConcurrentInteractions(t, goroutineCount, client)
}

// TestErrors tests some error cases.
func TestErrors(t *testing.T) {
	// Test empty key
	client := createClient(t, encoding.JSON)
	err := client.Set("", "bar")
	if err == nil {
		t.Error("Expected an error")
	}
	_, err = client.Get("", new(string))
	if err == nil {
		t.Error("Expected an error")
	}
	err = client.Delete("")
	if err == nil {
		t.Error("Expected an error")
	}

	// Test client creation with bad options
	options := dynamodb.Options{
		AWSaccessKeyID: "foo",
	}
	_, err = dynamodb.NewClient(options)
	if err.Error() != "when passing credentials via options, you need to set BOTH AWSaccessKeyID AND AWSsecretAccessKey" {
		t.Error("An error was expected, but didn't occur.")
	}
	options = dynamodb.Options{
		AWSsecretAccessKey: "foo",
	}
	_, err = dynamodb.NewClient(options)
	if err.Error() != "when passing credentials via options, you need to set BOTH AWSaccessKeyID AND AWSsecretAccessKey" {
		t.Error("An error was expected, but didn't occur.")
	}
	// Bad credentials on actual AWS DynamoDB endpoint (no custom endpoint for local Docker container)
	options = dynamodb.Options{
		AWSaccessKeyID:     "foo",
		AWSsecretAccessKey: "bar",
		Region:             endpoints.UsWest2RegionID,
	}
	_, err = dynamodb.NewClient(options)
	if strings.Index(err.Error(), "UnrecognizedClientException: The security token included in the request is invalid.") != 0 {
		t.Errorf("An UnrecognizedClientException was expected, but it seems like it didn't occur. Instead, the error was: %v", err)
	}
}

// TestNil tests the behaviour when passing nil or pointers to nil values to some methods.
func TestNil(t *testing.T) {
	// Test setting nil

	t.Run("set nil with JSON marshalling", func(t *testing.T) {
		client := createClient(t, encoding.JSON)
		err := client.Set("foo", nil)
		if err == nil {
			t.Error("Expected an error")
		}
	})

	t.Run("set nil with Gob marshalling", func(t *testing.T) {
		client := createClient(t, encoding.Gob)
		err := client.Set("foo", nil)
		if err == nil {
			t.Error("Expected an error")
		}
	})

	// Test passing nil or pointer to nil value for retrieval

	createTest := func(codec encoding.Codec) func(t *testing.T) {
		return func(t *testing.T) {
			client := createClient(t, codec)

			// Prep
			err := client.Set("foo", test.Foo{Bar: "baz"})
			if err != nil {
				t.Error(err)
			}

			_, err = client.Get("foo", nil) // actually nil
			if err == nil {
				t.Error("An error was expected")
			}

			var i any // actually nil
			_, err = client.Get("foo", i)
			if err == nil {
				t.Error("An error was expected")
			}

			var valPtr *test.Foo // nil value
			_, err = client.Get("foo", valPtr)
			if err == nil {
				t.Error("An error was expected")
			}
		}
	}
	t.Run("get with nil / nil value parameter", createTest(encoding.JSON))
	t.Run("get with nil / nil value parameter", createTest(encoding.Gob))
}

// TestClose tests if the close method returns any errors.
func TestClose(t *testing.T) {
	client := createClient(t, encoding.JSON)
	err := client.Close()
	if err != nil {
		t.Error(err)
	}
}

// checkConnection returns true if a connection could be made, false otherwise.
func checkConnection() bool {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			CredentialsChainVerboseErrors: aws.Bool(true),
			// Local DynamoDB requires credentials, and with the shared config disabled these must be explicitly provided.
			Credentials: credentials.NewStaticCredentials("foo", "foo", ""),
			Endpoint:    aws.String(customEndpoint),
			Region:      aws.String(endpoints.EuCentral1RegionID),
		},
		Profile:           session.DefaultSharedConfigProfile,
		SharedConfigState: session.SharedConfigDisable,
	})
	if err != nil {
		log.Printf("An error occurred during testing the connection to the server: %v\n", err)
		return false
	}
	svc := awsdynamodb.New(sess)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	limit := int64(1)
	listTablesInput := awsdynamodb.ListTablesInput{
		Limit: &limit,
	}
	_, err = svc.ListTablesWithContext(timeoutCtx, &listTablesInput)
	if err != nil {
		log.Printf("An error occurred during testing the connection to the server: %v\n", err)
		return false
	}
	return true
}

func createClient(t *testing.T, codec encoding.Codec) dynamodb.Client {
	options := dynamodb.Options{
		Region:             endpoints.EuCentral1RegionID,
		AWSaccessKeyID:     "foo",
		AWSsecretAccessKey: "foo",
		CustomEndpoint:     customEndpoint,
		Codec:              codec,
	}
	client, err := dynamodb.NewClient(options)
	if err != nil {
		t.Fatal(err)
	}
	return client
}
