// services/discovery_database.go
// Comprehensive database service for storing CDR discovery sessions and composite reports

package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"o-dan-go/models"

	_ "github.com/mattn/go-sqlite3"
)

type DiscoveryDatabaseService struct {
	db *sql.DB
}

// NewDiscoveryDatabaseService creates a new discovery-focused database service
func NewDiscoveryDatabaseService(dbPath string) (*DiscoveryDatabaseService, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	service := &DiscoveryDatabaseService{db: db}

	// Create tables if they don't exist
	if err := service.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return service, nil
}

// Close closes the database connection
func (dds *DiscoveryDatabaseService) Close() error {
	return dds.db.Close()
}

// createTables creates the comprehensive discovery-focused database schema
func (dds *DiscoveryDatabaseService) createTables() error {
	// Discovery Sessions - stores complete CDRDiscoveryResult metadata
	createDiscoverySessionsTable := `
	CREATE TABLE IF NOT EXISTS discovery_sessions (
		session_id TEXT PRIMARY KEY,
		search_criteria TEXT NOT NULL,        -- JSON of CDRSearchCriteria
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		total_cdrs INTEGER DEFAULT 0,
		unique_cdrs INTEGER DEFAULT 0,
		endpoints_queried INTEGER DEFAULT 0,
		successful_endpoints INTEGER DEFAULT 0,
		failed_endpoints INTEGER DEFAULT 0,
		total_query_time_ms INTEGER DEFAULT 0,
		raw_data_used BOOLEAN DEFAULT 0,
		errors TEXT,                          -- JSON array of errors
		session_metadata TEXT,               -- JSON for additional session info
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Endpoint Results - tracks individual endpoint performance within sessions
	createEndpointResultsTable := `
	CREATE TABLE IF NOT EXISTS endpoint_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		endpoint_name TEXT NOT NULL,
		endpoint_url TEXT NOT NULL,
		record_count INTEGER DEFAULT 0,
		success BOOLEAN DEFAULT 0,
		error_message TEXT,
		query_time_ms INTEGER DEFAULT 0,
		http_status INTEGER,
		raw_data_used BOOLEAN DEFAULT 0,
		parameter_count INTEGER DEFAULT 0,
		discovered_data BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES discovery_sessions(session_id)
	);`

	// Session CDRs - links raw CDR data to discovery sessions
	createSessionCDRsTable := `
	CREATE TABLE IF NOT EXISTS session_cdrs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		cdr_id TEXT NOT NULL,
		endpoint_source TEXT NOT NULL,        -- which endpoint provided this CDR
		raw_json TEXT NOT NULL,               -- complete CDR JSON data
		field_count INTEGER DEFAULT 0,
		duplicate_of TEXT,                    -- cdr_id if this is a duplicate
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES discovery_sessions(session_id),
		UNIQUE(session_id, cdr_id, endpoint_source)
	);`

	// Composite Reports - generated reports from discovery sessions
	createCompositeReportsTable := `
	CREATE TABLE IF NOT EXISTS composite_reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		report_name TEXT NOT NULL,
		report_type TEXT NOT NULL,            -- csv, json, xml, custom
		selected_fields TEXT,                 -- JSON array of field names
		filter_criteria TEXT,                -- JSON of additional filters
		output_format TEXT NOT NULL,
		report_data TEXT NOT NULL,            -- generated report content
		record_count INTEGER DEFAULT 0,
		file_size_bytes INTEGER DEFAULT 0,
		generation_time_ms INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES discovery_sessions(session_id)
	);`

	// Discovery Analytics - tracks successful discovery patterns
	createDiscoveryAnalyticsTable := `
	CREATE TABLE IF NOT EXISTS discovery_analytics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		parameter_combination TEXT NOT NULL,  -- JSON of successful parameter combo
		endpoint_name TEXT NOT NULL,
		success_count INTEGER DEFAULT 1,
		total_cdrs_found INTEGER DEFAULT 0,
		avg_query_time_ms INTEGER DEFAULT 0,
		last_successful_use DATETIME,
		discovery_value REAL DEFAULT 0.0,    -- calculated value of this combination
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// CDR Correlation - for call_id and phone number correlation (future use)
	createCDRCorrelationTable := `
	CREATE TABLE IF NOT EXISTS cdr_correlation (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		primary_cdr_id TEXT NOT NULL,
		related_cdr_id TEXT NOT NULL,
		correlation_type TEXT NOT NULL,       -- call_id, phone_match, time_proximity
		correlation_strength REAL DEFAULT 0.0,
		correlation_data TEXT,               -- JSON with correlation details
		session_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES discovery_sessions(session_id)
	);`

	// Execute table creation queries
	queries := []string{
		createDiscoverySessionsTable,
		createEndpointResultsTable,
		createSessionCDRsTable,
		createCompositeReportsTable,
		createDiscoveryAnalyticsTable,
		createCDRCorrelationTable,
	}

	for _, query := range queries {
		if _, err := dds.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create indexes for performance
	return dds.createIndexes()
}

// createIndexes creates indexes for optimal query performance
func (dds *DiscoveryDatabaseService) createIndexes() error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_discovery_sessions_start_time ON discovery_sessions(start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_endpoint_results_session_id ON endpoint_results(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_endpoint_results_endpoint_name ON endpoint_results(endpoint_name)`,
		`CREATE INDEX IF NOT EXISTS idx_session_cdrs_session_id ON session_cdrs(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_session_cdrs_cdr_id ON session_cdrs(cdr_id)`,
		`CREATE INDEX IF NOT EXISTS idx_composite_reports_session_id ON composite_reports(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_discovery_analytics_endpoint ON discovery_analytics(endpoint_name)`,
		`CREATE INDEX IF NOT EXISTS idx_cdr_correlation_primary ON cdr_correlation(primary_cdr_id)`,
	}

	for _, index := range indexes {
		if _, err := dds.db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// StoreDiscoverySession stores a complete CDRDiscoveryResult
func (dds *DiscoveryDatabaseService) StoreDiscoverySession(result *CDRDiscoveryResult) error {
	// Begin transaction for atomicity
	tx, err := dds.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Serialize search criteria and errors
	criteriaJSON, _ := json.Marshal(result.SearchCriteria)
	errorsJSON, _ := json.Marshal(result.Errors)

	// Calculate session metrics
	totalQueryTime := int64(0)
	successfulEndpoints := 0
	failedEndpoints := 0
	rawDataUsed := false

	for _, endpointResult := range result.EndpointResults {
		totalQueryTime += endpointResult.QueryTime.Milliseconds()
		if endpointResult.Success {
			successfulEndpoints++
		} else {
			failedEndpoints++
		}
		if endpointResult.RawDataUsed {
			rawDataUsed = true
		}
	}

	// Insert discovery session
	sessionQuery := `
	INSERT OR REPLACE INTO discovery_sessions (
		session_id, search_criteria, start_time, end_time, total_cdrs, unique_cdrs,
		endpoints_queried, successful_endpoints, failed_endpoints, total_query_time_ms,
		raw_data_used, errors
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.Exec(sessionQuery,
		result.SessionID,
		string(criteriaJSON),
		result.StartTime,
		result.EndTime,
		result.TotalCDRs,
		result.UniqueCDRs,
		len(result.EndpointResults),
		successfulEndpoints,
		failedEndpoints,
		totalQueryTime,
		rawDataUsed,
		string(errorsJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to insert discovery session: %w", err)
	}

	// Store endpoint results
	for _, endpointResult := range result.EndpointResults {
		err = dds.storeEndpointResult(tx, result.SessionID, endpointResult)
		if err != nil {
			return fmt.Errorf("failed to store endpoint result: %w", err)
		}
	}

	// Store CDR data
	for endpoint, cdrs := range result.CDRsByEndpoint {
		for _, cdr := range cdrs {
			err = dds.storeSessionCDR(tx, result.SessionID, endpoint, &cdr)
			if err != nil {
				return fmt.Errorf("failed to store session CDR: %w", err)
			}
		}
	}

	// Update discovery analytics
	err = dds.updateDiscoveryAnalytics(tx, result)
	if err != nil {
		return fmt.Errorf("failed to update analytics: %w", err)
	}

	return tx.Commit()
}

// storeEndpointResult stores individual endpoint performance data
func (dds *DiscoveryDatabaseService) storeEndpointResult(tx *sql.Tx, sessionID string, result EndpointResult) error {
	query := `
	INSERT INTO endpoint_results (
		session_id, endpoint_name, endpoint_url, record_count, success,
		error_message, query_time_ms, http_status, raw_data_used, parameter_count,
		discovered_data
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := tx.Exec(query,
		sessionID,
		result.EndpointName,
		result.URL,
		result.RecordCount,
		result.Success,
		result.Error,
		result.QueryTime.Milliseconds(),
		result.HTTPStatus,
		result.RawDataUsed,
		result.ParameterCount,
		result.DiscoveredData,
	)

	return err
}

// storeSessionCDR stores raw CDR data linked to session
func (dds *DiscoveryDatabaseService) storeSessionCDR(tx *sql.Tx, sessionID, endpoint string, cdr *models.FlexibleCDR) error {
	// Convert CDR to JSON
	rawJSON, err := json.Marshal(cdr.RawData)
	if err != nil {
		return err
	}

	query := `
	INSERT OR IGNORE INTO session_cdrs (
		session_id, cdr_id, endpoint_source, raw_json, field_count
	) VALUES (?, ?, ?, ?, ?)`

	_, err = tx.Exec(query,
		sessionID,
		cdr.GetID(),
		endpoint,
		string(rawJSON),
		len(cdr.GetFieldNames()),
	)

	return err
}

// updateDiscoveryAnalytics tracks successful parameter combinations
func (dds *DiscoveryDatabaseService) updateDiscoveryAnalytics(tx *sql.Tx, result *CDRDiscoveryResult) error {
	criteriaJSON, _ := json.Marshal(result.SearchCriteria)

	for _, endpointResult := range result.EndpointResults {
		if endpointResult.Success && endpointResult.RecordCount > 0 {
			query := `
			INSERT OR REPLACE INTO discovery_analytics (
				parameter_combination, endpoint_name, success_count, total_cdrs_found,
				avg_query_time_ms, last_successful_use, discovery_value
			) VALUES (?, ?, 
				COALESCE((SELECT success_count FROM discovery_analytics 
				         WHERE parameter_combination = ? AND endpoint_name = ?), 0) + 1,
				?,
				?,
				?,
				?
			)`

			discoveryValue := float64(endpointResult.RecordCount) / endpointResult.QueryTime.Seconds()

			_, err := tx.Exec(query,
				string(criteriaJSON),
				endpointResult.EndpointName,
				string(criteriaJSON),
				endpointResult.EndpointName,
				endpointResult.RecordCount,
				endpointResult.QueryTime.Milliseconds(),
				time.Now(),
				discoveryValue,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// GetDiscoverySession retrieves a stored discovery session
func (dds *DiscoveryDatabaseService) GetDiscoverySession(sessionID string) (*StoredDiscoverySession, error) {
	query := `
	SELECT session_id, search_criteria, start_time, end_time, total_cdrs, unique_cdrs,
		   endpoints_queried, successful_endpoints, failed_endpoints, total_query_time_ms,
		   raw_data_used, errors, created_at
	FROM discovery_sessions WHERE session_id = ?`

	var session StoredDiscoverySession
	var criteriaJSON, errorsJSON string

	err := dds.db.QueryRow(query, sessionID).Scan(
		&session.SessionID,
		&criteriaJSON,
		&session.StartTime,
		&session.EndTime,
		&session.TotalCDRs,
		&session.UniqueCDRs,
		&session.EndpointsQueried,
		&session.SuccessfulEndpoints,
		&session.FailedEndpoints,
		&session.TotalQueryTimeMs,
		&session.RawDataUsed,
		&errorsJSON,
		&session.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Deserialize JSON fields
	json.Unmarshal([]byte(criteriaJSON), &session.SearchCriteria)
	json.Unmarshal([]byte(errorsJSON), &session.Errors)

	return &session, nil
}

// GetSessionCDRs retrieves all CDRs for a session
func (dds *DiscoveryDatabaseService) GetSessionCDRs(sessionID string) ([]SessionCDR, error) {
	query := `
	SELECT cdr_id, endpoint_source, raw_json, field_count, created_at
	FROM session_cdrs WHERE session_id = ? ORDER BY created_at`

	rows, err := dds.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cdrs []SessionCDR
	for rows.Next() {
		var cdr SessionCDR
		err := rows.Scan(&cdr.CdrID, &cdr.EndpointSource, &cdr.RawJSON, &cdr.FieldCount, &cdr.CreatedAt)
		if err != nil {
			return nil, err
		}
		cdrs = append(cdrs, cdr)
	}

	return cdrs, nil
}

// StoreCompositeReport stores a generated report from a discovery session
func (dds *DiscoveryDatabaseService) StoreCompositeReport(sessionID string, report CompositeReport) error {
	selectedFieldsJSON, _ := json.Marshal(report.SelectedFields)
	filterCriteriaJSON, _ := json.Marshal(report.FilterCriteria)

	query := `
	INSERT INTO composite_reports (
		session_id, report_name, report_type, selected_fields, filter_criteria,
		output_format, report_data, record_count, file_size_bytes, generation_time_ms
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := dds.db.Exec(query,
		sessionID,
		report.Name,
		report.Type,
		string(selectedFieldsJSON),
		string(filterCriteriaJSON),
		report.OutputFormat,
		report.Data,
		report.RecordCount,
		len(report.Data),
		report.GenerationTimeMs,
	)

	return err
}

// GetDiscoveryAnalytics retrieves successful discovery patterns
func (dds *DiscoveryDatabaseService) GetDiscoveryAnalytics(limit int) ([]DiscoveryAnalytic, error) {
	query := `
	SELECT parameter_combination, endpoint_name, success_count, total_cdrs_found,
		   avg_query_time_ms, last_successful_use, discovery_value
	FROM discovery_analytics 
	ORDER BY discovery_value DESC, success_count DESC
	LIMIT ?`

	rows, err := dds.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analytics []DiscoveryAnalytic
	for rows.Next() {
		var analytic DiscoveryAnalytic
		var paramJSON string
		err := rows.Scan(
			&paramJSON,
			&analytic.EndpointName,
			&analytic.SuccessCount,
			&analytic.TotalCDRsFound,
			&analytic.AvgQueryTimeMs,
			&analytic.LastSuccessfulUse,
			&analytic.DiscoveryValue,
		)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(paramJSON), &analytic.ParameterCombination)
		analytics = append(analytics, analytic)
	}

	return analytics, nil
}

// Supporting structs for database operations
type StoredDiscoverySession struct {
	SessionID           string            `json:"session_id"`
	SearchCriteria      CDRSearchCriteria `json:"search_criteria"`
	StartTime           time.Time         `json:"start_time"`
	EndTime             time.Time         `json:"end_time"`
	TotalCDRs           int               `json:"total_cdrs"`
	UniqueCDRs          int               `json:"unique_cdrs"`
	EndpointsQueried    int               `json:"endpoints_queried"`
	SuccessfulEndpoints int               `json:"successful_endpoints"`
	FailedEndpoints     int               `json:"failed_endpoints"`
	TotalQueryTimeMs    int64             `json:"total_query_time_ms"`
	RawDataUsed         bool              `json:"raw_data_used"`
	Errors              []string          `json:"errors"`
	CreatedAt           time.Time         `json:"created_at"`
}

type SessionCDR struct {
	CdrID          string    `json:"cdr_id"`
	EndpointSource string    `json:"endpoint_source"`
	RawJSON        string    `json:"raw_json"`
	FieldCount     int       `json:"field_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type CompositeReport struct {
	Name             string      `json:"name"`
	Type             string      `json:"type"`
	SelectedFields   []string    `json:"selected_fields"`
	FilterCriteria   interface{} `json:"filter_criteria"`
	OutputFormat     string      `json:"output_format"`
	Data             string      `json:"data"`
	RecordCount      int         `json:"record_count"`
	GenerationTimeMs int64       `json:"generation_time_ms"`
}

type DiscoveryAnalytic struct {
	ParameterCombination CDRSearchCriteria `json:"parameter_combination"`
	EndpointName         string            `json:"endpoint_name"`
	SuccessCount         int               `json:"success_count"`
	TotalCDRsFound       int               `json:"total_cdrs_found"`
	AvgQueryTimeMs       int64             `json:"avg_query_time_ms"`
	LastSuccessfulUse    time.Time         `json:"last_successful_use"`
	DiscoveryValue       float64           `json:"discovery_value"`
}
