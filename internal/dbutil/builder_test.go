package dbutil

import (
	"strings"
	"testing"
)

func TestUserSearchField(t *testing.T) {
	allowed := AllowedFields{"users": {"email", UserSearchField}}

	filters := []Filter{{Model: "users", Field: UserSearchField, Operator: "ilike", Value: "  John   Smith "}}
	where, args, err := buildWhereClause(filters, []interface{}{"existing"}, allowed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// One shared parameter, numbered after the existing arg, covering all four columns.
	if strings.Count(where, "$2") != 4 {
		t.Errorf("expected $2 four times, got: %s", where)
	}
	for _, col := range []string{"users.email", "users.first_name", "users.last_name", "users.first_name || ' ' || users.last_name"} {
		if !strings.Contains(where, col) {
			t.Errorf("where clause missing %q: %s", col, where)
		}
	}
	if len(args) != 1 || args[0] != "%John Smith%" {
		t.Errorf("unexpected args: %v", args)
	}

	// Only ilike makes sense for the virtual field.
	bad := []Filter{{Model: "users", Field: UserSearchField, Operator: "equals", Value: "x"}}
	if _, _, err := buildWhereClause(bad, nil, allowed); err == nil {
		t.Error("expected error for non-ilike operator on search field")
	}

	// Not allowed unless the caller opts in.
	if _, _, err := buildWhereClause(filters, nil, AllowedFields{"users": {"email"}}); err == nil {
		t.Error("expected error when search field is not in allowed fields")
	}
}
