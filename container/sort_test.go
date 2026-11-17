package container

import "testing"

func TestUptimeSeconds(t *testing.T) {
	order := []string{"25 weeks", "3 weeks", "2 days", "2 hours", "1 minute", "5 seconds"}
	for i := 1; i < len(order); i++ {
		if uptimeSeconds(order[i-1]) <= uptimeSeconds(order[i]) {
			t.Errorf("%q should be longer than %q", order[i-1], order[i])
		}
	}
	if uptimeSeconds("") != -1 || uptimeSeconds("-") != -1 {
		t.Error("empty uptime should give -1")
	}
}
