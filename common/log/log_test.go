package log

import (
	"testing"
)

func TestLogToFile(t *testing.T) {
	log, err := New()
	if err != nil {
		t.Errorf("New error %s", err)
	}
	log.Info("test")

	log2, err := New("/tmp/ss.log")
	if err != nil {
		t.Errorf("New with filename error %s", err)
	}

	log2.Debug("test")
	log2.Info("info")
}
