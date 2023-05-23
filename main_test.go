package codectrl

import (
	"testing"
)

func TestSendLog(t *testing.T) {
	logger := Logger{}

	result := logger.Log("Test")

	if result.IsOk() {
		channel := result.Ok()

		t.Log(channel)
	} else {
		error := result.Err()

		t.Error(error.Message)
	}
}
