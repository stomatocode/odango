// services/database.go
// Simple database layer to store reports for later reference

package services

import (
	"database/sql"
	"encoding/json"
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

// createTables creates the simplified MVP-focused tables
func (ds *DatabaseService) createTables() error {
	// CDR Summaries - core processed CDR data
	createCDRSummaryTable := `
	CREATE TABLE IF NOT EXISTS cdr_summaries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cdr_id TEXT NOT NULL UNIQUE,
		domain TEXT,
		call_direction INTEGER,
		call_start_time DATETIME,
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

	// Search Sessions - simplified session tracking for user workflow
	createSearchSessionsTable := `
	CREATE TABLE IF NOT EXISTS search_sessions (
		session_id TEXT PRIMARY KEY,
		search_criteria TEXT NOT NULL,  -- JSON of search parameters
		total_cdrs INTEGER DEFAULT 0,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Generated Reports - stores user-generated reports
	createReportsTable := `
	CREATE TABLE IF NOT EXISTS reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT,
		report_name TEXT NOT NULL,
		report_type TEXT NOT NULL,      -- summary, detailed, custom
		report_data TEXT NOT NULL,      -- CSV, JSON, or HTML content
		record_count INTEGER DEFAULT 0,
		file_size_bytes INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES search_sessions(session_id)
	);`

	// Execute table creation
	queries := []string{
		createCDRSummaryTable,
		createSearchSessionsTable,
		createReportsTable,
	}

	for _, query := range queries {
		if _, err := ds.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create basic indexes for performance
	return ds.createIndexes()
}

// createIndexes creates minimal indexes for MVP performance
func (ds *DatabaseService) createIndexes() error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_cdr_summaries_domain ON cdr_summaries(domain)`,
		`CREATE INDEX IF NOT EXISTS idx_cdr_summaries_start_time ON cdr_summaries(call_start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_search_sessions_start_time ON search_sessions(start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_reports_session_id ON reports(session_id)`,
	}

	for _, index := range indexes {
		if _, err := ds.db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// StoreCDRSummary stores a processed CDR summary (core MVP function)
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
		startTime,
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

// StoreSearchSession stores a simplified search session for user workflow
func (ds *DatabaseService) StoreSearchSession(sessionID string, criteria CDRSearchCriteria, totalCDRs int) error {
	criteriaJSON, _ := json.Marshal(criteria)

	query := `
	INSERT OR REPLACE INTO search_sessions (
		session_id, search_criteria, total_cdrs, start_time, end_time
	) VALUES (?, ?, ?, ?, ?)`

	_, err := ds.db.Exec(query,
		sessionID,
		string(criteriaJSON),
		totalCDRs,
		time.Now(),
		time.Now(),
	)

	return err
}

// GetCDRSummaries retrieves CDR summaries with simple filtering (core MVP function)
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

// GenerateSimpleReport creates a comprehensive but simple report from stored CDRs
func (ds *DatabaseService) GenerateSimpleReport(sessionID, reportName string, criteria ReportCriteria) (*SimpleReport, error) {
	// Build query based on criteria
	query := `
	SELECT cdr_id, domain, call_direction, call_start_time, call_duration_seconds,
		   orig_user, term_user, disconnect_reason, has_transcription, has_sentiment
	FROM cdr_summaries WHERE 1=1`

	args := []interface{}{}

	if criteria.Domain != "" {
		query += " AND domain = ?"
		args = append(args, criteria.Domain)
	}

	if !criteria.StartDate.IsZero() {
		query += " AND call_start_time >= ?"
		args = append(args, criteria.StartDate)
	}

	if !criteria.EndDate.IsZero() {
		query += " AND call_start_time <= ?"
		args = append(args, criteria.EndDate)
	}

	query += " ORDER BY call_start_time DESC"

	if criteria.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, criteria.Limit)
	}

	rows, err := ds.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Generate comprehensive but simple report
	report := &SimpleReport{
		SessionID:   sessionID,
		Name:        reportName,
		GeneratedAt: time.Now(),
		Totals:      ReportTotals{},
		Records:     []ReportRecord{},
	}

	var totalDuration int
	var inboundCount, outboundCount int
	var transcriptionCount, sentimentCount int

	for rows.Next() {
		var record ReportRecord
		var hasTranscription, hasSentiment bool

		err := rows.Scan(
			&record.CdrID, &record.Domain, &record.CallDirection,
			&record.CallStartTime, &record.CallDurationSeconds,
			&record.OrigUser, &record.TermUser, &record.DisconnectReason,
			&hasTranscription, &hasSentiment,
		)
		if err != nil {
			return nil, err
		}

		// Calculate totals
		totalDuration += record.CallDurationSeconds
		if record.CallDirection == 1 {
			inboundCount++
		} else {
			outboundCount++
		}
		if hasTranscription {
			transcriptionCount++
		}
		if hasSentiment {
			sentimentCount++
		}

		report.Records = append(report.Records, record)
	}

	// Set comprehensive totals
	report.Totals = ReportTotals{
		TotalCalls:             len(report.Records),
		TotalDurationSeconds:   totalDuration,
		InboundCalls:           inboundCount,
		OutboundCalls:          outboundCount,
		CallsWithTranscription: transcriptionCount,
		CallsWithSentiment:     sentimentCount,
		AverageDurationSeconds: 0,
	}

	if len(report.Records) > 0 {
		report.Totals.AverageDurationSeconds = totalDuration / len(report.Records)
	}

	return report, nil
}

// StoreReport saves a generated report to database
func (ds *DatabaseService) StoreReport(report *SimpleReport, format string) error {
	var reportData string
	var err error

	// Convert report to requested format
	switch format {
	case "json":
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		reportData = string(data)
	case "csv":
		reportData, err = ds.convertToCSV(report)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	query := `
	INSERT INTO reports (session_id, report_name, report_type, report_data, record_count, file_size_bytes)
	VALUES (?, ?, ?, ?, ?, ?)`

	_, err = ds.db.Exec(query,
		report.SessionID,
		report.Name,
		format,
		reportData,
		report.Totals.TotalCalls,
		len(reportData),
	)

	return err
}

// GetStoredReports retrieves previously generated reports
func (ds *DatabaseService) GetStoredReports(sessionID string, limit int) ([]StoredReport, error) {
	query := `
	SELECT id, session_id, report_name, report_type, record_count, file_size_bytes, created_at
	FROM reports`

	args := []interface{}{}

	if sessionID != "" {
		query += " WHERE session_id = ?"
		args = append(args, sessionID)
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

	var reports []StoredReport
	for rows.Next() {
		var report StoredReport
		err := rows.Scan(
			&report.ID, &report.SessionID, &report.Name,
			&report.Type, &report.RecordCount, &report.FileSizeBytes,
			&report.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}

	return reports, nil
}

// convertToCSV converts a SimpleReport to CSV format
func (ds *DatabaseService) convertToCSV(report *SimpleReport) (string, error) {
	csv := "CDR_ID,Domain,Call_Direction,Start_Time,Duration_Seconds,Orig_User,Term_User,Disconnect_Reason\n"

	for _, record := range report.Records {
		csv += fmt.Sprintf("%s,%s,%d,%s,%d,%s,%s,%s\n",
			record.CdrID,
			record.Domain,
			record.CallDirection,
			record.CallStartTime.Format("2006-01-02 15:04:05"),
			record.CallDurationSeconds,
			record.OrigUser,
			record.TermUser,
			record.DisconnectReason,
		)
	}

	return csv, nil
}

// Supporting structs for simplified MVP database operations
type CDRSummary struct {
	CdrID               string    `json:"cdr_id"`
	Domain              string    `json:"domain"`
	CallDirection       int       `json:"call_direction"`
	CallStartTime       time.Time `json:"call_start_time"`
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

type ReportCriteria struct {
	Domain    string    `json:"domain"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Limit     int       `json:"limit"`
}

type SimpleReport struct {
	SessionID   string         `json:"session_id"`
	Name        string         `json:"name"`
	GeneratedAt time.Time      `json:"generated_at"`
	Totals      ReportTotals   `json:"totals"`
	Records     []ReportRecord `json:"records"`
}

type ReportTotals struct {
	TotalCalls             int `json:"total_calls"`
	TotalDurationSeconds   int `json:"total_duration_seconds"`
	InboundCalls           int `json:"inbound_calls"`
	OutboundCalls          int `json:"outbound_calls"`
	CallsWithTranscription int `json:"calls_with_transcription"`
	CallsWithSentiment     int `json:"calls_with_sentiment"`
	AverageDurationSeconds int `json:"average_duration_seconds"`
}

type ReportRecord struct {
	CdrID               string    `json:"cdr_id"`
	Domain              string    `json:"domain"`
	CallDirection       int       `json:"call_direction"`
	CallStartTime       time.Time `json:"call_start_time"`
	CallDurationSeconds int       `json:"call_duration_seconds"`
	OrigUser            string    `json:"orig_user"`
	TermUser            string    `json:"term_user"`
	DisconnectReason    string    `json:"disconnect_reason"`
}

type StoredReport struct {
	ID            int       `json:"id"`
	SessionID     string    `json:"session_id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	RecordCount   int       `json:"record_count"`
	FileSizeBytes int       `json:"file_size_bytes"`
	CreatedAt     time.Time `json:"created_at"`
}
