package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"

	tokenauth "./google-token-auth"
	_ "github.com/mattn/go-sqlite3"
)

var (
	// Set allowed domains here.
	authorized_domains = []string{
		"csumb.edu",
	}

	// Client-side client ID from your Google Developer Console
	// Same as in the front-end index.php
	authorized_client_ids = []string{
		"171509471210-8d883n4nfjebkqvkp29p50ijqmt6c5nd.apps.googleusercontent.com",
	}

	admin_users = map[string]bool{
		"gbruns@csumb.edu":   true,
		"cohunter@csumb.edu": true,
	}

	database_uri = "../db.sqlite3"
)

type Proof struct {
	Id_token       string   // Google Auth token
	Id             string   // mongo ID
	EntryType      string   // ?
	ProofName      string   // ?
	ProofType      string   // ?
	Premise        []string // ?
	Logic          []string // ?
	Rules          []string // ?
	ProofCompleted string   // ?
	Conclusion     string   // ?
	RepoProblem    string   // ?
}

type ProofSQLite struct {
	Id             int
	EntryType      string
	UserSubmitted  string
	ProofName      string
	ProofType      string
	Premise        []string
	Logic          []string
	Rules          []string
	ProofCompleted string
	TimeSubmitted  string
	Conclusion     string
	RepoProblem    string
}

func saveProof(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" || req.Body == nil {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	var submittedProof Proof

	if err := json.NewDecoder(req.Body).Decode(&submittedProof); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 400)
		return
	}

	log.Printf("%+v", submittedProof)

	if len(submittedProof.ProofName) == 0 {
		http.Error(w, "Proof name is empty", 400)
		return
	}

	tok, valid := tokenauth.Verify(submittedProof.Id_token)
	log.Printf("%+v, %+v", valid, tok)

	if !valid {
		http.Error(w, "Invalid token", 400)
	}

	db, err := sql.Open("sqlite3", database_uri)
	if err != nil {
		http.Error(w, "Database open error", 500)
		log.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database transaction begin error", 500)
		log.Fatal(err)
	}
	stmt, err := tx.Prepare(`INSERT INTO proofs (entryType,
												userSubmitted,
												proofName,
												proofType,
												Premise,
												Logic,
												Rules,
												proofCompleted,
												timeSubmitted,
												Conclusion,
												repoProblem)
							 VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), ?, ?)
							 ON CONFLICT (userSubmitted, proofName) DO UPDATE SET
							 	entryType = ?,
							 	proofType = ?,
							 	Premise = ?,
							 	Logic = ?,
							 	Rules = ?,
							 	proofCompleted = ?,
							 	timeSubmitted = datetime('now'),
							 	Conclusion = ?,
							 	repoProblem = ?`)
	defer stmt.Close()
	if err != nil {
		http.Error(w, "Transaction prepare error", 500)
		return
	}

	PremiseJSON, err := json.Marshal(submittedProof.Premise)
	if err != nil {
		http.Error(w, "Premise marshal error", 500)
		return
	}
	LogicJSON, err := json.Marshal(submittedProof.Logic)
	if err != nil {
		http.Error(w, "Logic marshal error", 500)
		return
	}
	RulesJSON, err := json.Marshal(submittedProof.Rules)
	if err != nil {
		http.Error(w, "Rules marshal error", 500)
		return
	}
	_, err = stmt.Exec(submittedProof.EntryType, tok.Email, submittedProof.ProofName, submittedProof.ProofType,
		PremiseJSON, LogicJSON, RulesJSON, submittedProof.ProofCompleted, submittedProof.Conclusion, submittedProof.RepoProblem,
		submittedProof.EntryType, submittedProof.ProofType, PremiseJSON, LogicJSON, RulesJSON, submittedProof.ProofCompleted,
		submittedProof.Conclusion, submittedProof.RepoProblem)
	if err != nil {
		http.Error(w, "Statement exec error", 500)
		log.Fatal(err)
		return
	}
	tx.Commit()

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"success": "true"}`)
}

func getProofs(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" || req.Body == nil {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	// Accepted JSON fields must be defined here
	type getProofRequest struct {
		Token     string `json:"id_token"`
		Selection string `json:"selection"`
	}

	var requestData getProofRequest

	decoder := json.NewDecoder(req.Body)

	// Will cause error if fields not defined in getProofRequest are sent
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, "Unable to decode request body.", 400)
		return
	}

	log.Printf("%+v", requestData)

	db, err := sql.Open("sqlite3", database_uri)
	if err != nil {
		http.Error(w, "Database open error", 500)
		log.Fatal(err)
	}
	defer db.Close()

	if len(requestData.Token) == 0 {
		http.Error(w, "Token required", 400)
		return
	}

	if len(requestData.Selection) == 0 {
		http.Error(w, "Selection required", 400)
		return
	}

	tok, valid := tokenauth.Verify(requestData.Token)
	if !valid {
		http.Error(w, "Token not valid", 400)
		return
	}

	user := tok.Email
	log.Printf("USER: %q", user)

	var stmt *sql.Stmt
	var rows *sql.Rows

	switch requestData.Selection {
	case "user":
		log.Println("user selection")
		stmt, err = db.Prepare("SELECT id, entryType, userSubmitted, proofName, proofType, Premise, Logic, Rules, proofCompleted, timeSubmitted, Conclusion, repoProblem FROM proofs WHERE userSubmitted = ? AND proofCompleted = 'false' AND proofName != 'n/a'")
		if err != nil {
			http.Error(w, "Statment prepare error", 500)
			log.Fatal(err)
			return
		}
		defer stmt.Close()
		rows, err = stmt.Query(user)

	case "repo":
		log.Println("repo selection")
		stmt, err = db.Prepare("SELECT id, entryType, userSubmitted, proofName, proofType, Premise, Logic, Rules, proofCompleted, timeSubmitted, Conclusion, repoProblem FROM proofs WHERE userSubmitted = ? AND proofName LIKE 'Repository%'")
		if err != nil {
			http.Error(w, "Statement prepare error", 500)
			log.Fatal(err)
			return
		}
		defer stmt.Close()
		rows, err = stmt.Query("gbruns@csumb.edu")

	case "completedrepo":
		log.Println("completedrepo selection")
		stmt, err = db.Prepare("SELECT id, entryType, userSubmitted, proofName, proofType, Premise, Logic, Rules, proofCompleted, timeSubmitted, Conclusion, repoProblem FROM proofs WHERE userSubmitted = ? AND proofCompleted = 'true'")
		if err != nil {
			http.Error(w, "Statement prepare error", 500)
			log.Fatal(err)
			return
		}
		defer stmt.Close()
		rows, err = stmt.Query(user)

	case "downloadrepo":
		log.Println("downloadrepo selection")
		//'id,entryType,userSubmitted, proofName, proofType, Premise, Logic, Rules, proofCompleted, timeSubmitted, Conclusion\n';

		stmt, err = db.Prepare("SELECT id, entryType, userSubmitted, proofName, proofType, Premise, Logic, Rules, proofCompleted, timeSubmitted, Conclusion, repoProblem FROM proofs")
		if err != nil {
			http.Error(w, "Statement prepare error", 500)
			log.Fatal(err)
			return
		}
		defer stmt.Close()
		rows, err = stmt.Query()

	default:
		http.Error(w, "invalid selection", 400)
		return
	}

	if err != nil {
		http.Error(w, "Query error", 500)
		return
	}
	defer rows.Close()

	var userProofs []ProofSQLite
	for rows.Next() {
		var userProof ProofSQLite
		var PremiseJSON string
		var LogicJSON string
		var RulesJSON string

		err = rows.Scan(&userProof.Id, &userProof.EntryType, &userProof.UserSubmitted, &userProof.ProofName, &userProof.ProofType, &PremiseJSON, &LogicJSON, &RulesJSON, &userProof.ProofCompleted, &userProof.TimeSubmitted, &userProof.Conclusion, &userProof.RepoProblem)
		if err != nil {
			http.Error(w, "Query read error", 500)
			log.Print(err)
			return
		}

		if err = json.Unmarshal([]byte(PremiseJSON), &userProof.Premise); err != nil {
			http.Error(w, "premise decode error", 500)
			return
		}
		if err = json.Unmarshal([]byte(LogicJSON), &userProof.Logic); err != nil {
			http.Error(w, "logic decode error", 500)
			return
		}
		if err = json.Unmarshal([]byte(RulesJSON), &userProof.Rules); err != nil {
			http.Error(w, "rules decode error", 500)
			return
		}

		log.Printf("%+v", userProof)
		userProofs = append(userProofs, userProof)
	}

	log.Printf("%+v", userProofs)
	userProofsJSON, err := json.Marshal(userProofs)
	if err != nil {
		http.Error(w, "json marshal error", 500)
		log.Print(err)
		return
	}

	io.WriteString(w, string(userProofsJSON))

	log.Printf("%q %q", user, tok)

	log.Printf("%+v", req.URL.Query())

}

func testFunc(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "Test Func")
}
func main() {
	log.Println("Server initializing")
	db, err := sql.Open("sqlite3", database_uri)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize database tables
	// proofs : [Premise, Logic, Rules] are JSON fields
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS proofs (
						id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
						entryType TEXT,
						userSubmitted TEXT,
						proofName TEXT,
						proofType TEXT,
						Premise TEXT,
						Logic TEXT,
						Rules TEXT,
						proofCompleted TEXT,
						timeSubmitted DATETIME,
						Conclusion TEXT,
						repoProblem TEXT
					)`)
	if err != nil {
		log.Fatal(err)
	}
	// proofs : Unique index on (userSubmitted, proofName)
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_user_proof
						ON proofs (userSubmitted, proofName)`)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize token auth/cache
	tokenauth.SetAuthorizedDomains(authorized_domains)
	tokenauth.SetAuthorizedClientIds(authorized_client_ids)

	// method saveproof : POST : JSON <- id_token, proof
	http.HandleFunc("/saveproof", saveProof)

	// method user : GET : JSON -> [proof, proof]
	http.HandleFunc("/proofs", getProofs)

	log.Println("Server started")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
