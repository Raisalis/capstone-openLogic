module backend

go 1.17

replace (
	datastore => ./datastore
	google-token-auth => ./google-token-auth
)

require (
	datastore v0.0.0-00010101000000-000000000000
	google-token-auth v0.0.0-00010101000000-000000000000
)

require github.com/mattn/go-sqlite3 v1.14.12 // indirect
