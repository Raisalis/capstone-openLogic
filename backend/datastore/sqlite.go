package datastore

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3" // import go-sqlite3 library
)

func InitDB(dataSourceName string) (*ProofStore, error) {
	log.Println("Initializing " + dataSourceName)
	// declare variables and assign values simultaneously using :=
	file, err := os.Create(dataSourceName) // Create the SQLite file
	if err != nil {                            // if an error occurred during database creation, log an error
		return nil, err
	}
	defer file.Close()
	
	sqliteDatabase, err := sql.Open("sqlite3", dataSourceName) // Open the created SQLite File
	if err != nil {
		return nil, err
	}
	// defer sqliteDatabase.Close()                       // Defer Closing the database
	err = createTables(sqliteDatabase)                // Create Database Tables using function defined below
	// check for errors when creating db
	if err != nil {
		return nil, err
	}
	log.Println(dataSourceName + " opened")

	// if all went well, return the db and nil error
	return &ProofStore{db: sqliteDatabase}, nil
}

// pass a db reference connection from main to method
// create tables for user, section, and roster within the referenced db
func createTables(db *sql.DB) (error) {
	createUserTableSQL := `CREATE TABLE IF NOT EXISTS user(
		"email" TEXT PRIMARY KEY,
		"firstName" TEXT,
		"lastName" TEXT,
		"admin" INTEGER DEFAULT 0
			CHECK (admin in (0, 1))
	);` // SQL statement for Create Table

	log.Println("Creating user table. . .")
	statement, err := db.Prepare(createUserTableSQL) // Prepare SQL statement
													 // Avoid SQL injection through prepared statements!
	if err != nil { // if an error occurred during statement preparation, return
		return err
	}
	defer statement.Close()

	statement.Exec() // execute the prepared SQL statement
	log.Println("user table created")

	createSectionTableSQL := `CREATE TABLE IF NOT EXISTS section(
		"instructorEmail" TEXT NOT NULL,
		"name" TEXT NOT NULL PRIMARY KEY,
		FOREIGN KEY (instructorEmail) REFERENCES user (email)
			ON UPDATE CASCADE
			ON DELETE CASCADE
	);`

	log.Println("Creating section table. . .")
	statement, err = db.Prepare(createSectionTableSQL)
	if err != nil {
		return err
	}
	statement.Exec()
	log.Println("section table created")

	createRosterRelationSQL := `CREATE TABLE IF NOT EXISTS roster(
		"sectionName" TEXT NOT NULL,
		"userEmail" TEXT NOT NULL,
		"role" TEXT NOT NULL
			CHECK (role in ('instructor', 'ta', 'student')),
		FOREIGN KEY (sectionName) REFERENCES section (name)
			ON UPDATE CASCADE
			ON DELETE CASCADE,
		FOREIGN KEY (userEmail) REFERENCES user (email)
			ON UPDATE CASCADE
			ON DELETE CASCADE
	);`

	log.Println("Creating roster relation. . .")
	statement, err = db.Prepare(createRosterRelationSQL)
	if err != nil {
		return err
	}
	statement.Exec()
	log.Println("roster relation created")

	// proof table from current proof-checker
	// foreign key and checks added, leveraging entryType for now
	createProofTableSQL := `CREATE TABLE IF NOT EXISTS proof (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		entryType TEXT DEFAULT 'proof'
			CHECK (entryType in ('proof', 'argument')),
		userSubmitted TEXT,
		proofName TEXT,
		proofType TEXT
			CHECK (proofType in ('prop', 'fol')),
		Premise TEXT,
		Logic TEXT,
		Rules TEXT,
		proofCompleted TEXT
			CHECK (proofCompleted in ('true', 'false', 'error')),
		timeSubmitted DATETIME,
		Conclusion TEXT,
		repoProblem TEXT
			CHECK (repoProblem in ('true', 'false')),
		FOREIGN KEY (userSubmitted) REFERENCES user (email)
	);`

	log.Println("Creating proof table. . .")
	statement, err = db.Prepare(createProofTableSQL)
	if err != nil {
		return err
	}
	statement.Exec()
	log.Println("proof table created")

	// createArgumentTableSQL := `CREATE TABLE IF NOT EXISTS argument (
	// 	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	// 	userSubmitted TEXT,
	// 	proofName TEXT,
	// 	proofType TEXT
	// 		CHECK (proofType in ('prop', 'fol')),
	// 	Premise TEXT,
	// 	timeSubmitted DATETIME,
	// 	Conclusion TEXT,
	// 	repoProblem TEXT
	// 		CHECK (repoProblem in ('true', 'false')),
	// 	FOREIGN KEY (userSubmitted) REFERENCES user (email)
	// );`

	// log.Println("Creating argument table. . .")
	// statement, err = db.Prepare(createArgumentTableSQL)
	// if err != nil {
	// 	return err
	// }
	// statement.Exec()
	// log.Println("argument table created")

	return nil
}

// close the database, prevent new queries from running
func (p *ProofStore) Close() error {
	return p.db.Close()
}