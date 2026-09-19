package public

import (
	"testing"
	"time"
)

func resetLoginFailureTrackerForTest() {
	loginFailureTracker.Lock()
	loginFailureTracker.entries = make(map[string]loginFailureState)
	loginFailureTracker.lastSweep = time.Time{}
	loginFailureTracker.Unlock()
}

func TestLoginFailureTrackerBlocksAfterTenFailures(t *testing.T) {
	resetLoginFailureTrackerForTest()
	t.Cleanup(resetLoginFailureTrackerForTest)

	ip := "203.0.113.10"
	for i := 1; i <= loginFailureLimit; i++ {
		attempt, blocked := recordLoginFailure(ip)
		if attempt != i {
			t.Fatalf("attempt = %d, want %d", attempt, i)
		}
		if i < loginFailureLimit && blocked {
			t.Fatalf("blocked too early at attempt %d", i)
		}
		if i == loginFailureLimit && !blocked {
			t.Fatal("IP was not blocked at failure limit")
		}
	}
	if blocked, remaining := loginBlocked(ip); !blocked || remaining <= 0 {
		t.Fatalf("loginBlocked = %v, remaining = %v", blocked, remaining)
	}
}

func TestClearLoginFailuresResetsState(t *testing.T) {
	resetLoginFailureTrackerForTest()
	t.Cleanup(resetLoginFailureTrackerForTest)

	ip := "203.0.113.11"
	recordLoginFailure(ip)
	clearLoginFailures(ip)
	if blocked, _ := loginBlocked(ip); blocked {
		t.Fatal("cleared IP remained blocked")
	}
	attempt, _ := recordLoginFailure(ip)
	if attempt != 1 {
		t.Fatalf("attempt after clear = %d, want 1", attempt)
	}
}
