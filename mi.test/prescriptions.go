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
//	User_and_eCert - Struct for storing the JSON of a user and their ecert
//==============================================================================================================================

type User_and_eCert struct {
	Identity string `json:"identity"`
	eCert string `json:"ecert"`
}

//==============================================================================================================================
//	Init Function - Called when the user deploys the chaincode
//==============================================================================================================================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	//Args
	//				0
	//			peer_address

	for i:=0; i < len(args); i=i+2 {
		t.add_ecert(stub, args[i], args[i+1])
	}

	return nil, nil
}

//==============================================================================================================================
//	 General Functions
//==============================================================================================================================
//	 get_ecert - Takes the name passed and calls out to the REST API for HyperLedger to retrieve the ecert
//				 for that user. Returns the ecert as retrived including html encoding.
//==============================================================================================================================
func (t *SimpleChaincode) get_ecert(stub shim.ChaincodeStubInterface, name string) ([]byte, error) {

	ecert, err := stub.GetState(name)

	if err != nil { return nil, errors.New("Couldn't retrieve ecert for user " + name) }

	return ecert, nil
}

//==============================================================================================================================
//	 add_ecert - Adds a new ecert and user pair to the table of ecerts
//==============================================================================================================================

func (t *SimpleChaincode) add_ecert(stub shim.ChaincodeStubInterface, name string, ecert string) ([]byte, error) {


	err := stub.PutState(name, []byte(ecert))

	if err == nil {
		return nil, errors.New("Error storing eCert for user " + name + " identity: " + ecert)
	}

	return nil, nil

}



//==============================================================================================================================
//	 get_caller - Retrieves the username of the user who invoked the chaincode.
//				  Returns the username as a string.
//==============================================================================================================================

func (t *SimpleChaincode) get_username(stub shim.ChaincodeStubInterface) (string, error) {

    username, err := stub.ReadCertAttribute("username");
	if err != nil { return "", errors.New("Couldn't get attribute 'username'. Error: " + err.Error()) }
	return string(username), nil
}

//==============================================================================================================================
//	 check_affiliation - Takes an ecert as a string, decodes it to remove html encoding then parses it and checks the
// 				  		certificates common name. The affiliation is stored as part of the common name.
//==============================================================================================================================

func (t *SimpleChaincode) check_affiliation(stub shim.ChaincodeStubInterface) (string, error) {
    affiliation, err := stub.ReadCertAttribute("role");
	if err != nil { return "", errors.New("Couldn't get attribute 'role'. Error: " + err.Error()) }
	return string(affiliation), nil

}

//==============================================================================================================================
//	 get_caller_data - Calls the get_ecert and check_role functions and returns the ecert and role for the
//					 name passed.
//==============================================================================================================================

func (t *SimpleChaincode) get_caller_data(stub shim.ChaincodeStubInterface) (string, string, error){

	user, err := t.get_username(stub)

    // if err != nil { return "", "", err }

	// ecert, err := t.get_ecert(stub, user);

    // if err != nil { return "", "", err }

	affiliation, err := t.check_affiliation(stub);

    if err != nil { return "", "", err }

	return user, affiliation, nil
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

	caller, caller_affiliation, err := t.get_caller_data(stub)

	if err != nil { return nil, errors.New("Error retrieving caller information")}


	if function == "create_request" {
        return t.create_request(stub, caller, caller_affiliation)
	} else {
			// If the function is not a create then there must be a request so we need to retrieve it.
			argPos := 1
			bytes, err := stub.GetState(args[argPos])

			if err != nil {	fmt.Printf("INVOKE: reqeust can't be found : %s", err); return nil, errors.New("INVOKE: reqeust can't be found "+string(argPos))	}

			err = json.Unmarshal(bytes, &r);
			if err != nil {	fmt.Printf("INVOKE: request corrupted : %s", err); return nil, errors.New("INVOKE: reqeust corrupted "+string(bytes))	}

			return t.review_request(stub, r, caller, "insurance_company")
		}

	return nil, errors.New("Function of the name "+ function +" doesn't exist.")

}

func (t *SimpleChaincode) create_request(stub shim.ChaincodeStubInterface, caller string, caller_affiliation string) ([]byte, error) {
	var r Request

	id         		 := "\"ID\":\"UNDEFINED\", "						// Variables to define the JSON
	din            := "\"DIN\":0, "
	state           := "\"State\":\"UNDEFINED\", "

	request_json := "{"+id+din+state+"}" 	// Concatenates the variables to create the total JSON object

	matched, err := regexp.Match("^[A-z][A-z][0-9]{7}", []byte("aa1234567")) // hack, always match, just to declare the 'err'
	if 	matched == false    {
					return nil, errors.New("Invalid v5cID provided")
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

	return nil, nil
}
//=================================================================================================================================
//	 Transfer Functions
//=================================================================================================================================
//	 authority_to_manufacturer
//=================================================================================================================================
func (t *SimpleChaincode) review_request(stub shim.ChaincodeStubInterface, r Request, caller string, caller_affiliation string) ([]byte, error) {

	if  r.State		== STATE_CREATED	&&
			caller_affiliation		== INSURANCE_COMPANY {		// If the roles and users are ok

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

	caller, caller_affiliation, err := t.get_caller_data(stub)
	if err != nil { fmt.Printf("QUERY: Error retrieving caller details", err); return nil, errors.New("QUERY: Error retrieving caller details: "+err.Error()) }

    logger.Debug("function: ", function)
    logger.Debug("caller: ", caller)
    logger.Debug("affiliation: ", caller_affiliation)

	if function == "get_request_details" {
		if len(args) != 1 { fmt.Printf("Incorrect number of arguments passed"); return nil, errors.New("QUERY: Incorrect number of arguments passed") }
		return t.get_request_details(stub, args[0], caller, caller_affiliation)
	}
	return nil, errors.New("Received unknown function invocation " + function)

}

//=================================================================================================================================
//	 Read Functions
//=================================================================================================================================
//	 get_vehicle_details
//=================================================================================================================================
func (t *SimpleChaincode) get_request_details(stub shim.ChaincodeStubInterface, id string, caller string, caller_affiliation string) ([]byte, error) {

	bytes, err := stub.GetState(id);

	if err != nil { return nil, errors.New("get_vehicle_details: Invalid request object") }

	return bytes, nil

}


//=================================================================================================================================
//	 Main - main - Starts up the chaincode
//=================================================================================================================================
func main() {

	err := shim.Start(new(SimpleChaincode))

	if err != nil { fmt.Printf("Error starting Chaincode: %s", err) }
}
