package database

// SQLStore is a lightweight base for stores that operate on a database
// connection or transaction. It is intended to be embedded in concrete store
// types so they share a common DB accessor without duplicating the field.
type SQLStore struct {
	DB DBTX
}

// NewSQLStore creates a store base backed by db.
func NewSQLStore(db DBTX) SQLStore {
	return SQLStore{DB: db}
}
