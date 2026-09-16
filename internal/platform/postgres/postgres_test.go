package postgres

import "testing"

func TestValidateCurrentSchemaVersion(t *testing.T) {
	if err := validateCurrentSchemaVersion(3, true, ExpectedSchemaVersion); err == nil {
		t.Fatal("schema version 3 must be rejected after migration 000004")
	}
	if err := validateCurrentSchemaVersion(4, true, ExpectedSchemaVersion); err != nil {
		t.Fatalf("schema version 4 must be accepted: %v", err)
	}
	if err := validateCurrentSchemaVersion(5, true, ExpectedSchemaVersion); err != nil {
		t.Fatalf("newer schema version must remain accepted: %v", err)
	}
	if err := validateCurrentSchemaVersion(0, false, ExpectedSchemaVersion); err == nil {
		t.Fatal("missing schema version must be rejected")
	}
}
