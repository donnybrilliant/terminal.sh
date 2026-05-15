package services

import (
	"path/filepath"
	"testing"

	"terminal-sh/database"
	"terminal-sh/models"

	"github.com/google/uuid"
)

func TestHasCompletedObjectiveUnknownTypeWithoutToolIsIncomplete(t *testing.T) {
	db := newTestDatabase(t)
	tracker := NewActionTracker(db)

	userID := uuid.New()
	objective := models.MissionObjective{
		ID:          1,
		Type:        "unknown_objective",
		Description: "Unknown objective without validation fields",
	}

	if tracker.HasCompletedObjective(userID, "mission-1", objective) {
		t.Fatal("expected unknown objective without a tool to remain incomplete")
	}
}

func TestHasCompletedObjectiveUnknownTypeWithToolRequiresRecordedAction(t *testing.T) {
	db := newTestDatabase(t)
	tracker := NewActionTracker(db)

	userID := uuid.New()
	objective := models.MissionObjective{
		ID:          1,
		Type:        "unknown_objective",
		Description: "Unknown objective with tool fallback",
		Tool:        "password_cracker",
	}

	if tracker.HasCompletedObjective(userID, "mission-1", objective) {
		t.Fatal("expected unknown objective with a tool to remain incomplete before the tool is used")
	}

	if err := db.Create(&models.TrackedAction{
		UserID:     userID,
		ActionType: models.ActionToolUse,
		ToolName:   "password_cracker",
		MissionID:  "mission-1",
	}).Error; err != nil {
		t.Fatalf("failed to create tracked action: %v", err)
	}

	if !tracker.HasCompletedObjective(userID, "mission-1", objective) {
		t.Fatal("expected unknown objective with a tool to complete after matching tool use")
	}
}

func newTestDatabase(t *testing.T) *database.Database {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "terminal-test.db")
	db, err := database.NewDB(dbPath, "")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close test database: %v", err)
		}
	})

	return db
}
