package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExitCodeFor_Nil(t *testing.T) {
	assert.Equal(t, ExitOK, ExitCodeFor(nil))
}

func TestExitCodeFor_GenericError(t *testing.T) {
	assert.Equal(t, ExitGeneral, ExitCodeFor(fmt.Errorf("something broke")))
}

func TestExitCodeFor_Auth(t *testing.T) {
	assert.Equal(t, ExitAuth, ExitCodeFor(&APIError{StatusCode: 401, Message: "unauthorized"}))
	assert.Equal(t, ExitAuth, ExitCodeFor(&APIError{StatusCode: 403, Message: "forbidden"}))
}

func TestExitCodeFor_NotFound(t *testing.T) {
	assert.Equal(t, ExitNotFound, ExitCodeFor(&APIError{StatusCode: 404, Message: "not found"}))
}

func TestExitCodeFor_Validation(t *testing.T) {
	assert.Equal(t, ExitValidation, ExitCodeFor(&APIError{StatusCode: 400, Message: "bad request"}))
	assert.Equal(t, ExitValidation, ExitCodeFor(&APIError{StatusCode: 422, Message: "unprocessable"}))
}

func TestExitCodeFor_ServerError(t *testing.T) {
	assert.Equal(t, ExitGeneral, ExitCodeFor(&APIError{StatusCode: 500, Message: "server error"}))
}
