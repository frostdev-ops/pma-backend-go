package database

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// QueryOptimizer defines the interface for database query optimization
type QueryOptimizer interface {
	OptimizeQuery(query string, params []interface{}) (string, []interface{}, error)
	AddIndex(table, column string) error
	AnalyzeQuery(query string) (*QueryAnalysis, error)
	GetSlowQueries() ([]*SlowQuery, error)
	OptimizeSchema() error
}

// QueryAnalysis contains detailed analysis of a query
type QueryAnalysis struct {
	EstimatedCost   float64       `json:"estimated_cost"`
	ExecutionPlan   string        `json:"execution_plan"`
	IndexUsage      []string      `json:"index_usage"`
	Recommendations []string      `json:"recommendations"`
	EstimatedTime   time.Duration `json:"estimated_time"`
}

// SlowQuery represents a slow query that needs optimization
type SlowQuery struct {
	Query       string        `json:"query"`
	Duration    time.Duration `json:"duration"`
	Frequency   int64         `json:"frequency"`
	LastSeen    time.Time     `json:"last_seen"`
	TablesScan  []string      `json:"tables_scan"`
	Suggestions []string      `json:"suggestions"`
}

// SQLiteOptimizer implements QueryOptimizer for SQLite
type SQLiteOptimizer struct {
	db            *sql.DB
	slowQueries   map[string]*SlowQuery
	queryPatterns map[string]*QueryPattern
}

// QueryPattern represents a common query pattern for optimization
type QueryPattern struct {
	Pattern      *regexp.Regexp
	Optimization func(string, []interface{}) (string, []interface{})
	Description  string
}

// NewSQLiteOptimizer creates a new SQLite query optimizer
func NewSQLiteOptimizer(db *sql.DB) *SQLiteOptimizer {
	optimizer := &SQLiteOptimizer{
		db:            db,
		slowQueries:   make(map[string]*SlowQuery),
		queryPatterns: make(map[string]*QueryPattern),
	}

	optimizer.initializePatterns()
	return optimizer
}

// initializePatterns sets up common query optimization patterns
func (o *SQLiteOptimizer) initializePatterns() {
	// SELECT * optimization
	o.queryPatterns["select_all"] = &QueryPattern{
		Pattern: regexp.MustCompile(`SELECT\s+\*\s+FROM\s+(\w+)`),
		Optimization: func(query string, params []interface{}) (string, []interface{}) {
			// This would be implemented to replace SELECT * with specific columns
			// For now, return as-is but add to recommendations
			return query, params
		},
		Description: "Replace SELECT * with specific columns for better performance",
	}

	// LIKE optimization
	o.queryPatterns["like_optimization"] = &QueryPattern{
		Pattern: regexp.MustCompile(`LIKE\s+'%([^%]+)%'`),
		Optimization: func(query string, params []interface{}) (string, []interface{}) {
			// Consider full-text search for better performance
			return query, params
		},
		Description: "Consider using FTS for complex LIKE queries",
	}

	// ORDER BY without LIMIT optimization
	o.queryPatterns["order_without_limit"] = &QueryPattern{
		Pattern: regexp.MustCompile(`ORDER\s+BY\s+.*(?!LIMIT)`),
		Optimization: func(query string, params []interface{}) (string, []interface{}) {
			// Suggest adding LIMIT for large datasets
			return query, params
		},
		Description: "Consider adding LIMIT clause when using ORDER BY",
	}
}

// OptimizeQuery optimizes a given query based on patterns and analysis
func (o *SQLiteOptimizer) OptimizeQuery(query string, params []interface{}) (string, []interface{}, error) {
	normalizedQuery := strings.ToUpper(strings.TrimSpace(query))

	// Apply optimization patterns
	optimizedQuery := query
	optimizedParams := params

	for _, pattern := range o.queryPatterns {
		if pattern.Pattern.MatchString(normalizedQuery) {
			optimizedQuery, optimizedParams = pattern.Optimization(optimizedQuery, optimizedParams)
		}
	}

	// Add query hints for SQLite
	if strings.HasPrefix(normalizedQuery, "SELECT") {
		// Add query planner hints if beneficial
		optimizedQuery = o.addQueryHints(optimizedQuery)
	}

	return optimizedQuery, optimizedParams, nil
}

// addQueryHints adds SQLite-specific query hints for optimization
func (o *SQLiteOptimizer) addQueryHints(query string) string {
	// Add appropriate hints based on query structure
	// For SQLite, this might include index hints or query structure modifications
	return query
}

// AddIndex creates an index on the specified table and column
func (o *SQLiteOptimizer) AddIndex(table, column string) error {
	indexName := fmt.Sprintf("idx_%s_%s", table, column)
	query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)", indexName, table, column)

	_, err := o.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create index %s: %w", indexName, err)
	}

	return nil
}

// AnalyzeQuery performs detailed analysis of a query
func (o *SQLiteOptimizer) AnalyzeQuery(query string) (*QueryAnalysis, error) {
	analysis := &QueryAnalysis{
		IndexUsage:      []string{},
		Recommendations: []string{},
	}

	// Get query plan from SQLite
	planQuery := fmt.Sprintf("EXPLAIN QUERY PLAN %s", query)
	rows, err := o.db.Query(planQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query: %w", err)
	}
	defer rows.Close()

	var planBuilder strings.Builder
	var cost float64 = 0

	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			continue
		}

		planBuilder.WriteString(fmt.Sprintf("%d|%d|%s\n", id, parent, detail))

		// Analyze plan details for cost estimation
		if strings.Contains(detail, "SCAN") {
			cost += 100 // Table scan is expensive
		} else if strings.Contains(detail, "SEARCH") {
			cost += 10 // Index search is cheaper
		}

		// Check for index usage
		if strings.Contains(detail, "USING INDEX") {
			indexName := o.extractIndexName(detail)
			if indexName != "" {
				analysis.IndexUsage = append(analysis.IndexUsage, indexName)
			}
		}
	}

	analysis.ExecutionPlan = planBuilder.String()
	analysis.EstimatedCost = cost
	analysis.EstimatedTime = time.Duration(cost * float64(time.Microsecond))

	// Generate recommendations based on analysis
	analysis.Recommendations = o.generateRecommendations(query, analysis.ExecutionPlan)

	return analysis, nil
}

// extractIndexName extracts index name from execution plan detail
func (o *SQLiteOptimizer) extractIndexName(detail string) string {
	// Extract index name from "USING INDEX index_name" pattern
	re := regexp.MustCompile(`USING INDEX (\w+)`)
	matches := re.FindStringSubmatch(detail)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// generateRecommendations generates optimization recommendations based on query analysis
func (o *SQLiteOptimizer) generateRecommendations(query, plan string) []string {
	recommendations := []string{}

	queryUpper := strings.ToUpper(query)

	// Check for table scans
	if strings.Contains(plan, "SCAN") {
		recommendations = append(recommendations, "Consider adding indexes for columns used in WHERE clauses")
	}

	// Check for SELECT *
	if strings.Contains(queryUpper, "SELECT *") {
		recommendations = append(recommendations, "Replace SELECT * with specific column names to reduce data transfer")
	}

	// Check for complex WHERE clauses
	if strings.Count(queryUpper, "WHERE") > 0 && strings.Count(queryUpper, "AND") > 2 {
		recommendations = append(recommendations, "Consider breaking complex WHERE clauses into simpler queries")
	}

	// Check for ORDER BY without LIMIT
	if strings.Contains(queryUpper, "ORDER BY") && !strings.Contains(queryUpper, "LIMIT") {
		recommendations = append(recommendations, "Consider adding LIMIT clause when using ORDER BY for better performance")
	}

	// Check for LIKE with leading wildcards
	if strings.Contains(queryUpper, "LIKE '%") {
		recommendations = append(recommendations, "Avoid LIKE patterns starting with % - consider full-text search instead")
	}

	// Apply pattern-based recommendations
	for _, pattern := range o.queryPatterns {
		if pattern.Pattern.MatchString(queryUpper) {
			recommendations = append(recommendations, pattern.Description)
		}
	}

	return recommendations
}

// GetSlowQueries returns a list of identified slow queries
func (o *SQLiteOptimizer) GetSlowQueries() ([]*SlowQuery, error) {
	queries := make([]*SlowQuery, 0, len(o.slowQueries))
	for _, query := range o.slowQueries {
		queries = append(queries, query)
	}

	return queries, nil
}

// RecordSlowQuery records a slow query for analysis
func (o *SQLiteOptimizer) RecordSlowQuery(query string, duration time.Duration) {
	key := strings.TrimSpace(query)

	if existing, exists := o.slowQueries[key]; exists {
		existing.Frequency++
		existing.LastSeen = time.Now()
		if duration > existing.Duration {
			existing.Duration = duration
		}
	} else {
		o.slowQueries[key] = &SlowQuery{
			Query:       query,
			Duration:    duration,
			Frequency:   1,
			LastSeen:    time.Now(),
			Suggestions: o.generateQuerySuggestions(query),
		}
	}
}

// generateQuerySuggestions generates specific suggestions for query optimization
func (o *SQLiteOptimizer) generateQuerySuggestions(query string) []string {
	suggestions := []string{}
	queryUpper := strings.ToUpper(query)

	// Table scan suggestions
	if strings.Contains(queryUpper, "SELECT") && strings.Contains(queryUpper, "WHERE") {
		suggestions = append(suggestions, "Add indexes on WHERE clause columns")
	}

	// Join optimization suggestions
	if strings.Contains(queryUpper, "JOIN") {
		suggestions = append(suggestions, "Ensure foreign key columns are indexed")
		suggestions = append(suggestions, "Consider the order of JOIN operations")
	}

	return suggestions
}

// OptimizeSchema performs overall schema optimization
func (o *SQLiteOptimizer) OptimizeSchema() error {
	// Run ANALYZE to update SQLite statistics
	if _, err := o.db.Exec("ANALYZE"); err != nil {
		return fmt.Errorf("failed to analyze database: %w", err)
	}

	// Run VACUUM to defragment the database
	if _, err := o.db.Exec("VACUUM"); err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}

	// Optimize common query patterns by suggesting indexes
	commonIndexes := []struct {
		table  string
		column string
	}{
		{"entities", "domain"},
		{"entities", "state"},
		{"device_states", "device_id"},
		{"device_states", "timestamp"},
		{"metrics", "metric_name"},
		{"metrics", "timestamp"},
		{"automation_executions", "automation_id"},
		{"automation_executions", "timestamp"},
	}

	for _, idx := range commonIndexes {
		if err := o.AddIndex(idx.table, idx.column); err != nil {
			// Log error but continue with other indexes
			continue
		}
	}

	return nil
}

// GetOptimizationReport generates a comprehensive optimization report
func (o *SQLiteOptimizer) GetOptimizationReport() (*OptimizationReport, error) {
	report := &OptimizationReport{
		GeneratedAt:       time.Now(),
		SlowQueries:       make([]*SlowQuery, 0),
		IndexSuggestions:  make([]IndexSuggestion, 0),
		SchemaSuggestions: make([]string, 0),
	}

	// Collect slow queries
	slowQueries, err := o.GetSlowQueries()
	if err != nil {
		return nil, err
	}
	report.SlowQueries = slowQueries

	// Generate index suggestions based on slow queries
	for _, sq := range slowQueries {
		suggestions := o.analyzeQueryForIndexes(sq.Query)
		report.IndexSuggestions = append(report.IndexSuggestions, suggestions...)
	}

	// Generate schema-level suggestions
	report.SchemaSuggestions = []string{
		"Run ANALYZE regularly to update query planner statistics",
		"Consider VACUUM for databases with frequent deletes/updates",
		"Use appropriate column types to minimize storage",
		"Consider partitioning for very large tables",
	}

	return report, nil
}

// OptimizationReport contains comprehensive optimization recommendations
type OptimizationReport struct {
	GeneratedAt       time.Time         `json:"generated_at"`
	SlowQueries       []*SlowQuery      `json:"slow_queries"`
	IndexSuggestions  []IndexSuggestion `json:"index_suggestions"`
	SchemaSuggestions []string          `json:"schema_suggestions"`
}

// IndexSuggestion represents a suggested index for performance improvement
type IndexSuggestion struct {
	Table    string   `json:"table"`
	Columns  []string `json:"columns"`
	Reason   string   `json:"reason"`
	Impact   string   `json:"impact"`
	Priority int      `json:"priority"` // 1-5, 5 being highest
}

// analyzeQueryForIndexes analyzes a query to suggest beneficial indexes
func (o *SQLiteOptimizer) analyzeQueryForIndexes(query string) []IndexSuggestion {
	suggestions := []IndexSuggestion{}

	// Simple pattern matching for common optimization opportunities
	// In a production system, this would use proper SQL parsing

	// WHERE clause analysis
	wherePattern := regexp.MustCompile(`WHERE\s+(\w+)\.?(\w+)\s*[=<>]`)
	matches := wherePattern.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			table := match[1]
			column := match[2]

			suggestions = append(suggestions, IndexSuggestion{
				Table:    table,
				Columns:  []string{column},
				Reason:   "Column used in WHERE clause",
				Impact:   "High - can eliminate table scans",
				Priority: 4,
			})
		}
	}

	// ORDER BY analysis
	orderPattern := regexp.MustCompile(`ORDER\s+BY\s+(\w+)\.?(\w+)`)
	orderMatches := orderPattern.FindAllStringSubmatch(query, -1)

	for _, match := range orderMatches {
		if len(match) >= 3 {
			table := match[1]
			column := match[2]

			suggestions = append(suggestions, IndexSuggestion{
				Table:    table,
				Columns:  []string{column},
				Reason:   "Column used in ORDER BY clause",
				Impact:   "Medium - can avoid sorting large result sets",
				Priority: 3,
			})
		}
	}

	return suggestions
}
