package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	"datastore"
	tokenauth "google-token-auth"
)

var (
	// Set allowed domains here.
	authorized_domains = []string{
		"csumb.edu",
	}

	// Client-side client ID from your Google Developer Console
	// Same as in the front-end index.php
	authorized_client_ids = []string{
		"91166551948-u7j6ip1827e1cgf9fvafu1m2mgb529b9.apps.googleusercontent.com",
	}

	// admins must be marked as false to remove their status from user table
	// - backend must run once to update new admin variables, then they may be removed from admin_users map
	// - (should change this in the future, so that removing admins is less confusing)
	admin_users = map[string]bool{
		"abiblarz@csumb.edu":  true,
		"sislam@csumb.edu":    true,
		"gbruns@csumb.edu":    true,
		"cohunter@csumb.edu":  true,
		"bkondo@csumb.edu":    false,
		"elarson@csumb.edu":   true,
		"jasbaker@csumb.edu":  false,
		"mkammerer@csumb.edu": true,
	}

	// When started via systemd, WorkingDirectory is set to one level above the public_html directory
	database_uri = "file:db.sqlite3?cache=shared&_foreign_keys=on&mode=rwc&_journal_mode=WAL"
)

type userWithEmail interface {
	GetEmail() string
}

type Env struct {
	ds datastore.IProofStore
}

type FailedRoster struct {
	email    string
	errorMsg string
}

// return a list of current admin emails
func (env *Env) getAdmins(w http.ResponseWriter, req *http.Request) {
	type adminUsers struct {
		Admins []string
	}
	var admins adminUsers
	// for adminEmail := range admin_users {
	// 	admins.Admins = append(admins.Admins, adminEmail)
	// }

	admins.Admins = env.ds.GetAdmins()

	output, err := json.Marshal(admins)
	if err != nil {
		http.Error(w, "Error returning admin users.", 500)
		return
	}

	// Allow browsers and intermediaries to cache this response for up to a day (86400 seconds)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	io.WriteString(w, string(output))
}

// add a proof entry or update a preexisting proof entry
func (env *Env) saveProof(w http.ResponseWriter, req *http.Request) {
	var user userWithEmail
	user = req.Context().Value("tok").(userWithEmail)

	var submittedProof datastore.Proof

	// read the JSON-encoded value from the HTTP request and store it in submittedProof
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

	// Replace submitted email (if any) with the email from the token
	submittedProof.UserSubmitted = user.GetEmail()

	if err := env.ds.Store(submittedProof); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"success": "true"}`)
}

func (env *Env) getProofs(w http.ResponseWriter, req *http.Request) {
	user := req.Context().Value("tok").(userWithEmail)
	log.Println("backend.go: getProofs(): 'tok': " + user.GetEmail())

	if req.Method != "POST" || req.Body == nil {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	// Accepted JSON fields must be defined here
	type getProofRequest struct {
		Selection string `json:"selection"`
	}

	var requestData getProofRequest

	decoder := json.NewDecoder(req.Body)

	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, "Unable to decode request body.", 400)
		return
	}

	log.Printf("%+v", requestData)

	if len(requestData.Selection) == 0 {
		http.Error(w, "Selection required", 400)
		return
	}

	log.Printf("USER: %q", user)

	var err error
	var proofs []datastore.Proof
	var sectionProofs []datastore.SectionProofs

	switch requestData.Selection {
	case "user":
		log.Println("user selection")
		err, proofs = env.ds.GetUserProofs(user)

	case "repo":
		log.Println("repo selection")
		// get repo problems associated with the sections that the user is in
		err, sectionProofs = env.ds.GetRepoProofs(user)

	case "completedrepo":
		log.Println("completedrepo selection")
		err, proofs = env.ds.GetUserCompletedProofs(user)

	case "downloadrepo":
		log.Println("downloadrepo selection")
		if !admin_users[user.GetEmail()] {
			http.Error(w, "Insufficient privileges", 403)
			return
		}
		err, proofs = env.ds.GetAllAttemptedRepoProofs()

	default:
		http.Error(w, "invalid selection", 400)
		return
	}

	if err != nil {
		log.Println("error: backend.go: getProofs(): " + err.Error())
		http.Error(w, "Query error", 500)
		return
	}

	log.Printf("%+v", proofs)
	var userProofsJSON []byte
	if proofs != nil {
		userProofsJSON, err = json.Marshal(proofs)
	} else {
		userProofsJSON, err = json.Marshal(sectionProofs)
	}

	if err != nil {
		http.Error(w, "json marshal error", 500)
		log.Print(err)
		return
	}

	io.WriteString(w, string(userProofsJSON))

	log.Printf("%q", user)
	log.Printf("%+v", req.URL.Query())
}

// get proof entries for current user where entryType is argument
func (env *Env) getUserArguments(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" || req.Body == nil {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	user := req.Context().Value("tok").(userWithEmail)

	arguments, err := env.ds.GetUserArguments(user)
	if err != nil {
		log.Println("error: backend.go: getUserArguments(): " + err.Error())
		http.Error(w, "Query error", 500)
		return
	}
	log.Printf("%+v", arguments)

	userProofsJSON, err := json.Marshal(arguments)
	if err != nil {
		http.Error(w, "json marshal error", 500)
		log.Print(err)
		return
	}

	io.WriteString(w, string(userProofsJSON))
}

// return section entries given a user's email
func (env *Env) getSections(w http.ResponseWriter, req *http.Request) {
	log.Println("inside backend.go: getSections")
	userEmail := req.URL.Query().Get("user")

	if req.Method != "GET" || userEmail == "" {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	log.Printf("for section: %q\n", userEmail)

	sections, err := env.ds.GetSections(userEmail)
	if err != nil {
		http.Error(w, "db access error", 500)
		log.Println(err)
		return
	}
	log.Printf("all sections: %+v\n\n", sections)

	sectionsJSON, err := json.Marshal(sections)
	if err != nil {
		http.Error(w, "json marshal error", 500)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(sectionsJSON))

	/*
		if req.Method != "GET" || req.Body == nil {
			http.Error(w, "Request not accepted.", 400)
			return
		}

		// Accepted JSON fields must be defined here
		type getSectionsRequest struct {
			User string `json:"user"`
		}

		var requestData getSectionsRequest

		decoder := json.NewDecoder(req.Body)

		if err := decoder.Decode(&requestData); err != nil {
			http.Error(w, "Unable to decode request body.", 400)
			return
		}

		log.Printf("%+v", requestData)

		if requestData.User == "" {
			http.Error(w, "user required", 400)
			return
		}

		var err error

		var sections []datastore.Section
		sections, err = env.ds.GetSections(requestData.User)
		if err != nil {
			http.Error(w, "db access error", 500)
			log.Println(err)
			return
		}

		log.Printf("All Sections: %+v\n\n", sections)

		sectionsJSON, err := json.Marshal(sections)
		if err != nil {
			http.Error(w, "json marshal error", 500)
			log.Println(err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, string(sectionsJSON))
		// fmt.Printf("All Sections JSON: %+v\n", sectionsJSON)\
	*/
}

// return student and ta roster entries given a sectionName
func (env *Env) getRoster(w http.ResponseWriter, req *http.Request) {
	log.Println("inside backend.go: getRoster")

	sectionName := req.URL.Query().Get("sectionName")

	if req.Method != "GET" || sectionName == "" {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	log.Printf("for section: %q\n", sectionName)

	roster, err := env.ds.GetRoster(sectionName)
	if err != nil {
		http.Error(w, "db access error", 500)
		log.Println(err)
		return
	}
	log.Printf("full roster: %+v\n\n", roster)

	rosterJSON, err := json.Marshal(roster)
	if err != nil {
		http.Error(w, "json marshal error", 500)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(rosterJSON))
	// log.Printf("marshalled rosterJSON: %+v\n", rosterJSON)
	/*

		if req.Method != "GET" || req.Body == nil {
			http.Error(w, "Request not accepted.", 400)
			return
		}

		// Accepted JSON fields must be defined here
		type getRosterRequest struct {
			SectionName string `json:"sectionName"`
		}

		var requestData getRosterRequest

		decoder := json.NewDecoder(req.Body)

		if err := decoder.Decode(&requestData); err != nil {
			http.Error(w, "Unable to decode request body.", 400)
			return
		}

		log.Printf("%+v", requestData)

		if requestData.SectionName == "" {
			http.Error(w, "section required", 400)
			return
		}

		var err error

		var roster []datastore.Roster
		roster, err = env.ds.GetRoster(requestData.SectionName)
		if err != nil {
			http.Error(w, "db access error", 500)
			log.Println(err)
			return
		}

		log.Printf("Roster: %+v\n\n", roster)

		rosterJSON, err := json.Marshal(roster)
		if err != nil {
			http.Error(w, "json marshal error", 500)
			log.Println(err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, string(rosterJSON))
	*/
}

// return proof entries completed by students associated with a given section
func (env *Env) getCompletedProofsBySection(w http.ResponseWriter, req *http.Request) {
	log.Println("inside backend.go: getCompletedProofsBySection")
	sectionName := req.URL.Query().Get("sectionName")

	if req.Method != "GET" || sectionName == "" {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	log.Printf("for section: %q\n", sectionName)

	proofs, err := env.ds.GetCompletedProofsBySection(sectionName)
	if err != nil {
		http.Error(w, "db access error", 500)
		log.Println(err)
		return
	}

	proofsJSON, err := json.Marshal(proofs)
	if err != nil {
		http.Error(w, "json marshal error", 500)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(proofsJSON))

	/*
		if req.Method != "GET" || req.Body == nil {
			http.Error(w, "Request not accepted.", 400)
			return
		}

		// Accepted JSON fields must be defined here
		type getgetCompletedProofsBySectionRequest struct {
			SectionName string `json:"sectionName"`
		}

		var requestData getgetCompletedProofsBySectionRequest

		decoder := json.NewDecoder(req.Body)

		if err := decoder.Decode(&requestData); err != nil {
			http.Error(w, "Unable to decode request body.", 400)
			return
		}

		log.Printf("%+v", requestData)

		if requestData.SectionName == "" {
			http.Error(w, "section required", 400)
			return
		}

		var err error

		var proofs []datastore.Proof
		proofs, err = env.ds.GetCompletedProofsBySection(requestData.SectionName)
		if err != nil {
			http.Error(w, "db access error", 500)
			log.Println(err)
			return
		}

		log.Printf("CompletedProofsBySection: %+v", proofs)

		proofsJSON, err := json.Marshal(proofs)
		if err != nil {
			http.Error(w, "json marshal error", 500)
			log.Println(err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, string(proofsJSON))
	*/

}

// get all assignments associated with a specific section
func (env *Env) getAssignmentsBySection(w http.ResponseWriter, req *http.Request) {
	/*
		if req.Method != "GET" || req.Body == nil {
			http.Error(w, "Request not accepted.", 400)
			return
		}

		// Accepted JSON fields must be defined here
		type getAssignmentsRequest struct {
			SectionName string `json:"sectionName"`
		}

		var requestData getAssignmentsRequest

		decoder := json.NewDecoder(req.Body)

		if err := decoder.Decode(&requestData); err != nil {
			http.Error(w, "Unable to decode request body.", 400)
			return
		}

		log.Printf("%+v", requestData)

		if requestData.SectionName == "" {
			http.Error(w, "section name required", 400)
			return
		}

		var err error

		var assignmentDetails []datastore.Assignment
		assignmentDetails, err = env.ds.GetAssignmentsBySection(requestData.SectionName)
		if err != nil {
			http.Error(w, "db access error", 500)
			log.Println(err)
			return
		}
	*/
	log.Println("inside backend.go: getAssignmentBySection")
	sectionName := req.URL.Query().Get("sectionName")

	if req.Method != "GET" || sectionName == "" {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	assignmentDetails, err := env.ds.GetAssignmentsBySection(sectionName)
	if err != nil {
		http.Error(w, "db access error", 500)
		log.Println(err)
		return
	}

	type assignmentWithProofs struct {
		Name       string            `json:"name"`
		ProofList  []datastore.Proof `json:"proofList"`
		Visibility string            `json:"visibility"`
	}

	var assignments []assignmentWithProofs
	for _, v := range assignmentDetails {
		var singleAssign assignmentWithProofs
		singleAssign.Name = v.Name
		singleAssign.Visibility = v.Visibility
		singleAssign.ProofList, err = env.ds.GetAssignmentProofs(v)
		if err != nil {
			http.Error(w, "db access error", 500)
			log.Println(err)
			return
		}
		assignments = append(assignments, singleAssign)
	}
	log.Printf("assignments array: %+v", assignments)

	assignmentsJSON, err := json.Marshal(assignments)
	if err != nil {
		http.Error(w, "json marshal error", 500)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(assignmentsJSON))
}

func (env *Env) getCompletedProofsByAssignment(w http.ResponseWriter, req *http.Request) {
	/*
		if req.Method != "GET" || req.Body == nil {
			http.Error(w, "Request not accepted.", 400)
			return
		}

		// Accepted JSON fields must be defined here
		type getCompletedProofsRequest struct {
			SectionName string `json:"sectionName"`
			AssignmentName string `json:"assignmentName"`
		}

		var requestData getCompletedProofsRequest

		decoder := json.NewDecoder(req.Body)

		if err := decoder.Decode(&requestData); err != nil {
			http.Error(w, "Unable to decode request body.", 400)
			return
		}

		log.Printf("%+v", requestData)

		if requestData.SectionName == "" {
			http.Error(w, "section name required", 400)
			return
		}
		if requestData.AssignmentName == "" {
			http.Error(w, "assignment name required", 400)
			return
		}

		var err error

		var completedProofs []datastore.Proof
		completedProofs, err = env.ds.GetCompletedProofsByAssignment(requestData.SectionName, requestData.AssignmentName)
	*/
	log.Println("inside backend.go: getCompletedProofByAssignment")
	sectionName := req.URL.Query().Get("sectionName")
	assignmentName := req.URL.Query().Get("assignmentName")

	if req.Method != "GET" || sectionName == "" || assignmentName == "" {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	var completedProofs []datastore.Proof
	completedProofs, err := env.ds.GetCompletedProofsByAssignment(sectionName, assignmentName)

	completedProofsJSON, err := json.Marshal(completedProofs)
	if err != nil {
		http.Error(w, "json marshal error", 500)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(completedProofsJSON))
}

// add a section based on current admin user and given sectionName
func (env *Env) addSection(w http.ResponseWriter, req *http.Request) {
	log.Println("inside backend.go: addSection")
	user := req.Context().Value("tok").(userWithEmail)

	if req.Method != "POST" || req.Body == nil {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	// Accepted JSON fields must be defined here
	type reqBody struct {
		SectionName string `json:"sectionName"`
	}

	var requestData reqBody

	decoder := json.NewDecoder(req.Body)

	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, "Unable to decode request body.", 400)
		return
	}
	log.Printf("user: [%q], requestData: %+v", user.GetEmail(), requestData)

	err := env.ds.InsertSection(datastore.Section{InstructorEmail: user.GetEmail(), Name: requestData.SectionName})
	if err != nil {
		http.Error(w, "db section insertion error: "+err.Error(), 500)
		log.Println(err)
		return
	}
	err = env.ds.InsertRoster(datastore.Roster{SectionName: requestData.SectionName, UserEmail: user.GetEmail(), Role: "instructor"})
	if err != nil {
		http.Error(w, "db roster insertion error: "+err.Error(), 500)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"success": "true"}`)
}

// add a roster entry: requires sectionName, studentEmails, and taEmails
func (env *Env) addRoster(w http.ResponseWriter, req *http.Request) {
	log.Println("inside backend.go: addRoster")

	if req.Method != "POST" || req.Body == nil {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	// Accepted JSON fields must be defined here
	type reqBody struct {
		SectionName   string   `json:"sectionName"`
		StudentEmails []string `json:"studentEmails"`
		TaEmails      []string `json:"taEmails"`
	}

	var requestData reqBody

	decoder := json.NewDecoder(req.Body)

	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, "Unable to decode request body.", 400)
		return
	}

	type insertionErr struct {
		Email string `json:"email"`
		Msg   string `json:"msg"`
	}
	var insertionErrList []insertionErr
	for _, email := range requestData.StudentEmails {
		// working here! adjust for InsertRoster - need to grab sectionName, email, and role
		err := env.ds.InsertUser(datastore.User{Email: email, FirstName: "", LastName: "", Admin: 0})
		err = env.ds.InsertRoster(datastore.Roster{SectionName: requestData.SectionName, UserEmail: email, Role: "student"})
		if err != nil {
			insertionErrList = append(insertionErrList, insertionErr{Email: email, Msg: err.Error()})
		}
	}
	for _, email := range requestData.TaEmails {
		// working here! adjust for InsertRoster - need to grab sectionName, email, and role
		err := env.ds.InsertUser(datastore.User{Email: email, FirstName: "", LastName: "", Admin: 1})
		err = env.ds.InsertRoster(datastore.Roster{SectionName: requestData.SectionName, UserEmail: email, Role: "ta"})
		if err != nil {
			insertionErrList = append(insertionErrList, insertionErr{Email: email, Msg: err.Error()})
		}
	}

	insertionErrJSON, err := json.Marshal(insertionErrList)
	if err != nil {
		http.Error(w, "json marshal error", 500)
		log.Print(err)
		return
	}

	if len(insertionErrList) != 0 {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, fmt.Sprintf(`{"success": "false", "errors": %+v}`, string(insertionErrJSON)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"success": "true"}`)
}

func (env *Env) addAssignment(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" || req.Body == nil {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	type reqBody struct {
		SectionName string `json:"sectionName"`
		Name        string `json:"name"`
		ProofIds    []int  `json:"proofIds"`
		Visibility  string `json:"visibility"`
	}

	var requestData reqBody
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, "Unable to decode request body.", 400)
		return
	}

	var assignment datastore.Assignment
	assignment.SectionName = requestData.SectionName
	assignment.Name = requestData.Name
	assignment.ProofIds = fmt.Sprint(requestData.ProofIds)
	assignment.Visibility = requestData.Visibility

	err := env.ds.InsertAssignment(assignment)
	if err != nil {
		http.Error(w, "db assignment insertion error: "+err.Error(), 500)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"success": "true"}`)
}

func (env *Env) updateAssignment(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" || req.Body == nil {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	type reqBody struct {
		SectionName       string `json:"sectionName"`
		CurrentName       string `json:"currentName"`
		UpdatedName       string `json:"updatedName"`
		UpdatedProofIds   []int  `json:"updatedProofIds"`
		UpdatedVisibility string `json:"updatedVisibility`
	}

	var requestData reqBody
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, "Unable to decode request body.", 400)
		return
	}

	var UpdatedAssignment datastore.Assignment
	UpdatedAssignment.SectionName = requestData.SectionName
	UpdatedAssignment.Name = requestData.UpdatedName
	UpdatedAssignment.ProofIds = fmt.Sprint(requestData.UpdatedProofIds)
	UpdatedAssignment.Visibility = requestData.UpdatedVisibility

	err := env.ds.UpdateAssignment(requestData.CurrentName, UpdatedAssignment)
	if err != nil {
		http.Error(w, "db assignment update error: "+err.Error(), 500)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"success": "true"}`)
}

// remove 1 roster entry, based on user email and section name
func (env *Env) removeFromRoster(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" || req.Body == nil {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	type reqBody struct {
		SectionName string `json:"sectionName"`
		UserEmail   string `json:"userEmail"`
	}

	var requestData reqBody

	decoder := json.NewDecoder(req.Body)

	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, "Unable to decode request body.", 400)
		return
	}

	err := env.ds.RemoveFromRoster(requestData.SectionName, requestData.UserEmail)
	if err != nil {
		http.Error(w, "db roster deletion error", 500)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"success": "true"}`)
}

// remove section entry and associated roster entries, given a section name
func (env *Env) removeSection(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" || req.Body == nil {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	type reqBody struct {
		SectionName string `json:"sectionName"`
	}

	var requestData reqBody

	decoder := json.NewDecoder(req.Body)

	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, "Unable to decode request body.", 400)
		return
	}

	err := env.ds.RemoveSection(requestData.SectionName)
	if err != nil {
		http.Error(w, "db section deletion error", 500)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"success": "true"}`)
}

// remove 1 assignment entry, based on section name and name of assignment
func (env *Env) removeAssignment(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" || req.Body == nil {
		http.Error(w, "Request not accepted.", 400)
		return
	}

	type reqBody struct {
		SectionName string `json:"sectionName"`
		Name        string `json:"name"`
	}

	var requestData reqBody

	decoder := json.NewDecoder(req.Body)

	if err := decoder.Decode(&requestData); err != nil {
		http.Error(w, "Unable to decode request body.", 400)
		return
	}

	err := env.ds.RemoveAssignment(requestData.SectionName, requestData.Name)
	if err != nil {
		http.Error(w, "db assignment deletion error", 500)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"success": "true"}`)
}

// This will delete all roster, assignment, section, and non-argument proof rows, but does not reset the auto_increment id
func (env *Env) clearDatabase() {
	if err := env.ds.EmptyRosterTable(); err != nil {
		log.Fatal(err)
	}
	if err := env.ds.EmptyAssignmentTable(); err != nil {
		log.Fatal(err)
	}
	if err := env.ds.EmptySectionTable(); err != nil {
		log.Fatal(err)
	}
	if err := env.ds.EmptyProofTable(); err != nil {
		log.Fatal(err)
	}
}

func (env *Env) populateTestProofRow() {
	err := env.ds.Store(datastore.Proof{
		EntryType:      "argument",
		UserSubmitted:  "elarson@csumb.edu",
		ProofName:      "Repository - Code Test",
		ProofType:      "prop",
		Premise:        []string{"P", "P → Q", "Q → R", "R → S"},
		Logic:          []string{"[{\"wffstr\":\"P\",\"jstr\":\"Pr\"},{\"wffstr\":\"P → Q\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q → R\",\"jstr\":\"Pr\"},{\"wffstr\":\"R → S\",\"jstr\":\"Pr\"}"},
		Rules:          []string{},
		EverCompleted:  "false",
		ProofCompleted: "false",
		Conclusion:     "S",
		TimeSubmitted:  "2019-04-29T01:45:44.452+0000",
		RepoProblem:    "true",
	})

	if err != nil {
		log.Println("error from Store(elarson argument)")
		log.Fatal(err)
	}

	err = env.ds.Store(datastore.Proof{
		EntryType:      "proof",
		UserSubmitted:  "jduboiTEST@csumb.edu",
		ProofName:      "Repository - Code Test",
		ProofType:      "prop",
		Premise:        []string{"P", "P → Q", "Q → R", "R → S"},
		Logic:          []string{"[{\"wffstr\":\"P\",\"jstr\":\"Pr\"},{\"wffstr\":\"P → Q\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q → R\",\"jstr\":\"Pr\"},{\"wffstr\":\"R → S\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q\",\"jstr\":\"1, 2 →E\"},{\"wffstr\":\"R\",\"jstr\":\"3, 5 →E\"}]"},
		Rules:          []string{},
		EverCompleted:  "false",
		ProofCompleted: "false",
		Conclusion:     "S",
		TimeSubmitted:  "2022-03-14T03:10:44.452+0000",
		RepoProblem:    "true",
	})

	if err != nil {
		log.Println("error: populateTestProofRow() from Store(student-proof1f)")
		log.Fatal(err)
	}

	err = env.ds.Store(datastore.Proof{
		EntryType:      "proof",
		UserSubmitted:  "jduboisTEST@csumb.edu",
		ProofName:      "Repository - Code Test",
		ProofType:      "prop",
		Premise:        []string{"P", "P → Q", "Q → R", "R → S"},
		Logic:          []string{"[{\"wffstr\":\"P\",\"jstr\":\"Pr\"},{\"wffstr\":\"P → Q\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q → R\",\"jstr\":\"Pr\"},{\"wffstr\":\"R → S\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q\",\"jstr\":\"1, 2 →E\"},{\"wffstr\":\"R\",\"jstr\":\"3, 5 →E\"},{\"wffstr\":\"S\",\"jstr\":\"4, 6 →E\"}]"},
		Rules:          []string{},
		EverCompleted:  "true",
		ProofCompleted: "true",
		Conclusion:     "S",
		TimeSubmitted:  "2022-03-14T03:41:44.452+0000",
		RepoProblem:    "true",
	})

	err = env.ds.Store(datastore.Proof{
		EntryType:      "argument",
		UserSubmitted:  "bkondo@csumb.edu",
		ProofName:      "Repository - Code Test 2",
		ProofType:      "prop",
		Premise:        []string{"P", "P → Q", "Q → R"},
		Logic:          []string{"[{\"wffstr\":\"P\",\"jstr\":\"Pr\"},{\"wffstr\":\"P → Q\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q → R\",\"jstr\":\"Pr\"}"},
		Rules:          []string{},
		EverCompleted:  "false",
		ProofCompleted: "false",
		Conclusion:     "Q",
		TimeSubmitted:  "2022-04-03T00:44:49Z",
		RepoProblem:    "true",
	})

	err = env.ds.Store(datastore.Proof{
		EntryType:      "proof",
		UserSubmitted:  "jduboisTEST@csumb.edu",
		ProofName:      "Repository - Code Test 2",
		ProofType:      "prop",
		Premise:        []string{"P", "P → Q", "Q → R"},
		Logic:          []string{"[{\"wffstr\":\"P\",\"jstr\":\"Pr\"},{\"wffstr\":\"P → Q\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q → R\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q\",\"jstr\":\"1, 2 →E\"},{\"wffstr\":\"R\",\"jstr\":\"3, 5 →E\"}]"},
		Rules:          []string{},
		EverCompleted:  "false",
		ProofCompleted: "false",
		Conclusion:     "Q",
		TimeSubmitted:  "2022-04-03T00:44:49Z",
		RepoProblem:    "true",
	})

	err = env.ds.Store(datastore.Proof{
		EntryType:      "proof",
		UserSubmitted:  "jasbaker@csumb.edu",
		ProofName:      "Repository - Code Test 2",
		ProofType:      "prop",
		Premise:        []string{"P", "P → Q", "Q → R"},
		Logic:          []string{"[{\"wffstr\":\"P\",\"jstr\":\"Pr\"},{\"wffstr\":\"P → Q\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q → R\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q\",\"jstr\":\"1, 2 →E\"}]"},
		Rules:          []string{},
		EverCompleted:  "true",
		ProofCompleted: "true",
		Conclusion:     "Q",
		TimeSubmitted:  "2022-04-03T00:44:49Z",
		RepoProblem:    "true",
	})
}

func main() {
	log.Println("Server initializing")

	ds, err := datastore.InitDB(database_uri)
	if err != nil {
		log.Fatal(err)
	}
	defer ds.Close()

	// Add the admin users to the database for use in queries
	ds.MaintainAdmins(admin_users)

	Env := &Env{ds} // Put the instance into a struct to share between threads
	// Env.ds.PopulateTestUsersSectionsRosters()
	// Env.populateTestProofRow()

	doClearDatabase := flag.Bool("cleardb", false, "Remove all proofs from the database")
	doPopulateDatabase := flag.Bool("populate", false, "Add sample data to the public repository.")
	portPtr := flag.String("port", "8080", "Port to listen on")

	flag.Parse() // Check for command-line arguments
	if *doClearDatabase {
		Env.clearDatabase()
	}
	if *doPopulateDatabase {
		Env.populateTestProofRow()
	}

	// Initialize token auth/cache
	tokenauth.SetAuthorizedDomains(authorized_domains)
	tokenauth.SetAuthorizedClientIds(authorized_client_ids)

	// method saveproof : POST : JSON <- id_token, proof
	http.Handle("/saveproof", tokenauth.WithValidToken(http.HandlerFunc(Env.saveProof)))

	// method user : POST : JSON -> [proof, proof, ...]
	http.Handle("/proofs", tokenauth.WithValidToken(http.HandlerFunc(Env.getProofs)))

	// spr2022 GETs : use JSON req.body for arguments
	http.Handle("/sections", tokenauth.WithValidToken(http.HandlerFunc(Env.getSections)))
	http.Handle("/roster", tokenauth.WithValidToken(http.HandlerFunc(Env.getRoster)))
	http.Handle("/completed-proofs-by-section", tokenauth.WithValidToken(http.HandlerFunc(Env.getCompletedProofsBySection)))
	http.Handle("/completed-proofs-by-assignment", tokenauth.WithValidToken(http.HandlerFunc(Env.getCompletedProofsByAssignment)))
	http.Handle("/assignments-by-section", tokenauth.WithValidToken(http.HandlerFunc(Env.getAssignmentsBySection)))
	http.Handle("/arguments-by-user", tokenauth.WithValidToken(http.HandlerFunc(Env.getUserArguments)))

	// spr2022 POST (delete has also been treated as POST) : use JSON req.body for arguments
	http.Handle("/add-section", tokenauth.WithValidToken(http.HandlerFunc(Env.addSection)))
	http.Handle("/add-roster", tokenauth.WithValidToken(http.HandlerFunc(Env.addRoster)))
	http.Handle("/add-assignment", tokenauth.WithValidToken(http.HandlerFunc(Env.addAssignment)))
	http.Handle("/update-assignment", tokenauth.WithValidToken(http.HandlerFunc(Env.updateAssignment)))
	http.Handle("/remove-from-roster", tokenauth.WithValidToken(http.HandlerFunc(Env.removeFromRoster)))
	http.Handle("/remove-section", tokenauth.WithValidToken(http.HandlerFunc(Env.removeSection)))
	http.Handle("/remove-assignment", tokenauth.WithValidToken(http.HandlerFunc(Env.removeAssignment)))

	// Get admin users -- this is a public endpoint, no token required
	// Can be changed to require token, but would reduce cacheability
	http.Handle("/admins", http.HandlerFunc(Env.getAdmins))

	log.Println("Server started")
	log.Fatal(http.ListenAndServe("127.0.0.1:"+(*portPtr), nil))
}
