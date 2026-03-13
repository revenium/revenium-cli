package subscribers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSubscriber(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v2/api/subscribers", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-new", "email": "user@example.com", "firstName": "John", "lastName": "Doe"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--email", "user@example.com", "--first-name", "John", "--last-name", "Doe"})
	err := c.Execute()

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "sub-new")
	assert.Contains(t, out, "user@example.com")
	assert.Equal(t, "user@example.com", receivedBody["email"])
	assert.Equal(t, "John", receivedBody["firstName"])
	assert.Equal(t, "Doe", receivedBody["lastName"])
}

func TestCreateSubscriberEmailOnly(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": "sub-new", "email": "user@example.com"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newCreateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"--email", "user@example.com"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "user@example.com", receivedBody["email"])
	_, hasFirstName := receivedBody["firstName"]
	assert.False(t, hasFirstName, "firstName should not be sent when not specified")
	_, hasLastName := receivedBody["lastName"]
	assert.False(t, hasLastName, "lastName should not be sent when not specified")
}
