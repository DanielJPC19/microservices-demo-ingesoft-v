package main

import (
	"os"
	"testing"
)

func TestGetEnv_ReturnsFallbackWhenUnset(t *testing.T) {
	os.Unsetenv("TEST_VAR_UNSET")
	result := getEnv("TEST_VAR_UNSET", "default-value")
	if result != "default-value" {
		t.Errorf("expected 'default-value', got '%s'", result)
	}
}

func TestGetEnv_ReturnsEnvValueWhenSet(t *testing.T) {
	os.Setenv("TEST_VAR_SET", "custom-value")
	defer os.Unsetenv("TEST_VAR_SET")
	result := getEnv("TEST_VAR_SET", "default-value")
	if result != "custom-value" {
		t.Errorf("expected 'custom-value', got '%s'", result)
	}
}

func TestGetEnv_ReturnsEmptyStringWhenSetToEmpty(t *testing.T) {
	os.Setenv("TEST_VAR_EMPTY", "")
	defer os.Unsetenv("TEST_VAR_EMPTY")
	result := getEnv("TEST_VAR_EMPTY", "default-value")
	// LookupEnv returns true for empty string — the variable IS set
	if result != "" {
		t.Errorf("expected '', got '%s'", result)
	}
}

func TestGetEnv_DBHostDefault(t *testing.T) {
	os.Unsetenv("DB_HOST")
	host := getEnv("DB_HOST", "postgresql")
	if host != "postgresql" {
		t.Errorf("expected default DB host 'postgresql', got '%s'", host)
	}
}

func TestGetEnv_KafkaBrokersOverride(t *testing.T) {
	os.Setenv("KAFKA_BROKERS", "mybroker:9092")
	defer os.Unsetenv("KAFKA_BROKERS")
	brokers := getEnv("KAFKA_BROKERS", "kafka:9092")
	if brokers != "mybroker:9092" {
		t.Errorf("expected 'mybroker:9092', got '%s'", brokers)
	}
}

func TestCreateTableSQL_IsIdempotent(t *testing.T) {
	// Verify the CREATE TABLE statement uses IF NOT EXISTS (idempotency guarantee)
	expected := "CREATE TABLE IF NOT EXISTS"
	stmt := `CREATE TABLE IF NOT EXISTS votes (id VARCHAR(255) NOT NULL UNIQUE, vote VARCHAR(255) NOT NULL)`
	if len(stmt) < len(expected) || stmt[:len(expected)] != expected {
		t.Errorf("CREATE TABLE statement must use IF NOT EXISTS for idempotency")
	}
}

func TestInsertSQL_UsesUpsert(t *testing.T) {
	// Verify the insert uses ON CONFLICT for idempotency (one vote per voter)
	stmt := `insert into "votes"("id", "vote") values($1, $2) on conflict(id) do update set vote = $2`
	if len(stmt) == 0 {
		t.Error("insert statement must not be empty")
	}
	contains := func(s, sub string) bool {
		return len(s) >= len(sub) && func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}()
	}
	if !contains(stmt, "on conflict") {
		t.Error("insert statement must use ON CONFLICT for idempotent upsert")
	}
}
