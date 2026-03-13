package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPricingUpdate(t *testing.T) {
	var receivedBody map[string]interface{}
	var putMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/pricing") {
			// Return pricing wrapper with dimensions
			fmt.Fprint(w, `{"modelId":"mdl-1","dimensions":[
				{"id":"dim-1","billingUnit":"PER_TOKEN","modality":"TEXT","costType":"TEXT_TOKEN_INPUT","unitPrice":0.003,"isGlobal":false}
			]}`)
			return
		}
		if r.Method == "PUT" {
			putMethod = r.Method
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			fmt.Fprint(w, `{"id":"dim-1","billingUnit":"PER_TOKEN","modality":"TEXT","costType":"TEXT_TOKEN_INPUT","unitPrice":0.005,"isGlobal":false}`)
			return
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"mdl-1", "dim-1", "--price", "0.005"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "PUT", putMethod)
	assert.Equal(t, 0.005, receivedBody["unitPrice"])
	assert.Contains(t, buf.String(), "dim-1")
}

func TestPricingUpdatePath(t *testing.T) {
	var putPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			fmt.Fprint(w, `{"modelId":"my-model","dimensions":[
				{"id":"my-dim","billingUnit":"PER_TOKEN","modality":"TEXT","costType":"TEXT_TOKEN_INPUT","unitPrice":0.003,"isGlobal":false}
			]}`)
			return
		}
		if r.Method == "PUT" {
			putPath = r.URL.Path
			fmt.Fprint(w, `{"id":"my-dim","billingUnit":"PER_TOKEN","modality":"TEXT","costType":"TEXT_TOKEN_INPUT","unitPrice":0.005,"isGlobal":false}`)
			return
		}
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingUpdateCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"my-model", "my-dim", "--price", "0.005"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/sources/ai/models/my-model/pricing/dimensions/my-dim", putPath)
}

func TestPricingUpdateNoFields(t *testing.T) {
	var buf bytes.Buffer
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newPricingUpdateCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"mdl-1", "dim-1"})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified to update")
}
