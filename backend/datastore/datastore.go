package datastore

import (
   "database/sql"
   "encoding/json"
   "errors"
   "fmt"
   "log"
)

var (
   ErrDuplicate    = errors.New("record already exists")
   ErrNotExists    = errors.New("row does not exist")
   ErrUpdateFailed = errors.New("update failed")
   ErrDeleteFailed = errors.New("delete failed")
)

type Proof struct {
	Id             string   // SQL ID
	EntryType      string   // 'proof'
	UserSubmitted  string	// Used for results, ignored on user input
	ProofName      string   // user-chosen name (repo problems start with 'Repository - ')
	ProofType      string   // 'prop' (propositional/tfl) or 'fol' (first order logic)
	Premise        []string // premises of the proof; an array of WFFs
	Logic          []string // body of the proof; a JSON-encoded string
	Rules          []string // deprecated; now always an empty string
	ProofCompleted string   // 'true', 'false', or 'error'
	Conclusion     string   // conclusion of the proof
	RepoProblem    string   // 'true' if problem started from a repo problem, else 'false'
	TimeSubmitted  string
}

// Spring2022 - proposed table: argument

//type ProofStore interface {
//	GetByUser(string) Proof
//}

type UserWithEmail interface {
   GetEmail() string
}

type IProofStore interface {
	Close() error
	// Empty() error
	EmptyProofTable() error
	EmptyUserTable() error
	// EmptySectionTable() error
	// EmptyAssignmentTable() error
	InsertUser(user User) error
	InsertSection(section Section) error
	InsertRoster(rosterRow Roster) error
	GetAllAttemptedRepoProofs() (error, []Proof)
	GetRepoProofs() (error, []Proof)
	GetUserProofs(user UserWithEmail) (error, []Proof)
	GetUserCompletedProofs(user UserWithEmail) (error, []Proof)
	PopuplateTestUsersSectionsRosters()
	RemoveFromRoster(sectionName string, userEmail string) error
	RemoveSection(sectionName string) error
	Store(Proof) error
	MaintainAdmins(admin_users map[string]bool)
}

type ProofStore struct {
   db *sql.DB
}

// deprecated, see EmptyProofTable()
// func (p *ProofStore) Empty() error {
// 	_, err := p.db.Exec(`DELETE FROM proof`)
// 	return err
// }

func getProofsFromRows(rows *sql.Rows) (error, []Proof) {
	var userProofs []Proof
	for rows.Next() {
		var userProof Proof
		var PremiseJSON string
		var LogicJSON string
		var RulesJSON string

		err := rows.Scan(&userProof.Id, &userProof.EntryType, &userProof.UserSubmitted, &userProof.ProofName, &userProof.ProofType, &PremiseJSON, &LogicJSON, &RulesJSON, &userProof.ProofCompleted, &userProof.TimeSubmitted, &userProof.Conclusion, &userProof.RepoProblem)
		if err != nil {
			return err, nil
		}

		if err = json.Unmarshal([]byte(PremiseJSON), &userProof.Premise); err != nil {
			return err, nil
		}
		if err = json.Unmarshal([]byte(LogicJSON), &userProof.Logic); err != nil {
			return err, nil
		}
		if err = json.Unmarshal([]byte(RulesJSON), &userProof.Rules); err != nil {
			return err, nil
		}

		userProofs = append(userProofs, userProof)
	}

	return nil, userProofs
}

func (p *ProofStore) GetAllAttemptedRepoProofs() (error, []Proof) {
   // Create 'admin_repoproblems' view
   _, err := p.db.Exec(`DROP VIEW IF EXISTS admin_repoproblems`)
   if err != nil {
      return err, nil
   }

   _, err = p.db.Exec(`CREATE VIEW admin_repoproblems (userSubmitted, Premise, Conclusion) AS SELECT userSubmitted, Premise, Conclusion FROM proof WHERE userSubmitted IN (SELECT email FROM user WHERE admin = 1)`)
   if err != nil {
      return err, nil
   }

	stmt, err := p.db.Prepare(`SELECT id, entryType, userSubmitted, proofName, proofType, Premise, 
                              Logic, Rules, proofCompleted, timeSubmitted, Conclusion, repoProblem
                              FROM proof
                              INNER JOIN admin_repoproblems ON
                                 proofs.Premise = admin_repoproblems.Premise AND
                                 proofs.Conclusion = admin_repoproblems.Conclusion`)
   if err != nil {
      return err, nil
   }
   defer stmt.Close()
   
   rows, err := stmt.Query()
   if err != nil {
      return err, nil
   }
   defer rows.Close()

   return getProofsFromRows(rows)
}

func (p *ProofStore) GetRepoProofs() (error, []Proof) {
   stmt, err := p.db.Prepare(`SELECT id, entryType, userSubmitted, proofName, proofType, Premise,  
                              Logic, Rules, proofCompleted, timeSubmitted, Conclusion, repoProblem 
                              FROM proof WHERE repoProblem = 'true' AND userSubmitted 
                              IN (SELECT email FROM user WHERE admin = 1) ORDER BY userSubmitted`)
   if err != nil {
      return err, nil
   }
   defer stmt.Close()

   rows, err := stmt.Query()
   if err != nil {
      return err, nil
   }
   defer rows.Close()

   return getProofsFromRows(rows)
}

// retrieve unfinished proofs by user
func (p *ProofStore) GetUserProofs(user UserWithEmail) (error, []Proof) {
   stmt, err := p.db.Prepare(`SELECT id, entryType, userSubmitted, proofName, proofType, Premise, 
                              Logic, Rules, proofCompleted, timeSubmitted, Conclusion, repoProblem 
                              FROM proof WHERE userSubmitted = ? AND proofCompleted != 'true' AND proofName != 'n/a'`)
   if err != nil {
      return err, nil
   }
   defer stmt.Close()

   rows, err := stmt.Query(user.GetEmail())
   if err != nil {
      return err, nil
   }
   defer rows.Close()

   return getProofsFromRows(rows)
}

func (p *ProofStore) GetUserCompletedProofs(user UserWithEmail) (error, []Proof) {
   stmt, err := p.db.Prepare(`SELECT id, entryType, userSubmitted, proofName, proofType, Premise, 
                              Logic, Rules, proofCompleted, timeSubmitted, Conclusion, repoProblem 
                              FROM proof WHERE userSubmitted = ? AND proofCompleted = 'true'`)
   if err != nil {
      return err, nil
   }
   defer stmt.Close()

   rows, err := stmt.Query(user.GetEmail())
   if err != nil {
      return err, nil
   }
   defer rows.Close()

   return getProofsFromRows(rows)
}

func (p *ProofStore) Store(proof Proof) error {
   tx, err := p.db.Begin()
   if err != nil {
      return errors.New("Database transaction begin error")
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
      return errors.New("Transaction prepare error")
   }

   PremiseJSON, err := json.Marshal(proof.Premise)
   if err != nil {
      return errors.New("Premise marshal error")
   }
   LogicJSON, err := json.Marshal(proof.Logic)
   if err != nil {
      return errors.New("Logic marshal error")
   }
   RulesJSON, err := json.Marshal(proof.Rules)
	if err != nil {
		return errors.New("Rules marshal error")
	}
   _, err = stmt.Exec(proof.EntryType, proof.UserSubmitted, proof.ProofName, proof.ProofType,
                     PremiseJSON, LogicJSON, RulesJSON, proof.ProofCompleted, proof.Conclusion, 
                     proof.RepoProblem, proof.EntryType, proof.ProofType, PremiseJSON, LogicJSON, 
                     RulesJSON, proof.ProofCompleted, proof.Conclusion, proof.RepoProblem)
   if err != nil {
      return errors.New("Statement exec error")
   }
   tx.Commit()

   return nil
}

// ===== New Functions and Structs Spring Capstone 2022 =====

// still using the previous proof table for all proofs and arguments
// have not adapted to use separate argument table yet!
// type Argument struct {
//    Id             string   // SQL ID
//    UserSubmitted  string	// Used for results, ignored on user input
//    ProofName      string   // user-chosen name (repo problems start with 'Repository - ')
//    ProofType      string   // 'prop' (propositional/tfl) or 'fol' (first order logic)
//    Premise        []string // premises of the proof; an array of WFFs
//    Conclusion     string   // conclusion of the proof
//    TimeSubmitted  string
// }

// clear all proofs from proof table, retain arguments
func (p *ProofStore) EmptyProofTable() error {
	_, err := p.db.Exec(`DELETE FROM proof WHERE entryType = 'proof';`)
	return err
}

// clear all users from user table, retain admins
func (p *ProofStore) EmptyUserTable() error {
	_, err := p.db.Exec(`DELETE FROM user WHERE admin = 0;`)
	return err
}

type User struct {
   email string
   firstName string
   lastName string
   admin int
}

type Section struct {
   instructorEmail string
   name string
}

type Roster struct {
   sectionName string
   userEmail string
   role string
}

type Display interface {
   Display() string
}

func (user User) Display() (string) {
   return fmt.Sprintf("User: %s %s, %s, %d", user.firstName, user.lastName, user.email, user.admin)
}

func (section Section) Display() (string) {
   return fmt.Sprintf("Section: %s, %s", section.name, section.instructorEmail)
}

func (roster Roster) Display() (string) {
   return fmt.Sprintf("Roster: %s, %s, %s", roster.sectionName, roster.userEmail, roster.role)
}

func (p *ProofStore) MaintainAdmins(admins map[string]bool) {
   currentAdmins := p.GetAdmins()
   newAdmin := true
   updateUserSQL1 := `UPDATE user SET admin = 1 WHERE email = ?;`
   updateUserSQL0 := `UPDATE user SET admin = 0 WHERE email = ?;`

   for email, isAdmin := range admins {
      for _, currA := range currentAdmins {
         if email == currA {
            newAdmin = false
            break
         }
      }
      if isAdmin {
         if !newAdmin {
            result, err := p.db.Exec(updateUserSQL1, email)
            numUpdated, _ :=result.RowsAffected()
            if (err != nil) || (numUpdated != 1) {
               log.Printf("error during MaintainAdmins: '%s' for %s\n", err.Error(), email)
            }
         } else {
            p.InsertUser(User{email: email, firstName: "", lastName: "", admin:1})
         }
         // log.Printf("admin: %s\n", email)
      } else {
         if !newAdmin {
            result, err := p.db.Exec(updateUserSQL0, email)
            numUpdated, _ :=result.RowsAffected()
            if (err != nil) || (numUpdated != 1) {
               log.Printf("error during MaintainAdmins: '%s' for %s", err.Error(), email)
            }
         }
         // log.Printf("guest: %s\n", email)
      }
      newAdmin = true
   }
}

// pass a db reference connection from main to method with additional parameters
func (p *ProofStore) InsertUser(user User) (error){
   // log.Println("Inserting user record. . .")
   insertUserSQL := `INSERT INTO user(email, firstName, lastName, admin) VALUES (?, ?, ?, ?);`
   statement, err := p.db.Prepare(insertUserSQL)
   if err != nil {
      log.Println("error: InsertUser: preparing insertUserSQL statement")
      log.Println("-- ", err.Error())
      return err
   }
   defer statement.Close()

   _, err = statement.Exec(user.email, user.firstName, user.lastName, user.admin)
   if err != nil {
      log.Println("error: InsertUser: executing insertUserSQL statement")
      log.Println("-- ", err.Error())
      return err
   }
   return nil
}

func (p *ProofStore) InsertSection(section Section) (error) {
   // log.Println("Inserting section record. . .")
   insertSectionSQL := `INSERT INTO section(instructorEmail, name) VALUES (?, ?);`
   statement, err := p.db.Prepare(insertSectionSQL)
   if err != nil {
      log.Println("error: InsertSection: db.Prepare(insertSectionSQL)")
      log.Println("-- ", err.Error())
      return err
   }
   defer statement.Close()

   _, err = statement.Exec(section.instructorEmail, section.name)
   if err != nil {
      log.Println("error: InsertSection: statement.Exec(instructorEmail, name)")
      log.Println("-- ", err.Error())
      return err
   }
   return nil
}

func (p *ProofStore) InsertRoster(rosterRow Roster) (error) {
   // log.Println("Inserting roster record. . .")
   insertRosterSQL := `INSERT INTO roster(sectionName, userEmail, role) VALUES (?, ?, ?);`
   statement, err := p.db.Prepare(insertRosterSQL)
   if err != nil {
      log.Println("error: InsertRoster: preparation of insertRosterSQL statement")
      log.Fatalln("-- ", err.Error())
      return err
   }
   defer statement.Close()

   _, err = statement.Exec(rosterRow.sectionName, rosterRow.userEmail, rosterRow.role)
   if err != nil {
      log.Println("error: InsertRoster: execution of insertRosterSQL statement")
      log.Println("-- ", err.Error())
      return err
   }
   return nil
}

func (p *ProofStore) RemoveSection(sectionName string) (error) {
   // log.Println("Deleting section record. . .")
   RemoveSectionSQL := `DELETE From section where name = ?;`
   statement, err := p.db.Prepare(RemoveSectionSQL)
   if err != nil {
      log.Println("error: RemoveSection: preparation of RemoveSectionSQL statement")
      log.Fatalln("-- ", err.Error())
      return err
   }
   defer statement.Close()

   _, err = statement.Exec(sectionName)
   if err != nil {
      log.Println("error: RemoveSection: execution of RemoveSectionSQL statement")
      log.Println("-- ", err.Error())
      return err
   }
   return nil
}

func (p *ProofStore) RemoveFromRoster(sectionName string, userEmail string) (error) {
   // log.Println("Deleting roster record. . .")
   RemoveFromRosterSQL := `DELETE From roster where sectionName = ? and userEmail = ?;`
   statement, err := p.db.Prepare(RemoveFromRosterSQL)
   if err != nil {
      log.Println("error: RemoveFromRoster: preparation of RemoveFromRosterSQL statement")
      log.Fatalln("-- ", err.Error())
      return err
   }
   defer statement.Close()

   _, err = statement.Exec(sectionName, userEmail)
   if err != nil {
      log.Println("error: RemoveFromRoster: execution of RemoveFromRosterSQL statement")
      log.Println("-- ", err.Error())
      return err
   }
   return nil
}

func (p *ProofStore) GetUsers() ([]User) {
   row, err := p.db.Query("Select * FROM user ORDER BY admin DESC, lastName;")
   if err != nil {
      log.Fatalln("error: GetUsers: ", err.Error())
   }
   defer row.Close()

   var users []User
   for row.Next() { // Iterate and fetch the records from result cursor
      var user User
      row.Scan(&user.email, &user.firstName, &user.lastName, &user.admin)
      // fmt.Printf("User: %s %s, %s, %d\n", user.firstName, user.lastName, user.email, user.admin)
      users = append(users, user)
   }
   return users
}

// return array of admin user emails
func (p *ProofStore) GetAdmins() ([]string) {
   row, err := p.db.Query(`SELECT email FROM user WHERE admin = 1 ORDER BY email;`)
   if err != nil {
      log.Fatalln("error: GetAdmins: ", err.Error())
   }
   defer row.Close()

   var admins []string
   for row.Next() {
      var admin string
      row.Scan(&admin)
      admins = append(admins, admin)
   }
   return admins
}

// return array of current sections
func (p *ProofStore) GetSections() ([]Section){
   row, err := p.db.Query(`SELECT firstName, lastName, user.email, name
                     FROM section JOIN user ON instructorEmail = user.email
                     ORDER BY name;`)
   if err != nil {
      log.Fatalln("error: GetSections: ", err.Error())
   }
   defer row.Close()

   var sections []Section
   for row.Next() { // Iterate and fetch the records from result cursor
      var user User
      var section Section
      row.Scan(&user.firstName, &user.lastName, &section.instructorEmail, &section.name)
      // log.Println("section name check: ", name)
      // fmt.Printf("Instructor: %s %s, email: %s, section: %s\n", user.firstName, user.lastName, section.instructorEmail, section.name)
      sections = append(sections, section)
   }
   return sections
}

func (p *ProofStore) GetRoster(sectionName string) ([]Roster) {
   selectSectionSql := `Select userEmail, role from roster where sectionName = ? order by role desc, userEmail`
   statement, err := p.db.Prepare(selectSectionSql)
   if err != nil {
      log.Println("--err during preparation of selectSectionSql statement")
      log.Fatalln("--", err.Error())
   }
   defer statement.Close()

   rows, err := statement.Query(sectionName)
   if err != nil {
      log.Println("err during execution of selectSectionSql statement")
      log.Println("--", err.Error())
   }
   defer rows.Close()

   var roster []Roster
   for rows.Next() { // Iterate and fetch the records from result cursor
      var rosterRow Roster
      rosterRow.sectionName = sectionName
      rows.Scan(&rosterRow.userEmail, &rosterRow.role)
      // log.Println("section name check: ", name)
      // fmt.Printf("Section: %s %s, email: %s, role: %s\n", rosterRow.section, rosterRow.userEmail, rosterRow.role)
      roster = append(roster, rosterRow)
   }
   return roster
}

func (p *ProofStore) getUser(email string) (*User, error) {
   var user User
   row := p.db.QueryRow("Select * from user where email = ?;", email).Scan(
      &user.firstName,
      &user.lastName,
      &user.email,
      &user.admin,
   )

   if row != nil {
      if errors.Is(row, sql.ErrNoRows) {
         return nil, ErrNotExists
      }
      return nil, row
   }
   return &user, nil
}

func (p *ProofStore) getSection(name string) (*Section, error) {
   var section Section
   err := p.db.QueryRow("Select * from section where name = ?;", name).Scan(
      &section.instructorEmail,
      &section.name,
   )

   if err != nil {
      if errors.Is(err, sql.ErrNoRows) {
         return nil, ErrNotExists
      }
      return nil, err
   }
   return &section, nil
}

func (p *ProofStore) PopuplateTestUsersSectionsRosters() {
	fmt.Println("\n========INSERT USER RECORDS========")
	userInfo := []User{
		{email: "psmithTEST@csumb.edu", firstName: "Paul", lastName: "Smith", admin: 1},
		{email: "rmarksTEST@csumb.edu", firstName: "Ryan", lastName: "Marks", admin: 0},
		{email: "lramirezTEST@csumb.edu", firstName: "LeAnne", lastName: "Ramirez", admin: 0}, 
		{email: "abookerTEST@csumb.edu", firstName: "Annette", lastName: "Booker", admin: 1}, 
		{email: "mpotterTEST@csumb.edu", firstName: "Maxwell", lastName: "Potter", admin: 0}, 
		{email: "jduboisTEST@csumb.edu", firstName: "Jeanne", lastName: "Dubois", admin: 0}, 
		{email: "gsloneTEST@csumb.edu", firstName: "Garrett", lastName: "Slone", admin: 1}, 
		{email: "t1deleteTEST@csumb.edu", firstName: "t1", lastName: "delete1", admin: 0}, 
		{email: "t2deleteTEST@csumb.edu", firstName: "t2", lastName: "delete2", admin: 0}, 
		{email: "t3deleteTEST@csumb.edu", firstName: "t3", lastName: "delete3", admin: 1},
	}

	for _,v := range userInfo {
		err := p.InsertUser(v)
		if err != nil {
			log.Printf("error from populateTest: InsertUser: %s for %s", err.Error(), v.email)
		}
	}

   sectionInfo := []Section{
      {
         instructorEmail: "abookerTEST@csumb.edu",
         name: "testSection 000-01",
      },
      {
         instructorEmail: "psmithTEST@csumb.edu",
         name: "testSection 000-02",
      },
      {
         instructorEmail: "psmithTEST@csumb.edu",
         name: "testSection 000-03",
      },
   }

	fmt.Println("\n========INSERT SECTION RECORDS========")
   for _,v := range sectionInfo {
		err := p.InsertSection(v)
		if err != nil {
			log.Printf("error from populateTest: InsertSection: %s for %s", err.Error(), v.instructorEmail)
		}
	}

   rosterInfo := []Roster{
      {
         sectionName: "testSection 000-01",
         userEmail: "abookerTEST@csumb.edu",
         role: "instructor",
      },
      {
		sectionName: "testSection 000-01",
         userEmail: "gsloneTEST@csumb.edu",
         role: "ta",
      },
      {
		sectionName: "testSection 000-01",
         userEmail: "mpotterTEST@csumb.edu",
         role: "student",
      },
	  {
		sectionName: "testSection 000-02",
		userEmail: "psmithTEST@csumb.edu",
		role: "instructor",
	 },
	 {
		sectionName: "testSection 000-01",
		userEmail: "jduboisTEST@csumb.edu",
		role: "student",
	 },
	 {
		sectionName: "testSection 000-02",
		userEmail: "t1deleteTEST@csumb.edu",
		role: "ta",
	 },
	 {
		sectionName: "testSection 000-02",
		userEmail: "t2deleteTEST@csumb.edu",
		role: "student",
	 },
	 {
		sectionName: "testSection 000-02",
		userEmail: "t3deleteTEST@csumb.edu",
		role: "student",
	 },
   }
	fmt.Println("\n========INSERT ROSTER RECORDS========")
	for _,v := range rosterInfo {
		err := p.InsertRoster(v)
		if err != nil {
			log.Printf("error from populateTest: InsertSection: %s for %s", err.Error(), v.userEmail)
		}
	}
	
}