package orders

import "testing"

func TestLuhnValid(t *testing.T) {
	t.Parallel()
	valid := []string{
		"79927398713",
		"4000000000000002",
		"4012888888881881",
		"4222222222222",
		"3530111333300000",
	}
	for _, s := range valid {
		if !LuhnValid(s) {
			t.Fatalf("expected valid: %s", s)
		}
	}

	invalid := []string{
		"79927398714",
		"1234567890",
		"abcdef",
		"",
		"1",
		"9111111111111111",
	}
	for _, s := range invalid {
		if LuhnValid(s) {
			t.Fatalf("expected invalid: %s", s)
		}
	}
}
