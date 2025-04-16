package internal

// QueryType defines the possible queries for the state machine.
type QueryType uint8

const (
	QueryTGet       QueryType = iota // Retrieve an entry by key.
	QueryTHas                        // Check if a key was ever inserted.
	QueryTGetDBInfo                  // Retrieve metadata about the database underlying the machine.
)

func (q QueryType) String() string {
	switch q {
	case QueryTGet:
		return "Get"
	case QueryTHas:
		return "Has"
	case QueryTGetDBInfo:
		return "GetDBInfo"
	default:
		return "Unknown"
	}
}

// Query defines the structure for lookup requests (read-only) sent via SyncRead or ReadStale
type Query struct {
	Type QueryType // The type of Query to perform.
	Key  string    // The key for the Query (emtpy for some queries).
}

// QueryResult is the result of a QueryTGet operation.
// All other query results are primitive types or predefined structs (bool, db.DatabaseInfo).
type QueryResult struct {
	Ok    bool
	Value []byte
}
