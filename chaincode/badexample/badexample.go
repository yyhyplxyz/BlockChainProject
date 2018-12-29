package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"fmt"
	"math/rand"
	"time"
	"strconv"
	"bytes"
)

type BadExampleCC struct {}

// Init is called during Instantiate transaction after the chaincode container
// has been established for the first time, allowing the chaincode to
// initialize its internal data
func (c *BadExampleCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke is called to update or query the ledger in a proposal transaction.
// Updated state variables are not committed to the ledger until the
// transaction is committed.
func (c *BadExampleCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(bytes.NewBufferString(strconv.Itoa(int(rand.Int63n(time.Now().Unix())))).Bytes())
}

func main() {
	err := shim.Start(new(BadExampleCC))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
