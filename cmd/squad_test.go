package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplySquadFlags_RegistersExpectedFlags(t *testing.T) {
	c := &cobra.Command{Use: "test"}
	var v SquadFlags
	AddSquadFlags(c, &v)

	assert.NotNil(t, c.Flags().Lookup("squad-id"), "expected --squad-id to be registered")
	assert.NotNil(t, c.Flags().Lookup("squad-name"), "expected --squad-name to be registered")
	assert.NotNil(t, c.Flags().Lookup("squad-role"), "expected --squad-role to be registered")
}

func TestApplySquadFlags_AbsentWhenNotChanged(t *testing.T) {
	c := &cobra.Command{Use: "test"}
	var v SquadFlags
	AddSquadFlags(c, &v)

	body := map[string]interface{}{}
	ApplySquadFlags(c, body, v)

	assert.NotContains(t, body, "squadId")
	assert.NotContains(t, body, "squadName")
	assert.NotContains(t, body, "squadRole")
}

func TestApplySquadFlags_PresentWhenSquadIDOnly(t *testing.T) {
	c := &cobra.Command{Use: "test"}
	var v SquadFlags
	AddSquadFlags(c, &v)
	require.NoError(t, c.Flags().Set("squad-id", "sq-1"))

	body := map[string]interface{}{}
	ApplySquadFlags(c, body, v)

	assert.Equal(t, "sq-1", body["squadId"])
	assert.NotContains(t, body, "squadName")
	assert.NotContains(t, body, "squadRole")
}

func TestApplySquadFlags_PresentWhenAllChanged(t *testing.T) {
	c := &cobra.Command{Use: "test"}
	var v SquadFlags
	AddSquadFlags(c, &v)
	require.NoError(t, c.Flags().Set("squad-id", "sq-1"))
	require.NoError(t, c.Flags().Set("squad-name", "Alpha"))
	require.NoError(t, c.Flags().Set("squad-role", "planner"))

	body := map[string]interface{}{}
	ApplySquadFlags(c, body, v)

	assert.Equal(t, "sq-1", body["squadId"])
	assert.Equal(t, "Alpha", body["squadName"])
	assert.Equal(t, "planner", body["squadRole"])
}
