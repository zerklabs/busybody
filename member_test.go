package busybody

import "testing"

func TestNew(t *testing.T) {
	b := make([]byte, 0)

	member, err := New(b)
	if err != nil {
		t.Error(err)
	}

	t.Logf("%#v", member)
}
