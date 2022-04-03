package datastore

import (
	"database/sql"
	"log"
	"os"
	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dataSourceName string) (*ProofStore, error) {
	log.Println("Initializing db.sqlite3...")
	file, err := os.Create("db.sqlite3") // Create the SQLite file
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
	log.Println("db.sqlite3 opened")

	// if all went well, return the db and nil error
	return &ProofStore{db: sqliteDatabase}, nil
}

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
	} else {
		statement.Exec() // execute the prepared SQL statement
		log.Println("user table created")
	}
	defer statement.Close()

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
	} else {
		statement.Exec()
		log.Println("section table created")
	}
	
	createRosterRelationSQL := `CREATE TABLE IF NOT EXISTS roster(
		"sectionName" TEXT NOT NULL,
		"userEmail" TEXT NOT NULL,
		"role" TEXT NOT NULL
			CHECK (role in ('instructor', 'ta', 'student')),
		PRIMARY KEY (sectionName, userEmail),
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
	} else {
		statement.Exec()
		log.Println("roster relation created")
	}
	

	log.Println("Creating proof table. . .")
	// proofs : [Premise, Logic, Rules] are JSON fields
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS proof (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		entryType TEXT,
		userSubmitted TEXT,
		proofName TEXT,
		proofType TEXT,
		Premise TEXT,
		Logic TEXT,
		Rules TEXT,
		everCompleted TEXT DEFAULT 'false',
		proofCompleted TEXT DEFAULT 'false',
		timeSubmitted DATETIME,
		Conclusion TEXT,
		repoProblem TEXT
	)`)
	if err != nil {
		return err
	} else {
		log.Println("proof table created")
	}

	createAssignmentTableSQL := `CREATE TABLE IF NOT EXISTS assignment (
		instructorEmail TEXT,
		sectionName TEXT,
		name TEXT,
		proofIds TEXT,
		PRIMARY KEY (instructorEmail, sectionName, name),
		FOREIGN KEY (instructorEmail) REFERENCES user (email),
		FOREIGN KEY (sectionName) REFERENCES section (name)
	);`

	log.Println("Creating assignment table. . .")
	statement, err = db.Prepare(createAssignmentTableSQL)
	if err != nil {
		return err
	} else {
		statement.Exec()
		log.Println("assignment table created")
	}
	
	// proofs : Unique index on (userSubmitted, proofName, proofCompleted)
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_user_proof
			ON proof (userSubmitted, proofName, proofCompleted)`)
	if err != nil {
		return err
	} else {
		log.Println("unique index for (userSubmitted, proofName, proofCompleted) created")
	}

	return nil
}

// close the database, prevent new queries from running
func (p *ProofStore) Close() error {
	return p.db.Close()
}