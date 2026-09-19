package auditlog

import (
	"os"
	"testing"
	"time"

	"github.com/kkx999/KomariX/cmd/flags"
	"github.com/kkx999/KomariX/database/dbcore"
	"github.com/kkx999/KomariX/database/models"
)

func TestMain(m *testing.M) {
	flags.DatabaseType = flags.DatabaseTypeSQLite
	flags.DatabaseFile = "file:komarix_auditlog_test?mode=memory&cache=shared"
	dbcore.GetDBInstance()
	os.Exit(m.Run())
}

func resetLogs(t *testing.T) {
	t.Helper()
	db := dbcore.GetDBInstance()
	if err := db.Where("1 = 1").Delete(&models.Log{}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestCleanupAppliesAgeAndRowLimits(t *testing.T) {
	resetLogs(t)
	db := dbcore.GetDBInstance()
	now := time.Now().UTC()
	rows := []models.Log{
		{Message: "old", MsgType: "test", Time: now.Add(-10 * 24 * time.Hour)},
		{Message: "keep-1", MsgType: "test", Time: now.Add(-2 * time.Hour)},
		{Message: "keep-2", MsgType: "test", Time: now.Add(-time.Hour)},
		{Message: "keep-3", MsgType: "test", Time: now},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	deleted, err := Cleanup(7, 2)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("deleted = %d, want 2", deleted)
	}
	var remaining []models.Log
	if err := db.Order("time asc").Find(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 2 || remaining[0].Message != "keep-2" || remaining[1].Message != "keep-3" {
		t.Fatalf("unexpected remaining logs: %+v", remaining)
	}
}

func TestCleanupZeroPoliciesKeepLogs(t *testing.T) {
	resetLogs(t)
	db := dbcore.GetDBInstance()
	if err := db.Create(&models.Log{Message: "keep", MsgType: "test", Time: time.Now().UTC().Add(-365 * 24 * time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	deleted, err := Cleanup(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
}
