package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/utils"
)

func Error[T any](err error, model T) error {
	if err == nil {
		return nil
	}

	// Get model name for contextual error messages
	modelName := utils.GetTemplateName[T]()

	// Handle standard SQL errors
	if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "no rows") {
		return inerr.NewErrNotFound(modelName)
	}

	// Handle connection timeouts and context cancellation
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("database operation timed out while processing %s", modelName)
	}

	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("database operation was cancelled for %s", modelName)
	}

	// Handle PostgreSQL specific errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return mapPostgreSQLError(pgErr, modelName)
	}

	// Handle connection errors
	if isConnectionError(err) {
		return fmt.Errorf("database connection failed")
	}

	// Fallback for unknown errors
	return fmt.Errorf("unexpected database error occurred: %w", err)
}

func mapPostgreSQLError(pgErr *pgconn.PgError, modelName string) error {
	switch pgErr.Code {
	// Class 08 - Connection Exception
	case "08000": // connection_exception
		return fmt.Errorf("database connection error")

	case "08003": // connection_does_not_exist
		return fmt.Errorf("database connection lost")

	case "08006": // connection_failure
		return fmt.Errorf("database connection failed")

	// Class 23 - Integrity Constraint Violation
	case "23000": // integrity_constraint_violation
		return inerr.NewErrConflict(modelName)

	case "23001": // restrict_violation
		return inerr.NewErrConflict(modelName)

	case "23502": // not_null_violation
		columnName := pgErr.ColumnName
		if columnName == "" {
			columnName = "required field"
		}
		return fmt.Errorf("%s is missing required field: %s", modelName, columnName)

	case "23503": // foreign_key_violation
		constraintName := pgErr.ConstraintName
		if constraintName == "" {
			constraintName = "related record"
		}
		return fmt.Errorf("%s references non-existent %s", modelName, constraintName)

	case "23505": // unique_violation
		constraintName := pgErr.ConstraintName
		if constraintName == "" {
			constraintName = "field"
		}
		return inerr.NewErrConflict(modelName)

	case "23514": // check_constraint_violation
		constraintName := pgErr.ConstraintName
		if constraintName == "" {
			constraintName = "constraint"
		}
		return fmt.Errorf("%s violates %s constraint", modelName, constraintName)

	// Class 42 - Syntax Error or Access Rule Violation
	case "42601": // syntax_error
		return fmt.Errorf("database query syntax error")

	case "42703": // undefined_column
		columnName := pgErr.ColumnName
		if columnName == "" {
			columnName = "unknown column"
		}
		return fmt.Errorf("database schema error: column '%s' does not exist", columnName)

	case "42P01": // undefined_table
		tableName := pgErr.TableName
		if tableName == "" {
			tableName = "unknown table"
		}
		return fmt.Errorf("database schema error: table '%s' does not exist", tableName)

	case "42501": // insufficient_privilege
		return fmt.Errorf("insufficient database privileges to perform %s operation", modelName)

	// Class 53 - Insufficient Resources
	case "53100":
		return fmt.Errorf("database storage is full")

	case "53200": // out_of_memory
		return fmt.Errorf("database out of memory")

	case "53300": // too_many_connections
		return fmt.Errorf("too many database connections")

	// Class 57 - Operator Intervention
	case "57000": // operator_intervention
		return fmt.Errorf("database operation was cancelled by administrator")

	case "57014": // query_canceled
		return fmt.Errorf("%s operation was cancelled", modelName)

	case "57P01": // admin_shutdown
		return fmt.Errorf("database is shutting down")

	// Class 58 - System Error
	case "58000": // system_error
		return fmt.Errorf("database system error")

	case "58030": // io_error
		return fmt.Errorf("database I/O error")
	default:
		return fmt.Errorf("database error occurred while processing %s", modelName)
	}
}

func isConnectionError(err error) bool {
	errStr := strings.ToLower(err.Error())
	connectionKeywords := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"no route to host",
		"network unreachable",
		"connection lost",
		"broken pipe",
		"connection closed",
		"dial tcp",
		"i/o timeout",
	}

	for _, keyword := range connectionKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}
	return false
}
