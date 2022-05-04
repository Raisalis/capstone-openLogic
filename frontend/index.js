'use strict';

const repositoryData = {
   'userProofs': [],
   'repoProofs': [],
   'completedUserProofs': [],
   'studentNames':[],
   'proofAverages':[]
}

let adminUsers = [];

/**
 * This function is called by the Google Sign-in Button
 * @param {*} googleUser 
 */
function onSignIn(googleUser) {
   console.log("onSignIn", googleUser);

   // This response will be cached after the first page load
   $.getJSON('/backend/admins', (admins) => {
      try {
	 console.log(admins);
	 adminUsers = admins['Admins'];
      } catch(e) {
	 console.error('Unable to load admin users', e);
      }

      new User(googleUser)
	 .initializeDisplay()
	 .loadProofs();
   });
}

/**
 * Class for functionality specific to user sign-in/authentication
 */
class User {
   // Constructor is called from User.onSignIn - not intended for direct use.
   constructor(googleUser) {
      this.profile = googleUser.getBasicProfile();
      this.domain = googleUser.getHostedDomain();
      this.email = this.profile.getEmail();
      this.name = this.profile.getName();

      if (adminUsers.indexOf(this.email) > -1) {
	 console.log('Logged in as an administrator.');
	 this.showAdminFunctionality();
      }

      this.attachSignInChangeListener();
      return this;
   }

   initializeDisplay() {
      $('#user-email').text(this.email);
      $('#load-container').show();
      $('#nameyourproof').show();

      return this;
   }

   showAdminFunctionality() {
      $('#adminLink').show();

      return this;
   }

   loadProofs() {
      loadUserProofs();
      loadRepoProofs();
      loadUserCompletedProofs();

      return this;
   }

   attachSignInChangeListener() {
      gapi.auth2.getAuthInstance().isSignedIn.listen(this.signInChangeListener);

      return this;
   }

   signInChangeListener(loggedIn) {
      console.log('Sign in status changed', loggedIn);
      window.location.reload();
   }

   static isSignedIn() {
      return gapi.auth2.getAuthInstance().isSignedIn.get();
   }

   static isAdministrator() {
      return adminUsers.indexOf(gapi.auth2.getAuthInstance().currentUser.get().getBasicProfile().getEmail()) > -1;
   }

   // Check if the current time (in unix timestamp) is after the token's expiration
   static isTokenExpired() {
      return + new Date() > gapi.auth2.getAuthInstance().currentUser.get().getAuthResponse().expires_at;
   }

   // Retrieve the last cached token
   static getIdToken() {
      return gapi.auth2.getAuthInstance().currentUser.get().getAuthResponse().id_token;
   }

   // Get a newly issued token (returns a promise)
   static refreshToken() {
      return gapi.auth2.getAuthInstance().currentUser.get().reloadAuthResponse();
   }
}

async function ViewClasses(){


   backendGET('Roster', {selection: 'UserEmail'}).then(   
   (data) => {
      // console.log("loadStudentNames", data);
      // repositoryData.studentNames = data;       

      
      prepareSelect('#studentNameSelect', data);
      
      }, console.log
   );

   backendGET('Proof', {selection: 'ProofName'}).then(   
   (data) => {
      // console.log("loadProofs", data);
      // repositoryData.proofAverages = data;      

      prepareSelect('#ProofNameSelect', data);
      }, console.log
   );
}


//this inserts the class and students in that class, the students should be seperated by a comma
async function insertClass(){
   var name=document.getElementById("className").value;
   
   var students= $("#involveStudents").val().split(",");
   console.log(students);
   if(name==""||students==""){
      alert("One or more input boxes are empty");
   }else{
      await backendPOST('add-section',{sectionName:name}).then(
         (data) =>{
            console.log('add section',data);
         }
      );
      backendPOST('add-roster', {sectionName: name, studentEmails: students}).then(
         (data) => {
            console.log("add roster", data);
                   
            
         }
      );
      
   
      
         //this will come up if the submission was successful
         alert("Your submission was accepted");
   }
   //will work on the rest after figuring out how to get function call properly

   
}

//this will drop the class of the admin by name, everything, including the students will be dropped
async function dropClass(){
   var x=document.getElementById("dropSectionName").value;
   console.log(x);
   if(confirm("Are you sure you want to drop the whole class?")==true){
      //waiting for tables to be ready to do rest
      //temporary idea
      backendPOST('remove-section',{sectionName: x});
      

      alert("Class deleted");
   }
}

//this will drop one student at a time, for one reason or another,
async function dropStudent(){
   var deadToClass=document.getElementById("sectionToRemoveStudent").value;
   var deadStudent= document.getElementById("dropKid").value;
   if(deadStudent==""||deadToClass==""){
      alert("One or more inputs are empty.");
   }else{
      console.log(deadToClass);
      console.log(deadStudent);
      //waiting for tables to be ready to do rest
      if(confirm("Are you sure you want to drop this student?")==true){
         
         backendPOST("remove-from-roster", {sectionName:deadToClass, userEmail:deadStudent});
         alert("Student removed");
      }
   }
   
   
}

//the following are just menu popups based on button clicks on the admin buttons
function showDrop(){
   var dropper= document.getElementById("hiddenDrop");
   if(dropper.style.display=== "block"){
      dropper.style.display= "none";
   }else{
      dropper.style.display="block";
   }
}

function showDropClass(){
   var dropper= document.getElementById("howToDropClass");
   if(dropper.style.display=== "block"){
      dropper.style.display= "none";
   }else{
      dropper.style.display="block";
   }
}


function showAssignments(){
   var assignment= document.getElementById("assignmentPage");
   if(assignment.style.display=="block"){
      assignment.style.display= "none";
   }else{
      assignment.style.display="block";
   }

}


function showAddProofAssignment(){
   var assignment=document.getElementById("addProofAssignmentDiv");
   if(assignment.style.display=="block"){
      assignment.style.display= "none";
   }else{
      assignment.style.display="block";
   }
}

function showRemoveProofAssignment(){
   var assignment=document.getElementById("removeProofAssignmentDiv");
   if(assignment.style.display=="block"){
      assignment.style.display= "none";
   }else{
      assignment.style.display="block";
   }
}

function showAddAssignmentClass(){
   var assignment=document.getElementById("addAssignmentClassDiv");
   if(assignment.style.display=="block"){
      assignment.style.display= "none";
   }else{
      assignment.style.display="block";
   }
}

function showRemoveAssignmentClass(){
   var assignment=document.getElementById("removeAssignmentClassDiv");
   if(assignment.style.display=="block"){
      assignment.style.display= "none";
   }else{
      assignment.style.display="block";
   }
}

async function addAssignmentToClass(){
   var add=document.getElementById("classAssignmentIn").value;
   var classIn=document.getElementById("classIn").value;
   if(add==""||classIn==""){
      alert("At least one input is empty, please insert the class name and assignment name in their respective input boxes");
   }else{

      backendPOST('add-assignment', {sectionName: classIn, assignmentName: add});
      alert("Assignment has been added to the class");
   }
}

async function removeAssignmentFromClass(){
   var sub=document.getElementById("classAssignmentOut").value;
   var classOut=document.getElementById("classOut").value;
   if(sub==""||classOut==""){
      alert("At least one input is empty, please insert the class name and assignment name in their respective input boxes");
   }else{
      backendPOST('remove-assignment',{sectionName:classOut,assignmentName:sub});
      alert("Assignment removed from class");
   }
}


async function insertAssignment(){
   var assignmentN=document.getElementById("assignmentName").value;
   var classN=document.getElementById("assignedClass").value;
   if(assignmentN==""||classN==""){
      alert("The input is empty, please enter assignment name and class name.");
   }else{
      backendPOST('add-assignment', {name:assignmentN, sectionName:classN});
      alert("Assignment Made");
   }
}

async function removeAssignment(){
   var assignmentO=document.getElementById("assignmentName").value;
   var classN=document.getElementById("assignedClass").value;
   if(assignmentO==""||classN==""){
      alert("The input is empty, please enter assignment name and class name.");
   }else{
      backendPOST('remove-assignment',{name:assignmentO, sectionName:classN});
      alert("Assignment Removed");
   }
}

async function addProofAssignment(){
   
   var assignment=document.getElementById("proofAssignmentIn").value;
   var proof=document.getElementById("proofIn").value;

   if(assignment==""||proof==""){
      alert("One or more inputs are empty, please select the proof and assignments in their respective options");
   }else{
      backendPOST("update-assignment",{currentName:assignment,updatedProofIds:proof});

      alert("Proof is added to assignment");
   }

}
async function removeProofAssignment(){
   var assignment=document.getElementById("proofAssignmentOut").value;
   var proof=document.getElementById("proofOut").value;
   if(assignment==""||proof==""){
      alert("One or more inputs are empty, please select the proof and assignments in their respective options");
   }else{
      backendPOST("update-assignment",{currentName:assignment,updatedProofIds:proof});
      alert("Proof removed from assignment");
   }
}

async function fillClassNames() {
   var userEmail = document.getElementById("userEmail").text;
   await backendGET("sections", {user:userEmail}).then(
      (data)=>{
         let elem = document.querySelector("#assignmentClassNames");

         // Remove all child nodes from the select element
         $(elem).empty();

         // Create placeholder option
         elem.appendChild(
            new Option('Select...', null, true, true)
         );

         // Set placeholder to disabled so it does not show as selectable
         elem.querySelector('option').setAttribute('disabled', 'disabled');

         // Add option elements for the options
         (data) && data.forEach( section => {
            let option = new Option(section.Name, section.Name);
            elem.appendChild(option);
         });
      }, console.log
   );
}


async function fillAddProofAssignment(){
   var classRoom=document.getElementById("proofClassIn").value;

   await backendGET("assignments-by-section",{sectionName:classRoom}).then(
      (data)=>{
         prepareSelect("#proofAssignmentIn", data);

      }, console.log
   );
   backendGET('Proof', {selection: 'ProofName'}).then(   
      (data) => {
              
         prepareSelect('#ProofIn', data);
         }, console.log
      );
}

async function fillDropProofAssignment(){
   var classRoom=document.getElementById("proofClassIn").value;

   await backendGET("assignments-by-section",{sectionName:classRoom}).then(
      (data)=>{
         prepareSelect("#proofAssignmentOut", data);

      }, console.log
   );
   backendGET('Proof', {selection: 'ProofName'}).then(   
      (data) => {
              
   
         prepareSelect('#ProofOut', data);
         }, console.log
      );
}
async function fillProof(){
   //will need to wait for the get functions to work
   // backendGET("proof-values", {proofName, proof}).then(
   //    (data)=>{

   //    }
   // );
}

async function fillAssignmentCheckboxes() {
   var sectionName = document.getElementById('publishClass');
   var checkboxHolder = document.getElementById('checkboxHolder')
   await backendGET("assignments-by-section", {sectionName:sectionName}).then(
      (data)=>{
         var i = 0;
         for(let assignment of data) {
            const newDiv = document.createElement("div");
            const newCheck = document.createElement("INPUT");
            newCheck.setAttribute("type", "checkbox");
            newCheck.setAttribute("name", "assignment");
            newCheck.setAttribute("value", assignment.name);
            newCheck.setAttribute("id", "option"+i)
            if(assignment.visibility == "true") {
               newCheck.setAttribute("checked", true);
            } else {
               newCheck.setAttribute("checked", false);
            }
            newDiv.append(newCheck);
            const newLabel = document.createElement("LABEL");
            newLabel.textContent = assignment.name;
            newLabel.setAttribute("for", "option"+i);
            newDiv.append(newLabel);
            checkboxHolder.appendChild(newDiv);
            i++;
         }
      }
   );
}
// function showClassAssignment(){
//    backendGET()
// }

// Verifies signed in and valid token, then calls authenticatedBackendPOST
// Returns a promise which resolves to the response body or undefined
function backendPOST(path_str, data_obj) {
   if (!User.isSignedIn()) {
      console.warn('Cannot send POST request to backend from unknown user.');
      if (sessionStorage.getItem('loginPromptShown') == null) {
	 alert('You are not signed in.\nTo save your work, please sign in and then try again, or refresh the page.');
	 sessionStorage.setItem('loginPromptShown', "true");
      }
      
      return Promise.reject( 'Unauthenticated user' );
   }

   if (User.isTokenExpired()) {
      console.warn('Token expired; attempting to refresh token.');
      return User.refreshToken().then(
	 (googleUser) => authenticatedBackendPOST(path_str, data_obj, googleUser.id_token));
   } else {
      return authenticatedBackendPOST(path_str, data_obj, User.getIdToken());
   }
}

//we added this part in order to get the appropriate GET call for some other functions
function backendGET(path_str, data_obj) {
   if (!User.isSignedIn()) {
      console.warn('Cannot send GET request to backend from unknown user.');
      if (sessionStorage.getItem('loginPromptShown') == null) {
	 alert('You are not signed in.\nTo save your work, please sign in and then try again, or refresh the page.');
	 sessionStorage.setItem('loginPromptShown', "true");
      }
      
      return Promise.reject( 'Unauthenticated user' );
   }

   if (User.isTokenExpired()) {
      console.warn('Token expired; attempting to refresh token.');
      return User.refreshToken().then(
	 (googleUser) => authenticatedBackendGET(path_str, data_obj, googleUser.id_token));
   } else {
      return authenticatedBackendGET(path_str, data_obj, User.getIdToken());
   }
}

// Send a POST request to the backend, with auth token included
function authenticatedBackendPOST(path_str, data_obj, id_token) {
   return $.ajax({
      url: '/backend/' + path_str,
      method: 'POST',
      data: JSON.stringify(data_obj),
      dataType: 'json',
      contentType: 'application/json; charset=utf-8',
      headers: {
	 'X-Auth-Token': id_token
      }
   }).then(
      (data, textStatus, jqXHR) => {
	 return data;
      },
      (jqXHR, textStatus, errorThrown) => {
	 console.error(textStatus, errorThrown);
      }
   )
}


function authenticatedBackendGET(path_str, data_obj, id_token) {
   return $.ajax({
      url: '/backend/' + path_str,
      method: 'GET',
      data: JSON.stringify(data_obj),
      dataType: 'json',
      contentType: 'application/json; charset=utf-8',
      headers: {
	 'X-Auth-Token': id_token
      }
   }).then(
      (data, textStatus, jqXHR) => {
	 return data;
      },
      (jqXHR, textStatus, errorThrown) => {
	 console.error(textStatus, errorThrown);
      }
   )
}

// For administrators only - backend requires valid admin token
function getCSV() {
   backendPOST('proofs', { selection: 'downloadrepo' }).then(
      (data) => {
	 console.log("downloadRepo", data);

	 if (!Array.isArray(data) || data.length < 1) {
            console.error('No proofs received.');
            return;
	 }

	 let csv_header = Object.keys(data[0]).join(',') + '\n';

	 let csv = data.reduce( (rows, proof) => {
            return rows + Object.values(proof).reduce( (accum, elem) => {
               if (Array.isArray(elem)) {
		  return accum + ',"' + elem.join('|') + '"';
               }
               return accum + ',"' + elem + '"';
            }) + '\n';
	 }, csv_header);

	 let downloadLink = document.createElement('a');
	 downloadLink.download = "Student_Problems.csv";
	 downloadLink.href = 'data:text/csv;charset=utf-8,' + encodeURI(csv);
	 downloadLink.target = '_blank';
	 downloadLink.click();
      }, console.log);
}

// Hides and displays the admin options
function showProofs(){
   var proofs=document.getElementById("proofValues");
   var student= document.getElementById("studentPage");

   if(proofs.style.display=== "block"){
      proofs.style.display= "none";
   }else{
      proofs.style.display="block";
   }
   if(student.style.display=== "block"){
      proofs.style.display= "none";
   }
   
}

function showStudents(){
   var proofs=document.getElementById("proofValues");
   var student= document.getElementById("studentPage");

   if(student.style.display=== "block"){
      student.style.display= "none";
   }else{
      student.style.display="block";
   }
   if(proofs.style.display=== "block"){
      student.style.display= "none";
   }
}





const prepareSelect = (selector, options) => {
   let elem = document.querySelector(selector);

   // Remove all child nodes from the select element
   $(elem).empty();

   // Create placeholder option
   elem.appendChild(
      new Option('Select...', null, true, true)
   );

   // Set placeholder to disabled so it does not show as selectable
   elem.querySelector('option').setAttribute('disabled', 'disabled');

   // Add option elements for the options
   (options) && options.forEach( proof => {
      let option = new Option(proof.ProofName, proof.Id);
      elem.appendChild(option);
   });
}

// load user's incomplete proofs
function loadUserProofs() {
   backendPOST('proofs', { selection: 'user' }).then(
      (data) => {
	 console.log("loadSelect", data);
	 repositoryData.userProofs = data;
	 prepareSelect('#userProofSelect', data);
	 $('#userProofSelect').data('repositoryDataKey', 'userProofs')
      }, console.log
   );
}

// load repository problems
function loadRepoProofs() {
   backendPOST('proofs', { selection: 'repo' }).then(
      (data) => {
	 console.log("loadRepoProofs", data);
	 repositoryData.repoProofs = data;

	 //prepareSelect('#repoProofSelect', data);
	 let elem = document.querySelector('#repoProofSelect');
	 $(elem).empty();

	 elem.appendChild(
            new Option('Select...', null, true, true)
	 );

	 let currentSectionName;
	 (data) && data.forEach( section => {
            if (currentSectionName !== section.SectionName) {
               currentSectionName = section.SectionName;
               console.log(section.SectionName);
               elem.appendChild(
		            new Option(section.SectionName, null, false, false)
               );
            }
            section.ProofList.forEach( proof => {
               console.log(proof);
               elem.appendChild(
                  new Option(proof.ProofName, proof.Id)
               );
            });
	 });

	 // Make section headers not selectable
	 $('#repoProofSelect option[value=null]').attr('disabled', true);

	 $('#repoProofSelect').data('repositoryDataKey', 'repoProofs');
      }, console.log
   );
}

// load user's completed proofs
function loadUserCompletedProofs() {
   backendPOST('proofs', { selection: 'completedrepo' }).then(
      (data) => {
	 console.log("loadUserCompletedProofs", data);
	 repositoryData.completedUserProofs = data;
	 prepareSelect('#userCompletedProofSelect', data);
	 $('#userCompletedProofSelect').data('repositoryDataKey', 'completedUserProofs')
      }, console.log
   );
}

$(document).ready(function() {

   // store proof when check button is clicked
   $('.proofContainer').on('checkProofEvent', (event) => {
      console.log(event, event.detail, event.detail.proofdata);

      let proofData = event.detail.proofdata;

      let Premises = [].concat(proofData.filter( elem => elem.jstr == "Pr" ).map( elem => elem.wffstr ));

      // The Logic and Rules arrays used to contain lines of the proof, but
      // this only worked for proofs with no subproofs.
      // Now Logic is always a array containing a single string, and Rules is
      // always an empty array.
      let Logic = [JSON.stringify(proofData)],
          Rules = [];
      
      let entryType = "";
      if (adminUsers.indexOf(User.email) && Logic.length == 0) {
         entryType = "argument";
      } else {
         entryType = "proof";
      }

      let proofName = $('.proofNameSpan').text() || "n/a";
      let repoProblem = $('#repoProblem').val() || "false";
      let proofType = predicateSettings ? "fol" : "prop";

      let everCompleted = event.detail.everCompleted;
      let proofCompleted = event.detail.proofCompleted;
      let conclusion = event.detail.wantedConc;

      let postData = new Proof(entryType, proofName, proofType, Premises, Logic, Rules,
			       everCompleted, proofCompleted, conclusion, repoProblem);

      console.log('saving proof', postData);
      backendPOST('saveproof', postData).then(
	 (data) => {
	    console.log('proof saved', data);
	    
	    if (postData.proofCompleted == "true") {
               loadUserCompletedProofs();
	    } else {
               loadUserProofs();
	    }
		
            loadRepoProofs();
	 }, console.log)
   });

   // admin users - publish problems to public repo
   $('.proofContainer').on('click', '#togglePublicButton', (event) => {
      let proofName = $('.proofNameSpan').text();
      if (!proofName || proofName == "") {
	 proofName = prompt("Please enter a name for your proof:");
      }
      if (!proofName) {
	 console.error('No proof name entered');
	 return;
      }

      if (!proofName.startsWith('Repository - ')) {
	 proofName = 'Repository - ' + proofName;
      }
      $('.proofNameSpan').text(proofName);

      let publicStatus = $('#repoProblem').val() || 'false';
      if (publicStatus === 'false') {
	 $('#repoProblem').val('true');
	 $('#togglePublicButton').fadeOut().text('Make Private').fadeIn();
      } else {
	 $('#repoProblem').val('false');
	 $('#togglePublicButton').fadeOut().text('Make Public').fadeIn();
      }

      $('#checkButton').click();
   });

   // populate form when any repository proof selected
   $('.proofSelect').change( (event) => {
      // get the name of the selected item and the selected repository
      let selectedDataId = event.target.value;
      let selectedDataSetName = $(event.target).data('repositoryDataKey');

      // get the proof from the repository (== means '3' is equal to 3)
      let selectedDataSet = repositoryData[selectedDataSetName];
      let selectedProof = selectedDataSet.filter( proof => proof.Id == selectedDataId );
      if (!selectedProof || selectedProof.length < 1) {
	 console.error("Selected proof ID not found.");
	 return;
      }
      selectedProof = selectedProof[0];
      console.log('selected proof', selectedProof);

      // set repoProblem if proof originally loaded from the repository select
      if (selectedDataSetName == 'repoProofs' || selectedProof.repoProblem == "true") {
	 $('#repoProblem').val('true');
      } else {
	 $('#repoProblem').val('false');
      }

      // attach the proof body to the proofContainer
      if (Array.isArray(selectedProof.Logic) && Array.isArray(selectedProof.Rules)) {
	 $('.proofContainer').data({
            'Logic': selectedProof.Logic,
            'Rules': selectedProof.Rules
	 });
      }

      // set proofName, probpremises, and probconc; then click on #createProb
      // (add a small delay to show the user what's being done)
      let delayTime = 200;
      $.when(
	 $('#folradio').prop('checked', true),
	 // Checking this radio button will uncheck the other radio button
	 $('#tflradio').prop('checked', (selectedProof.ProofType == 'prop')),
	 $('#proofName').delay(delayTime).val(selectedProof.ProofName),
	 $('#probpremises').delay(delayTime).val(selectedProof.Premise.join(',')),
	 $('#probconc').delay(delayTime).val(selectedProof.Conclusion)
      ).then(
	 function () {
            $('#createProb').click();
	 }
      );
   });

   // create a problem based on premise and conclusion
   // get the proof name, premises, and conclusion from the document
   $("#createProb").click( function() {
      // predicateSettings is a global var defined in syntax_upstream.js
      predicateSettings = (document.getElementById("folradio").checked);
      let premisesString = document.getElementById("probpremises").value;
      let conclusionString = document.getElementById("probconc").value;
      let proofName = document.getElementById('proofName').value;
      createProb(proofName, premisesString, conclusionString);
   });

   $('.newProof').click( event => {
      resetProofUI();

      // reset 'repoProblem'
      $('#repoProblem').val('false');

      $('.createProof').slideDown();
      $('.proofContainer').slideUp();
   });

   $('#proofName').popup({ on: 'hover' });
   $('#repoProofSelect').popup({ on: 'hover' });
   $('#userCompletedProofSelect').popup({ on: 'hover' });

   // Admin modal
   $('#adminLink').click( (event) => {
      $('.ui.modal').modal('show');
   });

   $('.downloadCSV').click( () => getCSV() );
   // End admin modal
});

function resetProofUI() {
   $('#proofName').val('');			// clear name
   $('#tflradio').prop('checked', true);	// set to Propositional
   $('#probpremises').val('');			// clear premises
   $('#probconc').val('');			// clear conclusion
   $('.proofNameSpan').text('');		// clear proof name
   $('#theproof').empty();			// remove all HTML from 'theproof' element
   $('.proofContainer').removeData();		// clear proof body

   // reset all select boxes to "Select..." (the first option element)
   $('#load-container select option:nth-child(1)').prop('selected', true);
}

// predicateSettings = (document.getElementById("folradio").checked);
// var pstr = document.getElementById("probpremises").value;
// var conc = fixWffInputStr(document.getElementById("probconc").value);
function createProb(proofName, premisesString, conclusionString) {

   // verify the premises are well-formed
   let pstr = premisesString.replace(/^[,;\s]*/,'');
   pstr = pstr.replace(/[,;\s]*$/,'');
   let prems = pstr.split(/[,;\s]*[,;][,;\s]*/);

   // verify the conclusion is well-formed
   let conc = fixWffInputStr(conclusionString);
   var cw = parseIt(conc);
   if (!(cw.isWellFormed)) {
      alert('The conclusion ' + fixWffInputStr(conc) + ', is not well formed.');
      return false;
   }
   if ((predicateSettings) && (!(cw.allFreeVars.length == 0))) {
      alert('The conclusion is not closed.');
      return false;
   }

   // set the body of the proof
   // If the proof body is attached to the proofContainerData as array Logic[],
   // get the proof body from that.  Otherwise initialize the proof body from
   // the premises.
   // Note: for legacy reasons Logic always contains a single element -- the
   // JSON encoding of the proof data.
   let proofdata = [];
   let proofContainerData = $('.proofContainer').data();
   if (proofContainerData.hasOwnProperty('Logic')) {
      if (Array.isArray(proofContainerData.Logic) && proofContainerData.Logic.length > 0) {
	 proofdata = JSON.parse(proofContainerData.Logic[0])
      } else {
	 console.warn('Error/unexpected: Logic is not a non-empty array', proofContainerData);
      }
   } else {
      for (let a=0; a<prems.length; a++) {
	 if (prems[a] != '') {
	    let w = parseIt(fixWffInputStr(prems[a]));
	    if (!(w.isWellFormed)) {
               alert('Premise ' + (a+1) + ', ' + fixWffInputStr(prems[a]) + ', is not well formed.');
               return false;
            }
	    if ((predicateSettings) && (!(w.allFreeVars.length == 0))) {
               alert('Premise ' + (a+1) + ' is not closed.');
               return false;
	    }
	    proofdata.push({
               "wffstr": wffToString(w, false),
               "jstr": "Pr"
	    });
	 }
      }
   }

   $('.createProof').slideUp();
   resetProofUI();
   $('.proofContainer').show();
   $('.proofNameSpan').text(proofName);

   // set the argument (premises/conclusion)  string
   var probstr = '';
   for (var k=0; k < prems.length; k++) {
      probstr += prettyStr(prems[k]);
      if ((k+1) < prems.length) {
	 probstr += ', ';
      }
   }
   document.getElementById("proofdetails").innerHTML = "Construct a proof for the argument: " + probstr + " ∴ " +  wffToString(cw, true);

   var tp = document.getElementById("theproof");
   makeProof(tp, proofdata, wffToString(cw, false));
   return true;
}
