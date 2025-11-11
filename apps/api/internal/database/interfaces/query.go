// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package interfaces

// Field represents a single field in a query, specifying whether it's an
// indexed column or a JSONB path.
type Field struct {
	Name       string      // The snake_case column name OR the JSONB path (e.g., "data->>'body'").
	Value      interface{} // The value to query against.
	Operator   string      // The SQL operator (e.g., "=", ">", "<>", "ANY", "~*").
	IsJSONB    bool        // True if the Name is a JSONB path.
	JSONBCast  string      // Optional cast for JSONB values (e.g., "::boolean", "::bigint").
}

// Query defines a structured, database-agnostic query.
// This pattern makes the service layer's intent explicit, decouples the repository
// from any field-name knowledge, and dramatically simplifies the query builder.
type Query struct {
	Conditions []Field    // A list of AND conditions.
	OrGroups   [][]Field  // A list of OR groups, e.g., [[field1, field2], [field3]].
}

