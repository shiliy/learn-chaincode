package main

import (
	"errors"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"encoding/json"
	"regexp"
)

var logger = shim.NewLogger("MIChaincode")

//==============================================================================================================================
//	 Participant types - Each participant type is mapped to an integer which we use to compare to the value stored in a
//						 user's eCert
//==============================================================================================================================
//CURRENT WORKAROUND USES ROLES CHANGE WHEN OWN USERS CAN BE CREATED SO THAT IT READ 1, 2, 3, 4, 5
const   DOCTOR      =  "doctor"
const   INSURANCE_COMPANY  =  "insurance_company"

//==============================================================================================================================
//	 Status types - Asset lifecycle is broken down into 5 statuses, this is part of the business logic to determine what can
//					be done to the vehicle at points in it's lifecycle
//==============================================================================================================================
const   STATE_CREATED		  			=  0
const   STATE_APPROVED  				=  1
const   STATE_REJECTED         	=  2

//==============================================================================================================================
//	 Structure Definitions
//==============================================================================================================================
//	Chaincode - A blank struct for use with Shim (A HyperLedger included go file used for get/put state
//				and other HyperLedger functions)
//==============================================================================================================================
type  SimpleChaincode struct {
}

//==============================================================================================================================
//	Request - Defines the structure for a medication authorization request object. JSON on right tells it what JSON fields to map to
//			  that element when reading a JSON object into the struct e.g. JSON price -> Struct Price.
//==============================================================================================================================
type Request struct {
	ID							 string	`json:"ID"`
	DIN              string `json:"DIN"`
	State 		     	 int		`json:"state"`
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
	var r Request
	r.ID = "ID000"
	r.DIN = "DIN000"
	r.State = STATE_CREATED
	_, err  = t.save_changes(stub, r)

	if err != nil { fmt.Printf("INIT: Error saving changes: %s", err); return nil, errors.New("Error saving changes") }

	return nil, nil
}

//==============================================================================================================================
// save_changes - Writes to the ledger the Vehicle struct passed in a JSON format. Uses the shim file's
//				  method 'PutState'.
//==============================================================================================================================
func (t *SimpleChaincode) save_changes(stub shim.ChaincodeStubInterface, r Request) (bool, error) {

	bytes, err := json.Marshal(r)

	if err != nil { fmt.Printf("SAVE_CHANGES: Error converting request record: %s", err); return false, errors.New("Error converting request record") }

	err = stub.PutState(r.ID, bytes)

	if err != nil { fmt.Printf("SAVE_CHANGES: Error storing request record: %s", err); return false, errors.New("Error storing request record") }

	return true, nil
}


//==============================================================================================================================
//	 Router Functions
//==============================================================================================================================
//	Invoke - Called on chaincode invoke. Takes a function name passed and calls that function. Converts some
//		  initial arguments passed to other things for use in the called function e.g. name -> ecert
//==============================================================================================================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	var r Request

	if function == "create_request" {
        return t.create_request(stub)
	} else {
			// If the function is not a create then there must be a request so we need to retrieve it.
			argPos := 1
			bytes, err := stub.GetState(args[argPos])

			if err != nil {	fmt.Printf("INVOKE: reqeust can't be found : %s", err); return nil, errors.New("INVOKE: reqeust can't be found "+string(argPos))	}

			err = json.Unmarshal(bytes, &r);
			if err != nil {	fmt.Printf("INVOKE: request corrupted : %s", err); return nil, errors.New("INVOKE: reqeust corrupted "+string(bytes))	}

			return t.review_request(stub, r)
		}

	return nil, errors.New("Function of the name "+ function +" doesn't exist.")

}

func (t *SimpleChaincode) create_request(stub shim.ChaincodeStubInterface) ([]byte, error) {
	var r Request

	id         		 := "\"ID\":\"UNDEFINED\", "						// Variables to define the JSON
	din            := "\"DIN\":0, "
	state           = STATE_CREATED

	request_json := "{"+id+din+state+"}" 	// Concatenates the variables to create the total JSON object

	matched, err := regexp.Match("^[A-z][A-z][0-9]{7}", []byte("aa1234567")) // hack, always match, just to declare the 'err'
	if 	matched == false    {
					return nil, errors.New("Invalid JSON provided")
	}
	err = json.Unmarshal([]byte(request_json), &r)							// Convert the JSON defined above into a vehicle object for go

	if err != nil { return nil, errors.New("Invalid JSON object") }

	record, err := stub.GetState(r.ID) 								// If not an error then a record exists so cant create a new request

  if record != nil { return nil, errors.New("request already exists") }

	// if 	caller_affiliation != AUTHORITY {							// Only the regulator can create a new v5c
	//	return nil, errors.New(fmt.Sprintf("Permission Denied. create_request. %v === %v", caller_affiliation, AUTHORITY))
	// }

	_, err  = t.save_changes(stub, r)

	if err != nil { fmt.Printf("CREATE_REQUEST: Error saving changes: %s", err); return nil, errors.New("Error saving changes") }

	return r.ID, nil
}
//=================================================================================================================================
//	 Transfer Functions
//=================================================================================================================================
//	 authority_to_manufacturer
//=================================================================================================================================
func (t *SimpleChaincode) review_request(stub shim.ChaincodeStubInterface, r Request) ([]byte, error) {

	if  r.State		== STATE_CREATED {		// If the roles and users are ok

					r.State = STATE_APPROVED			// and mark it in the state of manufacture

	} else {									// Otherwise if there is an error
					fmt.Printf("REVIEW_REQUEST: Review failed.");
          return nil, errors.New(fmt.Sprintf("Review failed. review_request. "))

	}

	_, err := t.save_changes(stub, r)						// Write new state

	if err != nil {	fmt.Printf("REVIEW_REQUEST: Error saving changes: %s", err); return nil, errors.New("Error saving changes")	}

	return nil, nil									// We are Done

}


//=================================================================================================================================
//	Query - Called on chaincode query. Takes a function name passed and calls that function. Passes the
//  		initial arguments passed are passed on to the called function.
//=================================================================================================================================
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {


	if function == "get_request_details" {
		if len(args) != 1 { fmt.Printf("Incorrect number of arguments passed"); return nil, errors.New("QUERY: Incorrect number of arguments passed") }
		return t.get_request_details(stub, args[0])
	}
	return nil, errors.New("Received unknown function invocation " + function)

}

//=================================================================================================================================
//	 Read Functions
//=================================================================================================================================
//	 get_vehicle_details
//=================================================================================================================================
func (t *SimpleChaincode) get_request_details(stub shim.ChaincodeStubInterface, id string) ([]byte, error) {

	bytes, err := stub.GetState(id);

	if err != nil { return nil, errors.New("get_request_details: Invalid request object") }

	return bytes, nil

}


//=================================================================================================================================
//	 Main - main - Starts up the chaincode
//=================================================================================================================================
func main() {

	err := shim.Start(new(SimpleChaincode))

	if err != nil { fmt.Printf("Error starting Chaincode: %s", err) }
}
