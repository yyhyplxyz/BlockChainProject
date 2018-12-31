package main

import (
	"fmt"
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type AssertsExchangeCC struct{}

const (
	originOwner = "originOwnerPlaceholder"
)

// 用户
type User struct {
	Name string `json:"name"` // messagepack || protobuf
	Id   string `json:"id"`
	phone string `json:"phone"`
	email string `json:"phone"`
	Assets []string `json:"assets"`
	Loan []string `json:"Loans"`
}

// 资产
type Asset struct {
	Name string `json:"name"`
	Id   string `json:"id"`
	//Metadata map[string]string `json:"metadata"` // 特殊属性
	Metadata string `json:"metadata"` // 特殊属性
}

//贷款
type Loan struct{
	creditor_ID string `json:"creditorID"`
	Id   string `json:"id"`
	Number string `json:"number"`
	Rate string `json:"rate"`
	time string `json:"time"`
}

//贷款变更
type LoanChange struct{
	LoanId   string `json:"Loanid"`
	Originalnumber string `json:"Originalnumber"`
	Newnumber string `json:"Newnumber"`
}


// 资产变更
type AssetHistory struct {
	AssetId        string `json:"asset_id"`
	OriginOwnerId  string `json:"origin_owner_id"`
	CurrentOwnerId string `json:"current_owner_id"`
}

func constructUserKey(userId string) string {
	return fmt.Sprintf("user_%s", userId)
}

func constructAssetKey(assetId string) string {
	return fmt.Sprintf("asset_%s", assetId)
}

func constructLoanKey (LoanID string)  string{
	return fmt.Sprintf("Loan_%s",LoanID)
}

// 用户开户
func userRegister(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 套路1：检查参数的个数
	if len(args) != 2 {
		return shim.Error("not enough args")
	}

	// 套路2：验证参数的正确性
	name := args[0]
	id := args[1]
	email := args[2]
	phone := args[3]
	if name == "" || id == "" {
		return shim.Error("invalid args")
	}

	// 套路3：验证数据是否存在 应该存在 or 不应该存在
	if userBytes, err := stub.GetState(constructUserKey(id)); err == nil && len(userBytes) != 0 {
		return shim.Error("user already exist")
	}

	// 套路4：写入状态
	user := &User{
		Name:   name,
		Id:     id,
		email: email,
		phone: phone,
		Assets: make([]string, 0),
		Loan: make([]string, 0),
	}

	// 序列化对象
	userBytes, err := json.Marshal(user)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error %s", err))
	}

	if err := stub.PutState(constructUserKey(id), userBytes); err != nil {
		return shim.Error(fmt.Sprintf("put user error %s", err))
	}

	// 成功返回
	return shim.Success(nil)
}

// 用户销户
func userDestroy(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 套路1：检查参数的个数
	if len(args) != 1 {
		return shim.Error("not enough args")
	}

	// 套路2：验证参数的正确性
	id := args[0]
	if id == "" {
		return shim.Error("invalid args")
	}

	// 套路3：验证数据是否存在 应该存在 or 不应该存在
	userBytes, err := stub.GetState(constructUserKey(id))
	if err != nil || len(userBytes) == 0 {
		return shim.Error("user not found")
	}

	// 套路4：写入状态
	if err := stub.DelState(constructUserKey(id)); err != nil {
		return shim.Error(fmt.Sprintf("delete user error: %s", err))
	}

	// 删除用户名下的资产
	user := new(User)
	if err := json.Unmarshal(userBytes, user); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	for _, assetid := range user.Assets {
		if err := stub.DelState(constructAssetKey(assetid)); err != nil {
			return shim.Error(fmt.Sprintf("delete asset error: %s", err))
		}
	}

	return shim.Success(nil)
}

//传入债务ID， 借债人ID，借债金额，借债利率
func LoanEnroll(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 套路1：检查参数的个数
	if len(args) != 4 {
		return shim.Error("not enough args")
	}

	// 套路2：验证参数的正确性
	Loan_ID := args[0]
	creditor_ID := args[1]
	number := args[3]
	rate := args[4]


	if Loan_ID == "" || creditor_ID == ""  {
		return shim.Error("invalid args")
	}

	// 套路3：验证数据是否存在 应该存在 or 不应该存在
	creditoruserBytes, err := stub.GetState(constructUserKey(creditor_ID))
	if err != nil || len(creditoruserBytes) == 0 {
		return shim.Error("creditor user not found")
	}


	if LoanBytes, err := stub.GetState(constructAssetKey(Loan_ID)); err == nil && len(LoanBytes) != 0 {
		return shim.Error("Loan already exist")
	}

	// 套路4：写入状态
	// 1. 写入资产对象 2. 更新用户对象 3. 写入资产变更记录
	loan := &Loan{
		creditor_ID:       creditor_ID,
		Id: Loan_ID,
		Number: number,
		Rate: rate,
	}
	loanBytes, err := json.Marshal(loan)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal loan error: %s", err))
	}
	if err := stub.PutState(constructAssetKey(Loan_ID), loanBytes); err != nil {
		return shim.Error(fmt.Sprintf("save loan error: %s", err))
	}

	user := new(User)
	// 反序列化user
	if err := json.Unmarshal(creditoruserBytes, user); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal creditor user error: %s", err))
	}

	user.Loan = append(user.Loan, Loan_ID)
	// 序列化user
	creditoruserBytes, err = json.Marshal(user)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(user.Id), creditoruserBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 资产变更历史
	history := &LoanChange{
		loan.Id,
		"0",
		number,
	}
	historyBytes, err := json.Marshal(history)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal assert history error: %s", err))
	}

	historyKey, err := stub.CreateCompositeKey("history", []string{
		Loan_ID,
		originOwner,
	})
	if err != nil {
		return shim.Error(fmt.Sprintf("create key error: %s", err))
	}

	if err := stub.PutState(historyKey, historyBytes); err != nil {
		return shim.Error(fmt.Sprintf("save assert history error: %s", err))
	}

	return shim.Success(nil)
}

// 资产登记
func assetEnroll(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 套路1：检查参数的个数
	if len(args) != 4 {
		return shim.Error("not enough args")
	}

	// 套路2：验证参数的正确性
	assetName := args[0]
	assetId := args[1]
	metadata := args[2]
	ownerId := args[3]
	if assetName == "" || assetId == "" || ownerId == "" {
		return shim.Error("invalid args")
	}

	// 套路3：验证数据是否存在 应该存在 or 不应该存在
	userBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(userBytes) == 0 {
		return shim.Error("user not found")
	}

	if assetBytes, err := stub.GetState(constructAssetKey(assetId)); err == nil && len(assetBytes) != 0 {
		return shim.Error("asset already exist")
	}

	// 套路4：写入状态
	// 1. 写入资产对象 2. 更新用户对象 3. 写入资产变更记录
	asset := &Asset{
		Name:     assetName,
		Id:       assetId,
		Metadata: metadata,
	}
	assetBytes, err := json.Marshal(asset)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal asset error: %s", err))
	}
	if err := stub.PutState(constructAssetKey(assetId), assetBytes); err != nil {
		return shim.Error(fmt.Sprintf("save asset error: %s", err))
	}

	user := new(User)
	// 反序列化user
	if err := json.Unmarshal(userBytes, user); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	user.Assets = append(user.Assets, assetId)
	// 序列化user
	userBytes, err = json.Marshal(user)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(user.Id), userBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 资产变更历史
	history := &AssetHistory{
		AssetId:        assetId,
		OriginOwnerId:  originOwner,
		CurrentOwnerId: ownerId,
	}
	historyBytes, err := json.Marshal(history)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal assert history error: %s", err))
	}

	historyKey, err := stub.CreateCompositeKey("history", []string{
		assetId,
		originOwner,
		ownerId,
	})
	if err != nil {
		return shim.Error(fmt.Sprintf("create key error: %s", err))
	}

	if err := stub.PutState(historyKey, historyBytes); err != nil {
		return shim.Error(fmt.Sprintf("save assert history error: %s", err))
	}

	return shim.Success(nil)
}

// 资产转让
func assetExchange(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 套路1：检查参数的个数
	if len(args) != 3 {
		return shim.Error("not enough args")
	}

	// 套路2：验证参数的正确性
	ownerId := args[0]
	assetId := args[1]
	currentOwnerId := args[2]
	if ownerId == "" || assetId == "" || currentOwnerId == "" {
		return shim.Error("invalid args")
	}

	// 套路3：验证数据是否存在 应该存在 or 不应该存在
	originOwnerBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(originOwnerBytes) == 0 {
		return shim.Error("user not found")
	}

	currentOwnerBytes, err := stub.GetState(constructUserKey(currentOwnerId))
	if err != nil || len(currentOwnerBytes) == 0 {
		return shim.Error("user not found")
	}

	assetBytes, err := stub.GetState(constructAssetKey(assetId))
	if err != nil || len(assetBytes) == 0 {
		return shim.Error("asset not found")
	}

	// 校验原始拥有者确实拥有当前变更的资产
	originOwner := new(User)
	// 反序列化user
	if err := json.Unmarshal(originOwnerBytes, originOwner); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	aidexist := false
	for _, aid := range originOwner.Assets {
		if aid == assetId {
			aidexist = true
			break
		}
	}
	if !aidexist {
		return shim.Error("asset owner not match")
	}

	// 套路4：写入状态
	// 1. 原是拥有者删除资产id 2. 新拥有者加入资产id 3. 资产变更记录
	assetIds := make([]string, 0)
	for _, aid := range originOwner.Assets {
		if aid == assetId {
			continue
		}

		assetIds = append(assetIds, aid)
	}
	originOwner.Assets = assetIds

	originOwnerBytes, err = json.Marshal(originOwner)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(ownerId), originOwnerBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 当前拥有者插入资产id
	currentOwner := new(User)
	// 反序列化user
	if err := json.Unmarshal(currentOwnerBytes, currentOwner); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s", err))
	}
	currentOwner.Assets = append(currentOwner.Assets, assetId)

	currentOwnerBytes, err = json.Marshal(currentOwner)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s", err))
	}
	if err := stub.PutState(constructUserKey(currentOwnerId), currentOwnerBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s", err))
	}

	// 插入资产变更记录
	history := &AssetHistory{
		AssetId:        assetId,
		OriginOwnerId:  ownerId,
		CurrentOwnerId: currentOwnerId,
	}
	historyBytes, err := json.Marshal(history)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal assert history error: %s", err))
	}

	historyKey, err := stub.CreateCompositeKey("history", []string{
		assetId,
		ownerId,
		currentOwnerId,
	})
	if err != nil {
		return shim.Error(fmt.Sprintf("create key error: %s", err))
	}

	if err := stub.PutState(historyKey, historyBytes); err != nil {
		return shim.Error(fmt.Sprintf("save assert history error: %s", err))
	}

	return shim.Success(nil)
}

// 用户查询
func queryUser(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 套路1：检查参数的个数
	if len(args) != 1 {
		return shim.Error("not enough args")
	}

	// 套路2：验证参数的正确性
	ownerId := args[0]
	if ownerId == "" {
		return shim.Error("invalid args")
	}

	// 套路3：验证数据是否存在 应该存在 or 不应该存在
	userBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(userBytes) == 0 {
		return shim.Error("user not found")
	}

	return shim.Success(userBytes)
}

// 资产查询
func queryAsset(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 套路1：检查参数的个数
	if len(args) != 1 {
		return shim.Error("not enough args")
	}

	// 套路2：验证参数的正确性
	assetId := args[0]
	if assetId == "" {
		return shim.Error("invalid args")
	}

	// 套路3：验证数据是否存在 应该存在 or 不应该存在
	assetBytes, err := stub.GetState(constructAssetKey(assetId))
	if err != nil || len(assetBytes) == 0 {
		return shim.Error("asset not found")
	}

	return shim.Success(assetBytes)
}

func queryLoan(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 套路1：检查参数的个数
	if len(args) != 1 {
		return shim.Error("not enough args")
	}

	// 套路2：验证参数的正确性
	loanId := args[0]
	if loanId == "" {
		return shim.Error("invalid args")
	}

	// 套路3：验证数据是否存在 应该存在 or 不应该存在
	loanBytes, err := stub.GetState(constructLoanKey(loanId))
	if err != nil || len(loanBytes) == 0 {
		return shim.Error("loan not found")
	}

	return shim.Success(loanBytes)
}

// 资产变更历史查询
func queryAssetHistory(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 套路1：检查参数的个数
	if len(args) != 2 && len(args) != 1 {
		return shim.Error("not enough args")
	}

	// 套路2：验证参数的正确性
	assetId := args[0]
	if assetId == "" {
		return shim.Error("invalid args")
	}

	queryType := "all"
	if len(args) == 2 {
		queryType = args[1]
	}

	if queryType != "all" && queryType != "enroll" && queryType != "exchange" {
		return shim.Error(fmt.Sprintf("queryType unknown %s", queryType))
	}

	// 套路3：验证数据是否存在 应该存在 or 不应该存在
	assetBytes, err := stub.GetState(constructAssetKey(assetId))
	if err != nil || len(assetBytes) == 0 {
		return shim.Error("asset not found")
	}

	// 查询相关数据
	keys := make([]string, 0)
	keys = append(keys, assetId)
	switch queryType {
	case "enroll":
		keys = append(keys, originOwner)
	case "exchange", "all": // 不添加任何附件key
	default:
		return shim.Error(fmt.Sprintf("unsupport queryType: %s", queryType))
	}
	result, err := stub.GetStateByPartialCompositeKey("history", keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("query history error: %s", err))
	}
	defer result.Close()

	histories := make([]*AssetHistory, 0)
	for result.HasNext() {
		historyVal, err := result.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("query error: %s", err))
		}

		history := new(AssetHistory)
		if err := json.Unmarshal(historyVal.GetValue(), history); err != nil {
			return shim.Error(fmt.Sprintf("unmarshal error: %s", err))
		}

		// 过滤掉不是资产转让的记录
		if queryType == "exchange" && history.OriginOwnerId == originOwner {
			continue
		}

		histories = append(histories, history)
	}

	historiesBytes, err := json.Marshal(histories)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal error: %s", err))
	}

	return shim.Success(historiesBytes)
}


func (c *AssertsExchangeCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (c *AssertsExchangeCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	funcName, args := stub.GetFunctionAndParameters()

	switch funcName {
	case "userRegister":
		return userRegister(stub, args)
	case "userDestroy":
		return userDestroy(stub, args)
	case "assetEnroll":
		return assetEnroll(stub, args)
	case "assetExchange":
		return assetExchange(stub, args)
	case "queryUser":
		return queryUser(stub, args)
	case "queryAsset":
		return queryAsset(stub, args)
	case "queryAssetHistory":
		return queryAssetHistory(stub, args)
	case"LoanEnroll":
		return LoanEnroll(stub,args)
	case "queryLoan":
		return queryLoan(stub,args)
	default:
		return shim.Error(fmt.Sprintf("unsupported function: %s", funcName))
	}
}

func main() {
	err := shim.Start(new(AssertsExchangeCC))
	if err != nil {
		fmt.Printf("Error starting AssertsExchange chaincode: %s", err)
	}
}
