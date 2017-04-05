package main

import (
	"errors"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"encoding/json"
)

var logger = shim.NewLogger("MIChaincode")

//==============================================================================================================================
//	 Participant types - Each participant type is mapped to an integer which we use to compare to the value stored in a
//						 user's eCert
// 	not really used at this point
//==============================================================================================================================
const	DOCTOR						=	"doctor"
const	INSURANCE_COMPANY	=	"insurance_company"
const BENEFIT_MANAGER		=	"benefit_manager"
const PATIENT						=	"patient"
const PHARMACY					= "pharmacy"

//==============================================================================================================================
//	 Status types - Asset lifecycle is broken down into 5 statuses, this is part of the business logic to determine what can
//					be done to the medication prescription at points in it's lifecycle
//==============================================================================================================================

const	MEDICATION_STATE_PRESCRIBED = 0
const	MEDICATION_STATE_SUBMITTED = 1
const	MEDICATION_STATE_AUTHORIZED = 2
const	MEDICATION_STATE_DENIED = 3
const	MEDICATION_STATE_SUSPENDED = 4


//==============================================================================================================================
//	 Status types - Asset lifecycle is broken down into 4 statuses, this is part of the business logic to determine what can
//					be done to the medication authorization at points in it's lifecycle
//==============================================================================================================================
const	AUTHORIZATION_STATE_CREATED = 0
const	AUTHORIZATION_STATE_APPROVED = 1
const	AUTHORIZATION_STATE_REJECTED = 2
const	AUTHORIZATION_STATE_CANCELED = 3

//==============================================================================================================================
//	 Structure Definitions
//==============================================================================================================================
//	Chaincode - A blank struct for use with Shim (A HyperLedger included go file used for get/put state
//				and other HyperLedger functions)
//==============================================================================================================================
type  SimpleChaincode struct {
}

//==============================================================================================================================
//	authorization - Defines the structure for a medication authorization authorization object.
// JSON on right tells it what JSON fields to map to
//	that element when reading a JSON object into the struct e.g. JSON state -> Struct State.
//==============================================================================================================================
type Authorization struct {
	ID								string	`json:"ID"`
	PrescriptionID		string	`json:"prescriptionID"` // prescription ID to get the details (din/patient)
	PatientID					string	`json:"patientID"` // info field
	InsurerID					string	`json:"insurerID"` // insurance company ID to send authorization authorization to
	DoctorID					string	`json:"doctorID"` // doctor
	State 						int			`json:"state"`
}


//==============================================================================================================================
//	authorization - Defines the structure for a prescription from the doctor to benefit manager for approval.
//  JSON on right tells it what JSON fields to map to
//  that element when reading a JSON object into the struct e.g. JSON state -> Struct State.
//==============================================================================================================================
type Prescription struct {
	ID					string	`json:"ID"`
	PatientID		string	`json:"patientID"` // patient ID, need for the benefit manager to find the right insurance componeny to requst authorization
	DIN					string `json:"DIN"` // DIN to identify the medication
	State 		  int		`json:"state"`
}


//==============================================================================================================================
//	Patient - Defines the structure for a patient
//  JSON on right tells it what JSON fields to map to
//  that element when reading a JSON object into the struct e.g. JSON state -> Struct State.
//==============================================================================================================================
type Patient struct {
	ID					string	`json:"ID"`
	InsurerID		string	`json:"insurerID"` // insurer
	DoctorID		string	`json:"doctorID"` // doctor

}

type Patient_Registry struct {
	Patients 	[]string `json:"patients"`
}


type Prescription_Registry struct {
	Prescriptions 	[]string `json:"prescriptions"`
}


type Authorization_Registry struct {
	Authorizations 	[]string `json:"authorizations"`
}


//==============================================================================================================================
//	Init Function - Called when the user deploys the chaincode
//==============================================================================================================================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	//Args
	//				0
	//			peer_address

	// for i:=0; i < len(args); i=i+2 {
	//	t.add_ecert(stub, args[i], args[i+1])
	// }
	/* maybe for later
	var patientR Patient_Registry
	var prescriptionR Prescription_Registry
	var authorizationR Authorization_Registry
	*/

	patient	:=	Patient{"patient0","insurer0", "doctor0"}
	prescription	:=	Prescription{ID: "prescription0", PatientID: patient.ID, DIN:"DIN0", State: MEDICATION_STATE_SUBMITTED}
	authorization	:=	Authorization{ID: "authorization0", PrescriptionID:prescription.ID, DoctorID:patient.DoctorID,
		InsurerID:patient.InsurerID, PatientID:prescription.PatientID,State: AUTHORIZATION_STATE_CREATED}

	_, err := t.save_changes(stub, patient, patient.ID)
	if err != nil { fmt.Printf("INIT: Error saving patient: %s", err); return nil, errors.New("Error saving patient") }

	_, err = t.save_changes(stub, prescription, prescription.ID)
	if err != nil { fmt.Printf("INIT: Error saving prescription: %s", err); return nil, errors.New("Error saving prescription") }

	_, err = t.save_changes(stub, authorization, authorization.ID)
	if err != nil { fmt.Printf("INIT: Error saving authorization: %s", err); return nil, errors.New("Error saving authorization") }

	return nil, nil
}
/*
//==============================================================================================================================
// save_changes - Writes to the ledger the Vehicle struct passed in a JSON format. Uses the shim file's
//				  method 'PutState'.
//==============================================================================================================================
func (entry Base) save_changes(stub shim.ChaincodeStubInterface) (error) {

	bytes, err := json.Marshal(entry)

	if err != nil { fmt.Printf("SAVE_CHANGES: Error converting authorization record: %s", err); return errors.New("Error converting authorization record") }

	err = stub.PutState(entry.ID, bytes)

	if err != nil { fmt.Printf("SAVE_CHANGES: Error storing authorization record: %s", err); return errors.New("Error storing authorization record") }

	return nil
}
*/

//==============================================================================================================================
// save_changes - Writes to the ledger the Vehicle struct passed in a JSON format. Uses the shim file's
//				  method 'PutState'.
//==============================================================================================================================
func (t *SimpleChaincode) save_changes(stub shim.ChaincodeStubInterface, entry interface{}, id string) (bool, error) {

	bytes, err := json.Marshal(entry)

	if err != nil { fmt.Printf("SAVE_CHANGES: Error converting authorization record: %s", err); return false, errors.New("Error converting authorization record") }

	err = stub.PutState(id, bytes)

	if err != nil { fmt.Printf("SAVE_CHANGES: Error storing authorization record: %s", err); return false, errors.New("Error storing authorization record") }

	return true, nil
}


//==============================================================================================================================
//	 Router Functions
//==============================================================================================================================
//	Invoke - Called on chaincode invoke. Takes a function name passed and calls that function. Converts some
//		  initial arguments passed to other things for use in the called function e.g. name -> ecert
//==============================================================================================================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	if function == "create_authorization" {
        return t.create_authorization(stub, args)
	} else {
			// If the function is not a create then there must be a authorization so we need to retrieve it.
			var a Authorization
			bytes, err := stub.GetState(args[0])

			if err != nil {	fmt.Printf("INVOKE: reqeust can't be found : %s", err); return nil, errors.New("INVOKE: reqeust can't be found ")	}

			err = json.Unmarshal(bytes, &a);
			if err != nil {	fmt.Printf("INVOKE: authorization corrupted : %s", err); return nil, errors.New("INVOKE: reqeust corrupted "+string(bytes))	}

			return t.approve_authorization(stub, a)
		}

	return nil, errors.New("Function of the name "+ function +" doesn't exist.")

}

func (t *SimpleChaincode) create_authorization(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var a Authorization
	a.ID = args[0]

	record, err := stub.GetState(a.ID) 								// If not an error then a record exists so cant create a new authorization

  if record != nil { return nil, errors.New("authorization already exists") }

	// if 	caller_affiliation != AUTHORITY {							// Only the regulator can create a new v5c
	//	return nil, errors.New(fmt.Sprintf("Permission Denied. create_authorization. %v === %v", caller_affiliation, AUTHORITY))
	// }

	_, err  = t.save_changes(stub, a, a.ID)

	if err != nil { fmt.Printf("create_authorization: Error saving changes: %s", err); return nil, errors.New("Error saving changes") }

	return []byte (a.ID), nil
}
//=================================================================================================================================
//	 Transfer Functions
//=================================================================================================================================
//	 authority_to_manufacturer
//=================================================================================================================================
func (t *SimpleChaincode) approve_authorization(stub shim.ChaincodeStubInterface, a Authorization) ([]byte, error) {

	if  a.State		== AUTHORIZATION_STATE_CREATED {		// If the roles and users are ok

					a.State = AUTHORIZATION_STATE_APPROVED			// and mark it in the state of manufacture

	} else {									// Otherwise if there is an error
					fmt.Printf("approve_authorization: Review failed.");
          return nil, errors.New(fmt.Sprintf("Review failed. approve_authorization. "))

	}

	_, err := t.save_changes(stub, a, a.ID)						// Write new state

	if err != nil {	fmt.Printf("approve_authorization: Error saving changes: %s", err); return nil, errors.New("Error saving changes")	}

	bytes, err := json.Marshal(a)
	return bytes, nil									// We are Done

}


//=================================================================================================================================
//	Query - Called on chaincode query. Takes a function name passed and calls that function. Passes the
//  		initial arguments passed are passed on to the called function.
//=================================================================================================================================
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {


	if function == "get_details" {
		if len(args) != 1 { fmt.Printf("Incorrect number of arguments passed"); return nil, errors.New("QUERY: Incorrect number of arguments passed") }
		return t.get_details(stub, args[0])
	}
	return nil, errors.New("Received unknown function invocation " + function)

}

//=================================================================================================================================
//	 Read Functions
//=================================================================================================================================
//	 get_vehicle_details
//=================================================================================================================================
func (t *SimpleChaincode) get_details(stub shim.ChaincodeStubInterface, id string) ([]byte, error) {

	bytes, err := stub.GetState(id);

	if err != nil { return nil, errors.New("get_details: Invalid object") }

	return bytes, nil

}


//=================================================================================================================================
//	 Main - main - Starts up the chaincode
//=================================================================================================================================
func main() {

	err := shim.Start(new(SimpleChaincode))

	if err != nil { fmt.Printf("Error starting Chaincode: %s", err) }
}
