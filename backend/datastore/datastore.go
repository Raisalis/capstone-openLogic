package datastore

import (
	"database/sql"
	"encoding/json"
	"errors"
   "fmt"
	"log"
   "strconv"
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
   EverCompleted  string   // 'true', 'false'
	ProofCompleted string   // 'true', 'false', or 'error'
	Conclusion     string   // conclusion of the proof
	RepoProblem    string   // 'true' if problem started from a repo problem, else 'false'
	TimeSubmitted  string
}

type SectionProofs struct {
   SectionName string
   ProofList []Proof
}

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
	EmptySectionTable() error
   EmptyRosterTable() error
	EmptyAssignmentTable() error
	InsertUser(user User) error
	InsertSection(section Section) error
	InsertRoster(rosterRow Roster) error
   InsertAssignment(assignment Assignment) error
   UpdateAssignment(currentName string, updatedAssignment Assignment) error
   GetAdmins() ([]string)
	GetAllAttemptedRepoProofs() (error, []Proof)
	GetRepoProofs(user UserWithEmail) (error, []SectionProofs)
	GetUserProofs(user UserWithEmail) (error, []Proof)
   GetUserArguments(user UserWithEmail) ([]Proof, error)
	GetUserCompletedProofs(user UserWithEmail) (error, []Proof)
   GetSections(userEmail string) ([]Section, error)
   GetRoster(sectionName string) ([]Roster, error)
   GetAssignmentsBySection(sectionName string) ([]Assignment, error)
   GetAssignmentProofs(assignment Assignment) ([]Proof, error)
   GetCompletedProofsBySection(sectionName string) ([]Proof, error)
   GetCompletedProofsByAssignment(sectionName string, assignmentName string) ([]Proof, error)
	PopulateTestUsersSectionsRosters()
	RemoveFromRoster(sectionName string, userEmail string) error
	RemoveSection(sectionName string) error
   RemoveAssignment(sectionName string, name string) error
   removeOneStudentsProofs(userEmail string) (error)
   removeAllStudentsProofs(sectionName string) (error)
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

		err := rows.Scan(&userProof.Id, &userProof.EntryType, &userProof.UserSubmitted, &userProof.ProofName, &userProof.ProofType, &PremiseJSON, &LogicJSON, &RulesJSON, &userProof.EverCompleted, &userProof.ProofCompleted, &userProof.TimeSubmitted, &userProof.Conclusion, &userProof.RepoProblem)
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

	_, err = p.db.Exec(`CREATE VIEW admin_repoproblems (userSubmitted, Premise, Conclusion) 
                        AS SELECT userSubmitted, Premise, Conclusion 
                        FROM proof WHERE userSubmitted IN (SELECT email FROM user WHERE admin = 1)`)
	if err != nil {
		return err, nil
	}
	stmt, err := p.db.Prepare(`SELECT id, entryType, proof.userSubmitted, proofName, proofType, proof.Premise, Logic, Rules, 
                                    everCompleted, proofCompleted, timeSubmitted, proof.Conclusion, repoProblem
								      FROM proof INNER JOIN admin_repoproblems 
                                 ON proof.Premise = admin_repoproblems.Premise AND proof.Conclusion = admin_repoproblems.Conclusion
                              WHERE entryType = 'proof'
                              ORDER BY proof.userSubmitted, proofName, proofCompleted;`)
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

// return the visibile assignment proofs and the corresponding section for a given user
// this is used for the "load repositoryproblems" dropdown
func (p *ProofStore) GetRepoProofs(user UserWithEmail) (error, []SectionProofs) {
   sections, err := p.GetSections(user.GetEmail())
   if err != nil {
		return err, nil
	}

   // stmt, err := p.db.Prepare(`SELECT id, entryType, userSubmitted, proofName, proofType, Premise, Logic, Rules, everCompleted, proofCompleted, timeSubmitted, Conclusion, repoProblem 
   //                            FROM proof JOIN section on userSubmitted = instructorEmail
   //                            WHERE repoProblem = 'true' AND userSubmitted IN (SELECT instructorEmail FROM section WHERE name = ?) 
   //                            ORDER BY proofName`)
   // if err != nil {
   //    return err, nil
   // }
   // defer stmt.Close()

   var repoList []SectionProofs
   var sectionProofList SectionProofs
   var sectionAssignments []Assignment
   var assignmentProofs []Proof
   for _,section:= range sections {
      sectionAssignments = nil
      // assignmentProofs = nil
      // sectionProofList = nil
      sectionProofList.SectionName = section.Name

      // rows, err := stmt.Query(section.Name)
      // if err != nil {
      //    return err, nil
      // }
      // defer rows.Close()

      // _,sectionProofList.ProofList = getProofsFromRows(rows)
      // repoList = append(repoList, sectionProofList)

      sectionAssignments, err = p.GetAssignmentsBySection(section.Name)
      if err == nil {
         for _,assignment := range sectionAssignments {
            if assignment.Visibility == "true" {
               assignmentProofs, err = p.GetAssignmentProofs(assignment)
               sectionProofList.ProofList = append(sectionProofList.ProofList, assignmentProofs...)
            }
         }
         repoList = append(repoList, sectionProofList)
      }
   }

   return nil, repoList
}

func (p *ProofStore) GetUserProofs(user UserWithEmail) (error, []Proof) {
	stmt, err := p.db.Prepare(`SELECT id, entryType, userSubmitted, proofName, proofType, Premise, Logic, Rules, everCompleted, proofCompleted, timeSubmitted, Conclusion, repoProblem 
                              FROM proof WHERE userSubmitted = ? AND everCompleted = 'false' AND proofCompleted != 'true' AND proofName != 'n/a' AND proofName NOT LIKE '%Test%' AND proofName NOT LIKE '%Quiz%' AND proofName NOT LIKE '%Final%'`)
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


func (p *ProofStore) GetUserArguments(user UserWithEmail) ([]Proof, error) {
   stmt, err := p.db.Prepare(`SELECT id, entryType, userSubmitted, proofName, proofType, Premise, Logic, Rules, everCompleted, proofCompleted, timeSubmitted, Conclusion, repoProblem 
                              FROM proof WHERE userSubmitted = ? AND entryType = 'argument';`)
	if err != nil {
		return  nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(user.GetEmail())
	if err != nil {
      return  nil, err
	}
	defer rows.Close()

   err, arguments := getProofsFromRows(rows)
   if err != nil {
      return  nil, err
	}

	return arguments, nil
}

func (p *ProofStore) GetUserCompletedProofs(user UserWithEmail) (error, []Proof) {
	stmt, err := p.db.Prepare(`SELECT id, entryType, userSubmitted, proofName, proofType, Premise, Logic, Rules, everCompleted, proofCompleted, timeSubmitted, Conclusion, repoProblem 
                              FROM proof WHERE userSubmitted = ? AND proofCompleted = 'true' AND proofName NOT LIKE '%Test%' AND proofName NOT LIKE '%Quiz%' AND proofName NOT LIKE '%Final%';`)
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
	stmt, err := tx.Prepare(`INSERT INTO proof (entryType,
							userSubmitted,
							proofName,
							proofType,
							Premise,
							Logic,
							Rules,
                     everCompleted,
							proofCompleted,
							timeSubmitted,
							Conclusion,
							repoProblem)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), ?, ?)
				 ON CONFLICT (userSubmitted, proofName, proofCompleted) DO UPDATE SET
					 	entryType = ?,
					 	proofType = ?,
					 	Premise = ?,
					 	Logic = ?,
					 	Rules = ?,
                  everCompleted = ?,
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

   if proof.EverCompleted == "" {
      proof.EverCompleted = "false"
   }
	_, err = stmt.Exec(proof.EntryType, proof.UserSubmitted, proof.ProofName, proof.ProofType,
      PremiseJSON, LogicJSON, RulesJSON, proof.EverCompleted, proof.ProofCompleted, proof.Conclusion, 
      proof.RepoProblem, proof.EntryType, proof.ProofType, PremiseJSON, LogicJSON, 
      RulesJSON, proof.EverCompleted, proof.Conclusion, proof.RepoProblem)
	if err != nil {
		return errors.New("Statement exec error")
	}
	tx.Commit()

	return nil
}

// ===== New Functions and Structs Spring Capstone 2022 =====

// clear all proofs from proof table, retain arguments
func (p *ProofStore) EmptyProofTable() error {
	_, err := p.db.Exec(`DELETE FROM proof WHERE entryType = 'proof';`)
	return err
}

func (p *ProofStore) EmptyUserTable() error {
	_, err := p.db.Exec(`DELETE FROM user WHERE admin = 0;`)
	return err
}

func (p *ProofStore) EmptySectionTable() error {
	_, err := p.db.Exec(`DELETE FROM section;`)
	return err
}

func (p *ProofStore) EmptyRosterTable() error {
	_, err := p.db.Exec(`DELETE FROM roster;`)
	return err
}

func (p *ProofStore) EmptyAssignmentTable() error {
	_, err := p.db.Exec(`DELETE FROM assignment;`)
	return err
}

type User struct {
   Email string
   FirstName string
   LastName string
   Admin int
}

type Section struct {
   InstructorEmail string
   Name string
}

type Roster struct {
   SectionName string
   UserEmail string
   Role string
}

type RosterAndProof struct {
   RowRoster Roster
   RowProof Proof
}

type Assignment struct {
   SectionName string
   Name string
   ProofIds string
   Visibility string
}

type Display interface {
   Display() string
}

func (user User) Display() (string) {
   return fmt.Sprintf("User: %s %s, %s, %d", user.FirstName, user.LastName, user.Email, user.Admin)
}

func (roster Roster) Display() (string) {
   return fmt.Sprintf("Roster: %s, %s, %s", roster.SectionName, roster.UserEmail, roster.Role)
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
            p.InsertUser(User{Email: email, FirstName: "", LastName: "", Admin:1})
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
   // check if user already exists, if so return nil
   _, err := p.getUser(user.Email)
   if err == nil {
      return nil
   }

   insertUserSQL := `INSERT INTO user(email, firstName, lastName, admin) VALUES (?, ?, ?, ?);`
   statement, err := p.db.Prepare(insertUserSQL)
   if err != nil {
      log.Println("error: InsertUser: preparing insertUserSQL statement")
      log.Println("-- ", err.Error())
      return err
   }
   defer statement.Close()

   _, err = statement.Exec(user.Email, user.FirstName, user.LastName, user.Admin)
   if err != nil {
      log.Println("error: InsertUser: executing insertUserSQL statement")
      log.Println("-- ", err.Error())
      return err
   }
   return nil
}

func (p *ProofStore) InsertSection(section Section) (error) {
   // log.Println("Inserting section record. . .")
   if section.Name == "" {
      err := errors.New("section insertion err: no name given")
      return err
   }

   insertSectionSQL := `INSERT INTO section(instructorEmail, name) VALUES (?, ?);`
   statement, err := p.db.Prepare(insertSectionSQL)
   if err != nil {
      log.Println("error: InsertSection: db.Prepare(insertSectionSQL)")
      log.Println("-- ", err.Error())
      return err
   }
   defer statement.Close()

   _, err = statement.Exec(section.InstructorEmail, section.Name)
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
      log.Println("-- ", err.Error())
      return err
   }
   defer statement.Close()

   _, err = statement.Exec(rosterRow.SectionName, rosterRow.UserEmail, rosterRow.Role)
   if err != nil {
      log.Println("error: InsertRoster: execution of insertRosterSQL statement")
      log.Println("-- ", err.Error())
      return err
   }
   return nil
}

func (p *ProofStore) InsertAssignment(assignment Assignment) (error){
   insertAssignmentSQL := `INSERT INTO assignment(sectionName, name, proofIds, visibility) VALUES (?, ?, ?, ?);`
   statement, err := p.db.Prepare(insertAssignmentSQL)
   if err != nil {
      log.Println("error: InsertAssignment: preparation of insertAssignmentSQL statement")
      log.Println("-- ", err.Error())
      return err
   }
   defer statement.Close()

   _, err = statement.Exec(assignment.SectionName, assignment.Name, assignment.ProofIds, assignment.Visibility)
   if err != nil {
      log.Println("error: InsertAssignment: execution of insertAssignmentSQL statement")
      log.Println("-- ", err.Error())
      return err
   }
   return nil
}

func (p *ProofStore) UpdateAssignment(currentName string, updatedAssignment Assignment) (error) {
   updateAssignmentSQL := `UPDATE assignment SET name = ?, proofIds = ?, visibility = ?
                           WHERE name = ? and sectionName = ?;`
   statement, err := p.db.Prepare(updateAssignmentSQL)
   if err != nil {
      log.Println("error: UpdateAssignment: preparation of updateAssignmentSQL statement")
      log.Println("-- ", err.Error())
      return err
   }
   defer statement.Close()

   _, err = statement.Exec(updatedAssignment.Name, updatedAssignment.ProofIds, updatedAssignment.Visibility,
                           currentName, updatedAssignment.SectionName)
   if err != nil {
      log.Println("error: UpdateAssignment: execution of updateAssignmentSQL statement")
      log.Println("-- ", err.Error())
      return err
   }

   return nil
}

func (p *ProofStore) RemoveSection(sectionName string) (error) {
   err := p.removeAllStudentsProofs(sectionName)
   if err != nil {
      return err
   }
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
   err := p.removeOneStudentsProofs(userEmail)
   if err != nil {
      return err
   }
   // log.Println("Deleting roster record. . .")
   RemoveFromRosterSQL := `DELETE From roster where sectionName = ? and userEmail = ?;`
   statement, err := p.db.Prepare(RemoveFromRosterSQL)
   if err != nil {
      log.Println("error: RemoveFromRoster: preparation of RemoveFromRosterSQL statement")
      log.Println("-- ", err.Error())
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

func (p *ProofStore) RemoveAssignment(sectionName string, name string) (error) {
   // log.Println("Deleting assignment record. . .")
   RemoveAssignmentSQL := `DELETE FROM assignment where sectionName = ? and name = ?;`
   statement, err := p.db.Prepare(RemoveAssignmentSQL)
   if err != nil {
      log.Println("error: RemoveAssignment: preparation of RemoveAssignmentSQL statement")
      log.Println("-- ", err.Error())
      return err
   }
   defer statement.Close()

   _, err = statement.Exec(sectionName, name)
   if err != nil {
      log.Println("error: RemoveAssignment: execution of RemoveAssignmentSQL statement")
      log.Println("-- ", err.Error())
      return err
   }
   return nil
}

// remove all assignment proofs associated with a given userEmail
func (p *ProofStore) removeOneStudentsProofs(userEmail string) (error) {
   removeProofsSQL := `DELETE FROM proof WHERE userSubmitted = ? AND entryType = 'proof' AND repoProof = 'true';`
   statement, err := p.db.Prepare(removeProofsSQL)
   result, err := statement.Exec(userEmail)
   if err != nil {
      log.Println("error: removeOneStudentsProofs: ", err.Error())
      return err
   }
   defer rows.Close()
   return nil
}

// remove all assignment proofs associated with userEmails that are connected to a sectionName via roster
func (p *ProofStore) removeAllStudentsProofs(sectionName string) (error) {
   removeProofsSQL := `DELETE FROM proof WHERE userSubmitted IN (SELECT userEmail FROM roster WHERE sectionName = ? AND role != "instructor") AND repoProof = 'true';`
   statement, err := p.db.Prepare(removeProofsSQL)
   result, err := statement.Exec(sectionName)
   if err != nil {
      log.Println("error: removeAllStudentsProofs: ", err.Error())
      return err
   }
   defer rows.Close()
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
      row.Scan(&user.Email, &user.FirstName, &user.LastName, &user.Admin)
      // fmt.Printf("User: %s %s, %s, %d\n", user.FirstName, user.LastName, user.Email, user.Admin)
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
func (p *ProofStore) GetSections(userEmail string) ([]Section, error){
   statement, err := p.db.Prepare(`SELECT instructorEmail, name FROM section JOIN roster ON section.name = roster.sectionName 
                                    WHERE roster.userEmail = ?`)
   if err != nil {
      log.Printf(`error: GetSections: preparation of getSectionsSQL statement
                  -- %s`, err.Error())
      return nil, err
   }
   defer statement.Close()

   rows, err := statement.Query(userEmail)
   if err != nil {
      log.Printf(`error: GetSections: Query of getSectionsSQL statement
                  -- %s`, err.Error())
      return nil, err
   }
   defer rows.Close()
   
   var sections []Section
   for rows.Next() { // Iterate and fetch the records from result cursor
      var section Section
      rows.Scan(&section.InstructorEmail, &section.Name)
      log.Printf("section scan check: %+v", section)
      sections = append(sections, section)
   }
   return sections, nil
}

// get students and tas from roster for a given section name
func (p *ProofStore) GetRoster(sectionName string) ([]Roster, error) {
   selectRoserSql := `SELECT userEmail, role FROM roster WHERE sectionName = ? AND role != "instructor" ORDER BY role, userEmail`
   statement, err := p.db.Prepare(selectRoserSql)
   if err != nil {
      log.Printf(`error: GetRoster: during preparation of selectRoserSql statement
                  -- %s`, err.Error())
      return nil, err
   }
   defer statement.Close()

   rows, err := statement.Query(sectionName)
   if err != nil {
      log.Printf(`error: GetRoster: during execution of selectRoserSql statement
                  -- %s`, err.Error())
      return nil, err
   }
   defer rows.Close()

   var roster []Roster
   for rows.Next() { // Iterate and fetch the records from result cursor
      var rosterRow Roster
      rosterRow.SectionName = sectionName
      rows.Scan(&rosterRow.UserEmail, &rosterRow.Role)
      // log.Println("roster Scan check: %+v", rosterRow)
      roster = append(roster, rosterRow)
   }
   return roster, nil
}

func (p *ProofStore) GetAssignmentsBySection(sectionName string) ([]Assignment, error) {
   selectAssignmentsSQL := `SELECT * FROM assignment WHERE sectionName = ?;`
   statement, err := p.db.Prepare(selectAssignmentsSQL)
   defer statement.Close()

   rows, err := statement.Query(sectionName)
   if err != nil {
      log.Printf(`error: GetAssignmentsBySection: during execution of selectAssignmentsSQL statement
                  -- %s`, err.Error())
      return nil, err
   }
   defer rows.Close()

   var assignments []Assignment
   for rows.Next() { 
      var assign Assignment
      rows.Scan(&assign.SectionName, &assign.Name, &assign.ProofIds, &assign.Visibility)
      log.Println("assignment Scan check: %+v", assign)
      assignments = append(assignments, assign)
   }
   return assignments, nil
}

func (p *ProofStore) GetAssignmentProofs(assignment Assignment) ([]Proof, error) {
   // get proofs by ids in assignment.proofIds
   log.Println("in datastore: GetAssignmentProofs")
   log.Printf("assignment: %s > %s > %s", assignment.SectionName, assignment.Name, assignment.ProofIds)

   var proofIdIntegers []int
   for _,v := range assignment.ProofIds {
      if string(v) != "[" && string(v) != "]" && string(v) != " " && string(v) != ","{
         id,_:= strconv.Atoi(string(v))
         proofIdIntegers = append(proofIdIntegers, id)
      }
   }
   log.Println("proof ids for selectProofsSQL: ", proofIdIntegers)

   selectProofsSQL := `SELECT * FROM proof WHERE id = ?;`
   statement, err := p.db.Prepare(selectProofsSQL)
   if err != nil {
      log.Printf(`error: GetAssignmentProofs: during selectProofSQL prep
                  -- %s`, err.Error())
      return nil, err
   }
   defer statement.Close()

   var proofs []Proof
   for _,v := range proofIdIntegers {
      rows, err := statement.Query(v)
      if err != nil {
         log.Printf(`error: GetAssignmentProofs: during selectProofSQL execution
                     -- %s`, err.Error())
         return nil, err
      }
      defer rows.Close()

      err, proof := getProofsFromRows(rows)
      if err != nil {
         log.Printf(`error: GetAssignmentProofs: during rows conversion
                     -- %s`, err.Error())
         return nil, err
      }
      proofs = append(proofs, proof[0])
   }
   

   return proofs, nil
}

func (p *ProofStore) GetCompletedProofsBySection(sectionName string) ([]Proof, error) {
   selectProofsSQL := `SELECT * FROM roster JOIN proof ON userEmail = userSubmitted
                        WHERE sectionName = ? AND role = 'student' AND entryType = 'proof' AND everCompleted = 'true' AND proofCompleted = 'true' AND repoProblem = 'true'
                        ORDER BY userEmail;`
   statement, err := p.db.Prepare(selectProofsSQL)
   if err != nil {
      log.Printf(`error: GetCompletedProofsBySection: during preparation of selectProofsSQL statement
                  -- %s`, err.Error())
      return nil, err
   }
   defer statement.Close()

   rows, err := statement.Query(sectionName)
   if err != nil {
      log.Printf(`error: GetCompletedProofsBySection: during execution of selectProofsSQL statement
                  -- %s`, err.Error())
      return nil, err
   }
   defer rows.Close()

   var completedProofs []Proof
   for rows.Next() {
      var row RosterAndProof // this is just to grab all values in returned row for ease. . .
      var PremiseJSON string
      var LogicJSON string
      var RulesJSON string
      rows.Scan(&row.RowRoster.SectionName, &row.RowRoster.UserEmail, &row.RowRoster.Role, &row.RowProof.Id, &row.RowProof.EntryType, 
                  &row.RowProof.UserSubmitted, &row.RowProof.ProofName, &row.RowProof.ProofType, &PremiseJSON, &LogicJSON, &RulesJSON, 
                  &row.RowProof.EverCompleted, &row.RowProof.ProofCompleted, &row.RowProof.TimeSubmitted, &row.RowProof.Conclusion, &row.RowProof.RepoProblem)

      if err = json.Unmarshal([]byte(PremiseJSON), &row.RowProof.Premise); err != nil {
         return nil, err
      }
      if err = json.Unmarshal([]byte(LogicJSON), &row.RowProof.Logic); err != nil {
         return nil, err
      }
      if err = json.Unmarshal([]byte(RulesJSON), &row.RowProof.Rules); err != nil {
         return nil, err
      }

      completedProofs = append(completedProofs, row.RowProof) // only keep the proof data
   }

   return completedProofs, nil
}

func (p *ProofStore) GetCompletedProofsByAssignment(sectionName string, assignmentName string) ([]Proof, error) {
   getProofIds := `SELECT proofIds FROM assignment WHERE sectionName = ? AND name = ?;`
   statement, err := p.db.Prepare(getProofIds)
   if err != nil {
      log.Printf(`error: GetCompletedProofsByAssignment: during preparation of getProofIds statement
                  -- %s`, err.Error())
      return nil, err
   }
   defer statement.Close()

   rows, err := statement.Query(sectionName, assignmentName)
   if err != nil {
      log.Printf(`error: GetCompletedProofsByAssignment: during execution of getProofIds statement
                  -- %s`, err.Error())
      return nil, err
   }
   defer rows.Close()

   var proofIdsString string
   for rows.Next() {
      rows.Scan(&proofIdsString)
   }
   log.Printf("GetCompletedProofsByAssignment: proofIdsString: %s", proofIdsString)
   // -----
   var assignmentDetails Assignment
   assignmentDetails.SectionName = sectionName
   assignmentDetails.Name = assignmentName
   assignmentDetails.ProofIds = proofIdsString

   assignmentProofs, err := p.GetAssignmentProofs(assignmentDetails)
   // -----
   selectProofsSQL := `SELECT * FROM proof WHERE userSubmitted IN (SELECT userEmail FROM roster WHERE sectionName = ? and role = 'student')
                        AND entryType = 'proof' AND everCompleted = 'true' AND proofCompleted = 'true' AND repoProblem = 'true'
                        ORDER BY userSubmitted, proofName;`
   statement, err = p.db.Prepare(selectProofsSQL)
   if err != nil {
      log.Printf(`error: GetCompletedProofsByAssignment: during preparation of selectProofsSQL statement
                  -- %s`, err.Error())
      return nil, err
   }

   rows, err = statement.Query(sectionName)
   if err != nil {
      log.Printf(`error: GetCompletedProofsByAssignment: during execution of selectProofsSQL statement
                  -- %s`, err.Error())
      return nil, err
   }

   err, allProofs := getProofsFromRows(rows)
   if err != nil {
      log.Printf(`error: GetCompletedProofsByAssignment: during copnversion of rows for allProofs
                  -- %s`, err.Error())
      return nil, err
   }

   log.Println("-----")
   log.Printf("assignmentProofs: %+v", assignmentProofs)
   log.Println("-----")
   log.Printf("allProofs: %+v", allProofs)
   log.Println("-----")
   
   var completedAssignedProofs []Proof
   for _,v1 := range allProofs {
      for _,v2 := range assignmentProofs {
         if (v1.ProofName == v2.ProofName) && (v1.Conclusion == v2.Conclusion) {
            // append v1 to completedAssignedProofs
            completedAssignedProofs = append(completedAssignedProofs, v1)
         }
      }
   }
   log.Printf("completedAssignedProofs: %+v", allProofs)
   log.Println("-----")
   return completedAssignedProofs, nil
}

func (p *ProofStore) getUser(email string) (*User, error) {
   var user User
   row := p.db.QueryRow("Select * from user where email = ?;", email).Scan(
      &user.FirstName,
      &user.LastName,
      &user.Email,
      &user.Admin,
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
      &section.InstructorEmail,
      &section.Name,
   )

   if err != nil {
      if errors.Is(err, sql.ErrNoRows) {
         return nil, ErrNotExists
      }
      return nil, err
   }
   return &section, nil
}

func (p *ProofStore) PopulateTestUsersSectionsRosters() {
	fmt.Println("\n========INSERT USER RECORDS========")
	userInfo := []User{
		{Email: "psmithTEST@csumb.edu", FirstName: "Paul", LastName: "Smith", Admin: 1},
		{Email: "rmarksTEST@csumb.edu", FirstName: "Ryan", LastName: "Marks", Admin: 0},
		{Email: "lramirezTEST@csumb.edu", FirstName: "LeAnne", LastName: "Ramirez", Admin: 0}, 
		{Email: "abookerTEST@csumb.edu", FirstName: "Annette", LastName: "Booker", Admin: 1}, 
		{Email: "mpotterTEST@csumb.edu", FirstName: "Maxwell", LastName: "Potter", Admin: 0}, 
		{Email: "jduboisTEST@csumb.edu", FirstName: "Jeanne", LastName: "Dubois", Admin: 0}, 
      {Email: "jdoeTEST@csumb.edu", FirstName: "John", LastName: "Doe", Admin: 0}, 
      {Email: "sadamsTEST@csumb.edu", FirstName: "Steven", LastName: "Adams", Admin: 0}, 
		{Email: "gsloneTEST@csumb.edu", FirstName: "Garrett", LastName: "Slone", Admin: 1}, 
		{Email: "t1deleteTEST@csumb.edu", FirstName: "t1", LastName: "delete1", Admin: 0}, 
		{Email: "t2deleteTEST@csumb.edu", FirstName: "t2", LastName: "delete2", Admin: 0}, 
		{Email: "t3deleteTEST@csumb.edu", FirstName: "t3", LastName: "delete3", Admin: 1},
      {Email: "mkammerer@csumb.edu", FirstName: "Michael", LastName: "Kammerer", Admin: 1},
      {Email: "jasbaker@csumb.edu", FirstName: "Jason", LastName: "Baker", Admin: 1},
	}

   for _,v := range userInfo {
		err := p.InsertUser(v)
		if err != nil {
			log.Printf("error from populateTest: InsertUser: %s for %s", err.Error(), v.Email)
		}
	}

   sectionInfo := []Section{
      {
         InstructorEmail: "jasbaker@csumb.edu",
         Name: "Baker Section",
      },
      {
         InstructorEmail: "bkondo@csumb.edu",
         Name: "Kondo Section",
      },
      {
         InstructorEmail: "mkammerer@csumb.edu",
         Name: "Kammerer Section",
      },
      {
         InstructorEmail: "elarson@csumb.edu",
         Name: "Larson Section",
      },
   }

	fmt.Println("\n========INSERT SECTION RECORDS========")
   for _,v := range sectionInfo {
		err := p.InsertSection(v)
		if err != nil {
			log.Printf("error from populateTest: InsertSection: %s for %s", err.Error(), v.InstructorEmail)
		}
	}

   rosterInfo := []Roster{
      {
         SectionName: "Baker Section",
         UserEmail: "jasbaker@csumb.edu",
         Role: "instructor",
      },
      {
         SectionName: "Baker Section",
         UserEmail: "gsloneTEST@csumb.edu",
         Role: "ta",
      },
      {
         SectionName: "Baker Section",
         UserEmail: "elarson@csumb.edu",
         Role: "student",
      },
	  {
		SectionName: "Kondo Section",
		UserEmail: "bkondo@csumb.edu",
		Role: "instructor",
	 },
	 {
		SectionName: "Kondo Section",
		UserEmail: "mkammerer@csumb.edu",
		Role: "student",
	 },
	 {
		SectionName: "Kondo Section",
		UserEmail: "t1deleteTEST@csumb.edu",
		Role: "ta",
	 },
	 {
		SectionName: "Kondo Section",
		UserEmail: "elarson@csumb.edu",
		Role: "student",
	 },
    {
		SectionName: "Kammerer Section",
		UserEmail: "mkammerer@csumb.edu",
		Role: "instructor",
	 },
    {
		SectionName: "Kammerer Section",
		UserEmail: "psmithTEST@csumb.edu",
		Role: "ta",
	 },
    {
		SectionName: "Kammerer Section",
		UserEmail: "bkondo@csumb.edu",
		Role: "student",
	 },
    
    {
		SectionName: "Kammerer Section",
		UserEmail: "lramirezTEST@csumb.edu",
		Role: "student",
	 },
    {
		SectionName: "Larson Section",
		UserEmail: "elarson@csumb.edu",
		Role: "instructor",
	 },
    {
		SectionName: "Kammerer Section",
		UserEmail: "abookerTEST@csumb.edu",
		Role: "ta",
	 },
    {
		SectionName: "Larson Section",
		UserEmail: "bkondo@csumb.edu",
		Role: "student",
	 },
    {
		SectionName: "Larson Section",
		UserEmail: "jduboisTEST@csumb.edu",
		Role: "student",
	 },
   }
	fmt.Println("\n========INSERT ROSTER RECORDS========")
	for _,v := range rosterInfo {
		err := p.InsertRoster(v)
		if err != nil {
         log.Printf("error from populateTest: InsertRoster: %s for %s, %s", err.Error(), v.SectionName, v.UserEmail)
      }
   }

   assignmentInfo := []Assignment{
      {
         SectionName: "Larson Section",
         Name: "L Test assignment",
         ProofIds: "[1, 4]",
         Visibility: "true",
      },
      {
         SectionName: "Kondo Section",
         Name: "K Test assignment",
         ProofIds: "[4]",
         Visibility: "true",
      },
   }
   fmt.Println("\n========INSERT Assignment RECORDS========")
	for _,v := range assignmentInfo {
		err := p.InsertAssignment(v)
		if err != nil {
         log.Printf("error from populateTest: InsertAssignment: %s for %s", err.Error(), v.Name)
      }
   }

   fmt.Println("\n========INSERTIONS COMPLETED========\n")
} 