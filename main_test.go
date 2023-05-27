package codectrl

import (
	"testing"
)

func TestSendLog(t *testing.T) {
	logger := NewLogger()

	result := logger.Log("Normal log")

	if result.IsOk() {
		requestResult := result.Ok()

		t.Log(requestResult)
	} else {
		error := result.Err()

		t.Error(error.Message)
	}
}

func TestSendLogIf(t *testing.T) {
	logger := NewLogger()

	result1 := logger.LogIf("Log if condition is true.", func(params ...struct{}) bool {
		return true
	})

	result2 := logger.LogIf("This shouldn't log", func(params ...struct{}) bool {
		return false
	})

	if result1.IsOk() {
		requestResult := result1.Ok()

		t.Log(requestResult)
	} else {
		error := result1.Err()

		t.Error(error.Message)
	}

	if result2.IsErr() {
		error := result2.Err()

		t.Log(error.Message)
	} else {
		requestResult := result2.Ok()

		t.Error(requestResult)
	}
}

func TestSendLogWhenEnv(t *testing.T) {
	logger := NewLogger()

	result := logger.LogWhenEnv("Log if environment variable is true.")

	if result.IsOk() {
		requestResult := result.Ok()

		t.Log(requestResult)
	} else {
		error := result.Err()

		t.Error(error.Message)
	}
}
