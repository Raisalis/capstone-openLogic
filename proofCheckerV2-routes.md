# Guide for Proof Checker V2 routes
### to view markdown in vs code: 
- <https://code.visualstudio.com/docs/languages/markdown>
  - dynamic outline/preview view: Ctrl+Shift+V
- <https://www.markdownguide.org/cheat-sheet/>
---
## Overview of routes:
### Route composition: 
- /backend/*path_str*

### path_str values available:
- GET
  - [admins](#admins)
  - [sections](#sections)
  - [roster](#roster)
  - [completed-proofs-by-assignment](#completed-proofs-by-assignment)
  - [completed-proofs-by-section](#completed-proofs-by-section)
  - [assignments-by-section](#assignments-by-section)
  - [arguments-by-user](#arguments-by-user)
- POST (delete and update are treated as post)
  - [add-section](#add-section)
  - [add-roster](#add-roster)
  - [add-assignment](#add-assignment)
  - [update-assignment](#update-assignment)
  - [remove-assignment](#remove-assignment)
  - [remove-from-roster](#remove-from-roster)
  - [remove-section](#remove-section)
  - [saveproof](#saveproof)
  - [proofs](#proofs)


### Note:
- all routes, except *admins*, require admin status in the database and an X-Auth-Token in the request header
- all routes are either GET or POST
  - all POST *request parameters* are given in the request body
  - all GET *request parameters* are given in the query string
    - the ajax GET request builder used by frontend will convert all request parameters into query string parameters!
      - that's why all the legacy proofs methods used a POST method.
- only csumb.edu users are accepted into the database

---

## path_str endpoints:
### **admins**:
- GET a list of current admins as assigned in backend/backend.go
- requires no request parameters
- response: an array of strings containing instructor emails
  ```
  {
      "Admins": [
          "jdoe@csumb.edu",
          "jadoe@csumb.edu",
      ]
  }
  ```
  [return](#pathstr-values-available)

---

### **sections**:
- GET a list of sections associated with given user
- requires a given *user*
  ```
  /backend/sections?user=email@csumb.edu
  ```
- response: a list of json object literals containing the *instructor* and *sectionName* associated with the given *user*
  - a student should only have one json object literal returned
  - an instructor may return many
  ```
  // if student
  [
    {
        "InstructorEmail": "instructor@csumb.edu",
        "Name": "Section 1"
    }
  ]
  // if instructor
  [
    {
        "InstructorEmail": "email@csumb.edu",
        "Name": "Section 1"
    },
    {
        "InstructorEmail": "email@csumb.edu",
        "Name": "Section 2"
    }
  ]
  ```
  [return](#pathstr-values-available)

---

### **roster**:
- GET a list of users and their roles given a section name
- requires: name of an already existing section
  ```
  /backend/roster?sectionName=Test Section
  ```
- response: an array of object literals containing the *sectionName*, *userEmail*, and *role* of each user associated with the given *sectionName*
  ```
  [
    {
        "SectionName": "Test Section",
        "UserEmail": "student01@csumb.edu",
        "Role": "student"
    },
    {
        "SectionName": "Test Section",
        "UserEmail": "student02@csumb.edu",
        "Role": "student"
    }
  ]
  ```
  [return](#pathstr-values-available)

---

### **completed-proofs-by-assignment**:
- GET a list of proofs submitted by students associated with a given section, the *proofName*s will match  
- requires: *sectionName* and *assignmentName*
  ```
  /backend/completed-proofs-by-assignment?sectionName=Larson Section&assignmentName=L Test assignment
  ```
- response: a list of Proof objects whose *proofName* matches a *proofName* associated with an id found in the assignment *proofIds* list
  ```
  [
    {
        "Id": "3",
        "EntryType": "proof",
        "UserSubmitted": "student01@csumb.edu",
        "ProofName": "Repository - Code Test",
        "ProofType": "prop",
        "Premise": [
            "P",
            "P → Q",
            "Q → R",
            "R → S"
        ],
        "Logic": [
            "[{\"wffstr\":\"P\",\"jstr\":\"Pr\"},{\"wffstr\":\"P → Q\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q → R\",\"jstr\":\"Pr\"},{\"wffstr\":\"R → S\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q\",\"jstr\":\"1, 2 →E\"},{\"wffstr\":\"R\",\"jstr\":\"3, 5 →E\"},{\"wffstr\":\"S\",\"jstr\":\"4, 6 →E\"}]"
        ],
        "Rules": [],
        "EverCompleted": "true",
        "ProofCompleted": "true",
        "Conclusion": "S",
        "RepoProblem": "true",
        "TimeSubmitted": "2022-05-05T00:05:56Z"
    }
  ]
  ```
  [return](#pathstr-values-available)

---

### **completed-proofs-by-section**:
- GET a list of completed student proofs according a given section
- requires: *sectionName* of an already existing section
  ```
  /backend/completed-proofs-by-section?sectionName=Test Section
  ```
- response: a list of Proof objects whose *userSubmitted* is associated with the *sectionName*
  ```
  [
    {
        "Id": "3",
        "EntryType": "proof",
        "UserSubmitted": "student01@csumb.edu",
        "ProofName": "Repository - Code Test",
        "ProofType": "prop",
        "Premise": [
            "P",
            "P → Q",
            "Q → R",
            "R → S"
        ],
        "Logic": [
            "[{\"wffstr\":\"P\",\"jstr\":\"Pr\"},{\"wffstr\":\"P → Q\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q → R\",\"jstr\":\"Pr\"},{\"wffstr\":\"R → S\",\"jstr\":\"Pr\"},{\"wffstr\":\"Q\",\"jstr\":\"1, 2 →E\"},{\"wffstr\":\"R\",\"jstr\":\"3, 5 →E\"},{\"wffstr\":\"S\",\"jstr\":\"4, 6 →E\"}]"
        ],
        "Rules": [],
        "EverCompleted": "true",
        "ProofCompleted": "true",
        "Conclusion": "S",
        "RepoProblem": "true",
        "TimeSubmitted": "2022-05-04T00:05:56Z"
    },
    {
        "Id": "12",
        "EntryType": "proof",
        "UserSubmitted": "student02@csumb.edu",
        "ProofName": "Repository - Code Test 3",
        "ProofType": "prop",
        "Premise": [
            "¬P",
            "P ∨ Q"
        ],
        "Logic": [
            "[{\"wffstr\":\"¬P\",\"jstr\":\"Pr\"},{\"wffstr\":\"P ∨ Q\",\"jstr\":\"Pr\"}]"
        ],
        "Rules": [],
        "EverCompleted": "true",
        "ProofCompleted": "true",
        "Conclusion": "S",
        "RepoProblem": "true",
        "TimeSubmitted": "2022-05-05T00:05:56Z"
    },
  ]
  ```
  [return](#pathstr-values-available)

---

### **assignments-by-section**:
- GET a list of assignments and their associated proofs(arguments)
- requires: *sectionName* of existing section
  ```
  /backend/assignments-by-section?sectionName=Test Section
  ```
- response: a list of json object literals containing the assignment name and a list of proofs(arguments) associated with that assignment
  ```
  [
    {
        "name": "L Test assignment",
        "proofList": [
            {
                "Id": "1",
                "EntryType": "argument",
                "UserSubmitted": "instructor@csumb.edu",
                "ProofName": "Repository - Code Test",
                "ProofType": "prop",
                "Premise": [
                    "P",
                    "P → Q",
                    "Q → R",
                    "R → S"
                ],
                "Logic": [],
                "Rules": [],
                "EverCompleted": "false",
                "ProofCompleted": "false",
                "Conclusion": "S",
                "RepoProblem": "true",
                "TimeSubmitted": "2022-05-05T00:05:56Z"
            },
            {
                "Id": "4",
                "EntryType": "argument",
                "UserSubmitted": "instructor@csumb.edu",
                "ProofName": "Repository - Code Test 2",
                "ProofType": "prop",
                "Premise": [
                    "P",
                    "P → Q",
                    "Q → R"
                ],
                "Logic": [],
                "Rules": [],
                "EverCompleted": "false",
                "ProofCompleted": "false",
                "Conclusion": "Q",
                "RepoProblem": "true",
                "TimeSubmitted": "2022-05-05T00:05:56Z"
            }
        ],
        "visibility": "true"
    },
    {
        "name": "L2 test assign",
        "proofList": [
            {
                "Id": "7",
                "EntryType": "argument",
                "UserSubmitted": "instructor@csumb.edu",
                "ProofName": "Repository - Code Test 3",
                "ProofType": "prop",
                "Premise": [
                    "¬P",
                    "P ∨ Q"
                ],
                "Logic": [
                    "[{\"wffstr\":\"¬P\",\"jstr\":\"Pr\"},{\"wffstr\":\"P ∨ Q\",\"jstr\":\"Pr\"}]"
                ],
                "Rules": [],
                "EverCompleted": "",
                "ProofCompleted": "false",
                "Conclusion": "Q",
                "RepoProblem": "true",
                "TimeSubmitted": "2022-05-06T00:06:43Z"
            }
        ],
        "visibility": "false"
    }
  ]
  ```
  [return](#pathstr-values-available)

---

### **arguments-by-user**:
- GET a list of arguments submitted by the current user
- requires no additional request parameters
  ```
  /backend/arguments-by-user
  ```
- response: a list of proofs(arguments) whose *userSubmitted* matches the email of the current user
  ```
  // if current user is instructor@csumb.edu
  [
      {
          "Id": "1",
          "EntryType": "argument",
          "UserSubmitted": "instructor@csumb.edu",
          "ProofName": "Repository - Code Test",
          "ProofType": "prop",
          "Premise": [
              "P",
              "P → Q",
              "Q → R",
              "R → S"
          ],
          "Logic": [],
          "Rules": [],
          "EverCompleted": "false",
          "ProofCompleted": "false",
          "Conclusion": "S",
          "RepoProblem": "true",
          "TimeSubmitted": "2022-05-05T00:05:56Z"
        },
        {
          "Id": "4",
          "EntryType": "argument",
          "UserSubmitted": "instructor@csumb.edu",
          "ProofName": "Repository - Code Test 2",
          "ProofType": "prop",
          "Premise": [
              "P",
              "P → Q",
              "Q → R"
          ],
          "Logic": [],
          "Rules": [],
          "EverCompleted": "false",
          "ProofCompleted": "false",
          "Conclusion": "Q",
          "RepoProblem": "true",
          "TimeSubmitted": "2022-05-05T00:05:56Z"
      },
      {
          "Id": "7",
          "EntryType": "argument",
          "UserSubmitted": "instructor@csumb.edu",
          "ProofName": "Repository - Code Test 3",
          "ProofType": "prop",
          "Premise": [
              "¬P",
              "P ∨ Q"
          ],
          "Logic": [
              "[{\"wffstr\":\"¬P\",\"jstr\":\"Pr\"},{\"wffstr\":\"P ∨ Q\",\"jstr\":\"Pr\"}]"
          ],
          "Rules": [],
          "EverCompleted": "",
          "ProofCompleted": "false",
          "Conclusion": "Q",
          "RepoProblem": "true",
          "TimeSubmitted": "2022-05-06T00:06:43Z"
      }
  ]
  ```

  [return](#pathstr-values-available)
  
---

### **add-section**:
- POST a new section for the current user
  - note: an instructor may not have duplicate section names
- requires a *sectionName* be given in the request body
  ```
  /backend/add-section
  
  {
    "sectionName": "Sp22 CST329-01"
  }
  ```
- response: a json object literal containing a success boolean value
  ```
  {
    "success": "true"
  }
  ```

  [return](#pathstr-values-available)
  
---

### **add-roster**:
- POST a new users to be associated with a section
- requires: name of an existing section, a list of student emails, a list of teacher's assistant emails
  - either list may be left empty
  ```
  /backend/add-roster

  {
    "sectionName": "Sp22 CST329-01",
    "studentEmails": ["student03@csumb.edu", "student04@csumb.edu"],
    "taEmails": ["student99@csumb.edu"]
  }
  ```
- response: a boolean success value and/or a list of errors containing the error message and email that the error occurred on
  - frontend may choose how to handle errors
  ```
  // if add succeeds
  {
    "success": "true"
  }

  // if add fails, the following example used a non-existent section
  {
    "success": "false",
    "errors": [
        {
            "email": "addRoster1@csumb.edu",
            "msg": "FOREIGN KEY constraint failed"
        },
        {
            "email": "addRoster2@csumb.edu",
            "msg": "FOREIGN KEY constraint failed"
        },
        {
            "email": "addRosterTA@csumb.edu",
            "msg": "FOREIGN KEY constraint failed"
        }
    ]
  }
  ```

  [return](#pathstr-values-available)
    
---

### **add-assignment**:
- POST a new assignment to be associated with a given section
  - note: the current user should only be able to make assignments for their own sections using their own proofs(arguments)
- requires: an existing *sectionName*, the *name* of the assignment, a list of *proofIds*, and a boolean *visibility* value
  ```
  /backend/add-assignment

  {
    "sectionName": "Test Section",
    "name": "L2 test assign",
    "proofIds": [1,4],
    "visibility": "false"
  }
  ```
- response: a boolean success value
  ```
  {
    "success": "true"
  }
  ```

  [return](#pathstr-values-available)
    
---

### **update-assignment**:
- POST updated values for an assignment
  - note: the current user should only be able to update assignments for their own sections using their own proofs(arguments)
- requires: an existing *sectionName*, an existing assignment name, the updated assignment name, the updated list of proof ids, and the updated visibility value
  - note:
    - the sectionName cannot be updated
    - the current(old) assignment must be given to find the current assignment to update
    - if no updates are required for a key, provide the current values
  ```
  /backend/update-assignment

  {
    "sectionName": "Test Section",
    "currentName": "L2 test assign",
    "updatedName": "L2 Test updated",
    "updatedProofIds": [1],
    "updatedVisibility": "false"
  }
  ```
- response: a boolean success value
  ```
  {
    "success": "true"
  }
  ```

  [return](#pathstr-values-available)
    
---

### **remove-assignment**:
- POST an assignment to be removed from a given section
  - note: the current user should only be able to remove assignments from their own sections
- requires: an existing *sectionName*, a *name* of the assignment to be removed that is associated with the given section
  ```
  /backend/remove-assignment

  {
    "sectionName": "Test Section",
    "name": "L2 Test updated"
  }
  ```
- response: a boolean success value
  ```
  {
    "success": "true"
  }
  ```

  [return](#pathstr-values-available)

---

### **remove-from-roster**:
- POST a given user to be removed from a given section
  - note: the current user should only be able to remove users from their own sections
- requires: an existing *sectionName*, an *userEmail* that is associated with the given section
  ```
  /backend/remove-from-roster

  {
    "sectionName": "Test Section",
    "userEmail": "student02@csumb.edu"
  }
  ```
- response: a boolean success value
  ```
  {
    "success": "true"
  }
  ```

  [return](#pathstr-values-available)
    
---

### **remove-section**:
- POST a section to be removed
  - note:
    - the current user should only be able to remove their own sections
    - removing a section will also remove roster data and assignment data associated with the section
- requires: an existing section name
  ```
  /backend/remove-section

  {
    "sectionName": "Test Section"
  }
  ```
- response: a boolean success value
  ```
  {
    "success": "true"
  }
  ```

  [return](#pathstr-values-available)
   
---

### **saveproof**:
- Legacy route: POST a proof to be added or updated
  - note:
    - 3 versions of a user's proof may be recorded
      - 1 proof entry for each of *proofCompleted*'s values: "true", "false", "error"
      - previously, saveproof would overwrite the single existing of the proof (only the last attempt would be recorded)
    - this remains the same as the legacy code, except for the addition of everCompleted
- response: a boolean success of true **or** an http 500 error 
  ```
  {
    "success": "true"
  }
  ```
  [return](#pathstr-values-available)
    
---

### **proofs**:
- Legacy route: POST a request to get a list of proofs based on a selection keyword
- requires: a selection keyword ("user", "repo", "completedRepo", "downloadRepo")
  ```
  {
    "selection": "keyword"
  }
  ```
- response: a list of proofs
  - "user":
    - returns all proofs whose *userSubmitted* matches the current user, *everCompleted* is "false", and proofCompleted does not equal "true"
  - "repo":
    - should return a list proofs associated with visible assignments that are associated with the current user's section(s)
  - "completedRepo":
    - return a list of proofs whose *userSubmitted* matches the current user and their *proofCompleted* value is "true"
  - "downloadrepo":
    - return a list of all proofs whose Premise matches the premise of an admin submitted proof, whose conclusion matches an admin submitted proof, and whose *entryType* is "proof"
    - ordered by userSubmitted, proofName, proofCompleted
    - ** please use completed-proofs-by-section or completed-proofs-by-assignment instead of the "downloadrepo" option **

  [return](#pathstr-values-available)