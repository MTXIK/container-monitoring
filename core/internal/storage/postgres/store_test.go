package postgres

import (
	"database/sql"
	"testing"
)

func TestAlertRuleTargetIDAllowsNullForGlobalRules(t *testing.T) {
	targetID := alertRuleTargetID(sql.NullString{})
	if targetID != "" {
		t.Fatalf("target id = %q, want empty string", targetID)
	}
}
