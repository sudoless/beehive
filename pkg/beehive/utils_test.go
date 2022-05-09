package beehive

import (
	"context"
	"testing"
)

func Test_Next_nil(t *testing.T) {
	t.Parallel()

	out := Next(context.Background(), nil)
	if out != nil {
		t.Errorf("out should be nil")
	}
}

func Test_ResponseWriter_nil(t *testing.T) {
	t.Parallel()

	out := ResponseWriter(context.Background())
	if out != nil {
		t.Errorf("out should be nil")
	}
}
