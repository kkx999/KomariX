package auditlog

import (
	"time"

	"github.com/kkx999/KomariX/database/dbcore"
	"github.com/kkx999/KomariX/database/models"
	"github.com/kkx999/KomariX/internal/config"
	logger "github.com/kkx999/KomariX/utils/log"
)

func Log(ip, uuid, message, msgType string) {
	db := dbcore.GetDBInstance()
	logEntry := &models.Log{
		IP:      ip,
		UUID:    uuid,
		Message: message,
		MsgType: msgType,
		Time:    time.Now().UTC(),
	}
	if err := db.Create(logEntry).Error; err != nil {
		logger.Error("audit", "failed to persist audit event", "error", err, "type", msgType)
	}
}

func EventLog(eventType, message string) {
	Log("", "", message, eventType)
}

func configuredRetention() (days int, maxRows int) {
	days, _ = config.GetAs[int](config.AuditLogRetentionDaysKey, 30)
	maxRows, _ = config.GetAs[int](config.AuditLogMaxRowsKey, 100000)
	if days < 0 {
		days = 30
	}
	if maxRows < 0 {
		maxRows = 100000
	}
	return days, maxRows
}

func Cleanup(retentionDays, maxRows int) (int64, error) {
	db := dbcore.GetDBInstance()
	var deleted int64
	if retentionDays > 0 {
		threshold := time.Now().UTC().AddDate(0, 0, -retentionDays)
		result := db.Where("time < ?", threshold).Delete(&models.Log{})
		if result.Error != nil {
			return deleted, result.Error
		}
		deleted += result.RowsAffected
	}
	if maxRows > 0 {
		for {
			var count int64
			if err := db.Model(&models.Log{}).Count(&count).Error; err != nil {
				return deleted, err
			}
			excess := count - int64(maxRows)
			if excess <= 0 {
				break
			}
			batch := excess
			if batch > 5000 {
				batch = 5000
			}
			var ids []uint
			if err := db.Model(&models.Log{}).Order("time asc").Order("id asc").Limit(int(batch)).Pluck("id", &ids).Error; err != nil {
				return deleted, err
			}
			if len(ids) == 0 {
				break
			}
			result := db.Delete(&models.Log{}, ids)
			if result.Error != nil {
				return deleted, result.Error
			}
			deleted += result.RowsAffected
		}
	}
	return deleted, nil
}

func CleanupConfigured() (int64, error) {
	days, maxRows := configuredRetention()
	return Cleanup(days, maxRows)
}

func ClearAll() (int64, error) {
	db := dbcore.GetDBInstance()
	result := db.Where("1 = 1").Delete(&models.Log{})
	return result.RowsAffected, result.Error
}

func RemoveOldLogs() {
	if _, err := CleanupConfigured(); err != nil {
		logger.ErrorArgs("audit", "Failed to clean audit logs:", err)
	}
}
