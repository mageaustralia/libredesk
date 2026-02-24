package dbutil

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// PaginationOptions represents the options for paginating a query.
type PaginationOptions struct {
	Page     int
	PageSize int
	OrderBy  string
	Order    string
}

// Order directions.
const (
	ASC  = "ASC"
	DESC = "DESC"
)

// Filter represents a filter to be applied to a query.
type Filter struct {
	Model    string `json:"model"`
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// AllowedFields is a map of model names to a list of allowed fields for that model.
type AllowedFields map[string][]string

// BuildPaginatedQuery builds a paginated query from the given base query, existing arguments, pagination options, filters JSON, and allowed fields.
func BuildPaginatedQuery(baseQuery string, existingArgs []any, opts PaginationOptions, filtersJSON string, allowedFields AllowedFields) (string, []any, error) {
	if opts.Page <= 0 {
		return "", nil, fmt.Errorf("invalid page number: %d", opts.Page)
	}
	if opts.PageSize <= 0 {
		return "", nil, fmt.Errorf("invalid page size: %d", opts.PageSize)
	}

	var filters []Filter
	if filtersJSON != "" {
		if err := json.Unmarshal([]byte(filtersJSON), &filters); err != nil {
			return "", nil, fmt.Errorf("invalid filters JSON: %w", err)
		}
	}

	whereClause, filterArgs, err := buildWhereClause(filters, existingArgs, allowedFields)
	if err != nil {
		return "", nil, err
	}

	query := baseQuery
	args := existingArgs

	if whereClause != "" {
		query += " AND " + whereClause
		args = append(args, filterArgs...)
	}

	if opts.OrderBy != "" {
		// Validate OrderBy.
		parts := strings.Split(opts.OrderBy, ".")
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("invalid OrderBy format: %s", opts.OrderBy)
		}
		model, field := parts[0], parts[1]

		modelFields, ok := allowedFields[model]
		if !ok || !slices.Contains(modelFields, field) {
			return "", nil, fmt.Errorf("invalid OrderBy field: %s", opts.OrderBy)
		}

		order := strings.ToUpper(opts.Order)
		if order != "" && order != ASC && order != DESC {
			return "", nil, fmt.Errorf("invalid order direction: %s", opts.Order)
		}
		query += fmt.Sprintf(" ORDER BY %s.%s %s NULLS LAST", model, field, order)
	}

	offset := (opts.Page - 1) * opts.PageSize
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, opts.PageSize, offset)

	return query, args, nil
}

// buildWhereClause builds a WHERE clause from the given filters and returns the WHERE clause and the arguments to be passed to the query.
func buildWhereClause(filters []Filter, existingArgs []interface{}, allowedFields AllowedFields) (string, []interface{}, error) {
	conditions := []string{}
	args := []interface{}{}
	paramCount := len(existingArgs) + 1

	for _, f := range filters {
		modelFields, ok := allowedFields[f.Model]
		if !ok {
			return "", nil, fmt.Errorf("invalid model: %s", f.Model)
		}
		if !slices.Contains(modelFields, f.Field) {
			return "", nil, fmt.Errorf("invalid field: %s for model: %s", f.Field, f.Model)
		}

		field := fmt.Sprintf("%s.%s", f.Model, f.Field)

		switch f.Operator {
		case "equals":
			conditions = append(conditions, field+fmt.Sprintf(" = $%d", paramCount))
			args = append(args, f.Value)
			paramCount++
		case "not equals":
			conditions = append(conditions, field+fmt.Sprintf(" != $%d", paramCount))
			args = append(args, f.Value)
			paramCount++
		case "set":
			conditions = append(conditions, field+" IS NOT NULL")
		case "not set":
			conditions = append(conditions, field+" IS NULL")
		case "in":
			var raw []json.RawMessage
			if err := json.Unmarshal([]byte(f.Value), &raw); err != nil {
				return "", nil, fmt.Errorf("invalid array format for 'in' operator: %v", err)
			}
			arr := make([]string, len(raw))
			for i, r := range raw {
				// Strip quotes from strings, keep numbers as-is
				s := strings.Trim(string(r), "\"")
				arr[i] = s
			}
			if len(arr) == 0 {
				continue
			}
			placeholders := make([]string, len(arr))
			for i, v := range arr {
				placeholders[i] = fmt.Sprintf("$%d", paramCount)
				args = append(args, v)
				paramCount++
			}
			conditions = append(conditions, field+" IN ("+strings.Join(placeholders, ",")+")")
		case "not_in":
			var raw []json.RawMessage
			if err := json.Unmarshal([]byte(f.Value), &raw); err != nil {
				return "", nil, fmt.Errorf("invalid array format for 'not_in' operator: %v", err)
			}
			arr := make([]string, len(raw))
			for i, r := range raw {
				// Strip quotes from strings, keep numbers as-is
				s := strings.Trim(string(r), "\"")
				arr[i] = s
			}
			if len(arr) == 0 {
				continue
			}
			placeholders := make([]string, len(arr))
			for i, v := range arr {
				placeholders[i] = fmt.Sprintf("$%d", paramCount)
				args = append(args, v)
				paramCount++
			}
			conditions = append(conditions, field+" NOT IN ("+strings.Join(placeholders, ",")+")")
		case "in_or_null":
			var raw []json.RawMessage
			if err := json.Unmarshal([]byte(f.Value), &raw); err != nil {
				return "", nil, fmt.Errorf("invalid array format for 'in_or_null' operator: %v", err)
			}
			arr := make([]string, len(raw))
			for i, r := range raw {
				// Strip quotes from strings, keep numbers as-is
				s := strings.Trim(string(r), "\"")
				arr[i] = s
			}
			if len(arr) == 0 {
				// No specific values, just match NULL
				conditions = append(conditions, field+" IS NULL")
			} else {
				placeholders := make([]string, len(arr))
				for i, v := range arr {
					placeholders[i] = fmt.Sprintf("$%d", paramCount)
					args = append(args, v)
					paramCount++
				}
				conditions = append(conditions, "("+field+" IN ("+strings.Join(placeholders, ",")+") OR "+field+" IS NULL)")
			}
		case "relative_date":
			now := time.Now()
			var start, end time.Time
			switch f.Value {
			case "today":
				start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				end = start.Add(24 * time.Hour)
			case "yesterday":
				end = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				start = end.Add(-24 * time.Hour)
			case "last_7_days":
				end = now
				start = now.AddDate(0, 0, -7)
			case "last_30_days":
				end = now
				start = now.AddDate(0, 0, -30)
			case "this_month":
				start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
				end = start.AddDate(0, 1, 0)
			default:
				return "", nil, fmt.Errorf("unknown relative_date preset: %s", f.Value)
			}
			conditions = append(conditions, fmt.Sprintf("%s >= $%d AND %s < $%d", field, paramCount, field, paramCount+1))
			args = append(args, start, end)
			paramCount += 2
		case "between":
			values := strings.Split(f.Value, ",")
			if len(values) != 2 {
				return "", nil, fmt.Errorf("between requires 2 values")
			}
			conditions = append(conditions, fmt.Sprintf("%s BETWEEN $%d AND $%d", field, paramCount, paramCount+1))
			args = append(args, strings.TrimSpace(values[0]), strings.TrimSpace(values[1]))
			paramCount += 2
		case "ilike":
			conditions = append(conditions, field+fmt.Sprintf(" ILIKE $%d", paramCount))
			args = append(args, "%"+f.Value+"%")
			paramCount++
		default:
			return "", nil, fmt.Errorf("invalid operator: %s", f.Operator)
		}
	}

	if len(conditions) == 0 {
		return "", nil, nil
	}

	return strings.Join(conditions, " AND "), args, nil
}
