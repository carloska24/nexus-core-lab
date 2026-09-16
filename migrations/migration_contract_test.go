package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestMSISDNLengthMigrationContract(t *testing.T) {
	up, err := os.ReadFile("000004_expand_msisdn_length.up.sql")
	if err != nil {
		t.Fatalf("missing incremental MSISDN migration: %v", err)
	}
	down, err := os.ReadFile("000004_expand_msisdn_length.down.sql")
	if err != nil {
		t.Fatalf("missing safe MSISDN rollback: %v", err)
	}

	upSQL := strings.ToUpper(string(up))
	if !strings.Contains(upSQL, "ALTER COLUMN MSISDN TYPE VARCHAR(16)") {
		t.Fatalf("UP migration must expand subscribers.msisdn to VARCHAR(16)")
	}

	downSQL := strings.ToUpper(string(down))
	for _, required := range []string{"CHAR_LENGTH(MSISDN) > 15", "RAISE EXCEPTION", "ALTER COLUMN MSISDN TYPE VARCHAR(15)"} {
		if !strings.Contains(downSQL, required) {
			t.Fatalf("DOWN migration is missing safe rollback clause %q", required)
		}
	}
	for _, forbidden := range []string{"SUBSTRING", "LEFT(", "TRUNCATE"} {
		if strings.Contains(downSQL, forbidden) {
			t.Fatalf("DOWN migration must not silently truncate MSISDN data: found %q", forbidden)
		}
	}
}
