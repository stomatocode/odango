package services

import (
	"database/sql"
	"fmt"
	"time"

	"o-dan-go/models"

	_ "github.com/mattn/go-sqlite3"
)

type DatabaseService struct {
	db *sql.DB
}

// NewDatabaseService creates a new database service instance
func NewDatabaseService(dbPath string) (*DatabaseService, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	service := &DatabaseService{db: db}

	// Create tables if they don't exist
	if err := service.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return service, nil
}

// Close closes the database connection
func (ds *DatabaseService) Close() error {
	return ds.db.Close()
}

// createTables creates the necessary tables for storing CDR reports
func (ds *DatabaseService) createTables() error {
	// Table for storing CDR summaries
	createCDRSummaryTable := `
	CREATE TABLE IF NOT EXISTS cdr_summaries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cdr_id TEXT NOT NULL UNIQUE,
		domain TEXT,
		call_direction INTEGER,
		call_start_time TEXT,
		call_duration_seconds INTEGER,
		orig_user TEXT,
		term_user TEXT,
		orig_caller_id INTEGER,
		term_caller_id INTEGER,
		disconnect_reason TEXT,
		field_count INTEGER,
		has_transcription BOOLEAN DEFAULT 0,
		has_sentiment BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Table for storing raw CDR data (all fields)
	createRawCDRTable := `
	CREATE TABLE IF NOT EXISTS raw_cdrs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cdr_id TEXT NOT NULL UNIQUE,
		raw_json TEXT NOT NULL,
		field_count INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Table for storing generated reports
	createReportsTable := `
	CREATE TABLE IF NOT EXISTS reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		report_name TEXT NOT NULL,
		report_type TEXT NOT NULL,
		parameters TEXT,
		result_data TEXT NOT NULL,
		record_count INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Execute table creation queries
	queries := []string{createCDRSummaryTable, createRawCDRTable, createReportsTable}

	for _, query := range queries {
		if _, err := ds.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// StoreCDRSummary stores a processed CDR summary
func (ds *DatabaseService) StoreCDRSummary(cdr *models.FlexibleCDR) error {
	startTime, _ := cdr.GetCallStartTime()

	query := `
	INSERT OR REPLACE INTO cdr_summaries (
		cdr_id, domain, call_direction, call_start_time, call_duration_seconds,
		orig_user, term_user, orig_caller_id, term_caller_id, disconnect_reason,
		field_count, has_transcription, has_sentiment
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := ds.db.Exec(query,
		cdr.GetID(),
		cdr.GetDomain(),
		cdr.GetCallDirection(),
		startTime.Format("2006-01-02 15:04:05"),
		cdr.GetCallDuration(),
		cdr.GetOrigUser(),
		cdr.GetTermUser(),
		cdr.GetOrigCallerID(),
		cdr.GetTermCallerID(),
		cdr.GetDisconnectReason(),
		len(cdr.GetFieldNames()),
		cdr.HasTranscriptionData(),
		cdr.HasSentimentData(),
	)

	return err
}

// StoreRawCDR stores the complete raw CDR data as JSON
func (ds *DatabaseService) StoreRawCDR(cdr *models.FlexibleCDR, rawJSON string) error {
	query := `
	INSERT OR REPLACE INTO raw_cdrs (cdr_id, raw_json, field_count)
	VALUES (?, ?, ?)`

	_, err := ds.db.Exec(query, cdr.GetID(), rawJSON, len(cdr.GetFieldNames()))
	return err
}

// GetCDRSummaries retrieves stored CDR summaries with optional filters
func (ds *DatabaseService) GetCDRSummaries(domain string, limit int) ([]CDRSummary, error) {
	query := `
	SELECT cdr_id, domain, call_direction, call_start_time, call_duration_seconds,
		   orig_user, term_user, orig_caller_id, term_caller_id, disconnect_reason,
		   field_count, has_transcription, has_sentiment, created_at
	FROM cdr_summaries`

	args := []interface{}{}

	if domain != "" {
		query += " WHERE domain = ?"
		args = append(args, domain)
	}

	query += " ORDER BY call_start_time DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := ds.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []CDRSummary
	for rows.Next() {
		var summary CDRSummary
		err := rows.Scan(
			&summary.CdrID, &summary.Domain, &summary.CallDirection,
			&summary.CallStartTime, &summary.CallDurationSeconds,
			&summary.OrigUser, &summary.TermUser, &summary.OrigCallerID,
			&summary.TermCallerID, &summary.DisconnectReason,
			&summary.FieldCount, &summary.HasTranscription,
			&summary.HasSentiment, &summary.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// StoreReport stores a generated report
func (ds *DatabaseService) StoreReport(name, reportType, parameters, resultData string, recordCount int) error {
	query := `
	INSERT INTO reports (report_name, report_type, parameters, result_data, record_count)
	VALUES (?, ?, ?, ?, ?)`

	_, err := ds.db.Exec(query, name, reportType, parameters, resultData, recordCount)
	return err
}

// GetReports retrieves stored reports
func (ds *DatabaseService) GetReports(reportType string, limit int) ([]Report, error) {
	query := `
	SELECT id, report_name, report_type, parameters, result_data, record_count, created_at
	FROM reports`

	args := []interface{}{}

	if reportType != "" {
		query += " WHERE report_type = ?"
		args = append(args, reportType)
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := ds.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		var report Report
		err := rows.Scan(
			&report.ID, &report.Name, &report.Type, &report.Parameters,
			&report.ResultData, &report.RecordCount, &report.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}

	return reports, nil
}

// Supporting structs for database operations
type CDRSummary struct {
	CdrID               string    `json:"cdr_id"`
	Domain              string    `json:"domain"`
	CallDirection       int       `json:"call_direction"`
	CallStartTime       string    `json:"call_start_time"`
	CallDurationSeconds int       `json:"call_duration_seconds"`
	OrigUser            string    `json:"orig_user"`
	TermUser            string    `json:"term_user"`
	OrigCallerID        int64     `json:"orig_caller_id"`
	TermCallerID        int64     `json:"term_caller_id"`
	DisconnectReason    string    `json:"disconnect_reason"`
	FieldCount          int       `json:"field_count"`
	HasTranscription    bool      `json:"has_transcription"`
	HasSentiment        bool      `json:"has_sentiment"`
	CreatedAt           time.Time `json:"created_at"`
}

type Report struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Parameters  string    `json:"parameters"`
	ResultData  string    `json:"result_data"`
	RecordCount int       `json:"record_count"`
	CreatedAt   time.Time `json:"created_at"`
}
