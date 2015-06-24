package roll

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

// -- Test helpers

type CustomError struct {
	s string
}

func (e *CustomError) Error() string {
	return e.s
}

func setup() {
	Token = os.Getenv("TOKEN")
	Environment = "test"
}

// -- Tests

func TestErrorClass(t *testing.T) {
	errors := map[string]error{
		"{508e076d}":       fmt.Errorf("Something is broken!"),
		"roll.CustomError": &CustomError{"Terrible mistakes were made."},
	}

	for expected, err := range errors {
		if errorClass(err) != expected {
			t.Error("Got:", errorClass(err), "Expected:", expected)
		}
	}
}

func TestCritical(t *testing.T) {
	setup()
	uuid, err := Critical(errors.New("global critical"), map[string]string{"extras": "true"})
	if err != nil {
		t.Error(err)
	}
	if len(uuid) != 32 {
		t.Errorf("expected UUID, got: %#v", uuid)
	}
}

func TestError(t *testing.T) {
	setup()
	uuid, err := Error(errors.New("global error"), map[string]string{"extras": "true"})
	if err != nil {
		t.Error(err)
	}
	if len(uuid) != 32 {
		t.Errorf("expected UUID, got: %#v", uuid)
	}
}

func TestWarning(t *testing.T) {
	setup()
	uuid, err := Warning(errors.New("global warning"), map[string]string{"extras": "true"})
	if err != nil {
		t.Error(err)
	}
	if len(uuid) != 32 {
		t.Errorf("expected UUID, got: %#v", uuid)
	}
}

func TestInfo(t *testing.T) {
	setup()
	uuid, err := Info("global info", map[string]string{"extras": "true"})
	if err != nil {
		t.Error(err)
	}
	if len(uuid) != 32 {
		t.Errorf("expected UUID, got: %#v", uuid)
	}
}

func TestDebug(t *testing.T) {
	setup()
	uuid, err := Debug("global debug", map[string]string{"extras": "true"})
	if err != nil {
		t.Error(err)
	}
	if len(uuid) != 32 {
		t.Errorf("expected UUID, got: %#v", uuid)
	}
}

func TestRollbarClientCritical(t *testing.T) {
	client := New(os.Getenv("TOKEN"), "test")

	uuid, err := client.Critical(errors.New("new client critical"), map[string]string{"extras": "true"})
	if err != nil {
		t.Error(err)
	}
	if len(uuid) != 32 {
		t.Errorf("expected UUID, got: %#v", uuid)
	}
}

func TestRollbarClientError(t *testing.T) {
	client := New(os.Getenv("TOKEN"), "test")

	uuid, err := client.Error(errors.New("new client error"), map[string]string{"extras": "true"})
	if err != nil {
		t.Error(err)
	}
	if len(uuid) != 32 {
		t.Errorf("expected UUID, got: %#v", uuid)
	}
}

func TestRollbarClientWarning(t *testing.T) {
	client := New(os.Getenv("TOKEN"), "test")

	uuid, err := client.Warning(errors.New("new client warning"), map[string]string{"extras": "true"})
	if err != nil {
		t.Error(err)
	}
	if len(uuid) != 32 {
		t.Errorf("expected UUID, got: %#v", uuid)
	}
}

func TestRollbarClientInfo(t *testing.T) {
	client := New(os.Getenv("TOKEN"), "test")

	uuid, err := client.Info("new client info", map[string]string{"extras": "true"})
	if err != nil {
		t.Error(err)
	}
	if len(uuid) != 32 {
		t.Errorf("expected UUID, got: %#v", uuid)
	}
}

func TestRollbarClientDebug(t *testing.T) {
	client := New(os.Getenv("TOKEN"), "test")

	uuid, err := client.Debug("new client debug", map[string]string{"extras": "true"})
	if err != nil {
		t.Error(err)
	}
	if len(uuid) != 32 {
		t.Errorf("expected UUID, got: %#v", uuid)
	}
}
