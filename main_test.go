package codectrl_test

import (
	"github.com/STBoyden/codectrl-go-logger"
	"testing"
)

func TestSendLog(t *testing.T) {
	logger := codectrl.NewLogger()

	result, err := logger.Log("Normal log")

	if err != nil {
		t.Error(err)
	}

	t.Log(result)
}

func TestSendLogIf(t *testing.T) {
	logger := codectrl.NewLogger()

	result1, err1 := logger.LogIf("Log if condition is true.", func(params ...struct{}) bool {
		return true
	})

	_, err2 := logger.LogIf("This shouldn't log", func(params ...struct{}) bool {
		return false
	})

	if err1 != nil {
		t.Error(err1)
	}

	if err2 == nil {
		t.Error(err2, "This should have not been logged")
	}

	t.Log(result1)
}

func TestSendLogWhenEnv(t *testing.T) {
	logger := codectrl.NewLogger()

	result, err := logger.LogWhenEnv("Log if environment variable is true.")

	if err != nil {
		t.Error(err)
	}

	t.Log(result)
}
