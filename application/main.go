package main

import (
	"fmt"

	"context"
	"time"

	"net/http"

	"bytes"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/event"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

func main() {
	router := gin.Default()

	// 定义路由
	{
		router.POST("/users", userRegister)
		router.GET("/users/:id", queryUser)
		router.DELETE("/users/:id", deleteUser)
		router.GET("/assets/get/:id", queryAsset)
		router.POST("/assets/enroll", assetsEnroll)
		router.POST("/assets/exchange", assetsExchange)
		router.GET("/assets/exchange/history", assetsExchangeHistory)
		router.POST("/loan/enroll", loansEnroll)
		router.POST("/loan/exchange", loansExchange)
		router.GET("/loan/exchange/history", loansExchangeHistory)
	}

	router.Run() // listen and serve on 0.0.0.0:8080
}

type UserRegisterRequest struct {
	Id   string `form:"id" binding:"required"`
	Name string `form:"name" binding:"required"`
}

// 用户开户
func userRegister(ctx *gin.Context) {
	// 参数处理
	req := new(UserRegisterRequest)
	if err := ctx.ShouldBind(req); err != nil {
		ctx.AbortWithError(400, err)
		return
	}

	// 区块链交互
	resp, err := channelExecute("userRegister", [][]byte{
		[]byte(req.Name),
		[]byte(req.Id),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// 查询用户
func queryUser(ctx *gin.Context) {
	userId := ctx.Param("id")

	resp, err := channelQuery("queryUser", [][]byte{
		[]byte(userId),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	//ctx.JSON(http.StatusOK, resp)
	ctx.String(http.StatusOK, bytes.NewBuffer(resp.Payload).String())
}

// 用户销户
func deleteUser(ctx *gin.Context) {
	userId := ctx.Param("id")

	resp, err := channelExecute("userDestroy", [][]byte{
		[]byte(userId),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// 资产查询
func queryAsset(ctx *gin.Context) {
	assetId := ctx.Param("id")

	resp, err := channelQuery("queryAsset", [][]byte{
		[]byte(assetId),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	//ctx.JSON(http.StatusOK, resp)
	ctx.String(http.StatusOK, bytes.NewBuffer(resp.Payload).String())
}

type AssetsEnrollRequest struct {
	AssetId   string `form:"assetsid" binding:"required"`
	AssetName string `form:"assetname" binding:"required"`
	Metadata  string `form:"metadata" binding:"required"`
	OwnerId   string `form:"ownerid" binding:"required"`
}

// 资产登记
func assetsEnroll(ctx *gin.Context) {
	req := new(AssetsEnrollRequest)
	if err := ctx.ShouldBind(req); err != nil {
		ctx.AbortWithError(400, err)
		return
	}

	resp, err := channelExecute("assetEnroll", [][]byte{
		[]byte(req.AssetName),
		[]byte(req.AssetId),
		[]byte(req.Metadata),
		[]byte(req.OwnerId),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

type AssetsExchangeRequest struct {
	AssetId        string `form:"assetsid" binding:"required"`
	OriginOwnerId  string `form:"originownerid" binding:"required"`
	CurrentOwnerId string `form:"currentownerid" binding:"required"`
}

// 资产转让
func assetsExchange(ctx *gin.Context) {
	req := new(AssetsExchangeRequest)
	if err := ctx.ShouldBind(req); err != nil {
		ctx.AbortWithError(400, err)
		return
	}

	resp, err := channelExecute("assetExchange", [][]byte{
		[]byte(req.OriginOwnerId),
		[]byte(req.AssetId),
		[]byte(req.CurrentOwnerId),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// 资产历史变更记录
func assetsExchangeHistory(ctx *gin.Context) {
	assetId := ctx.Query("assetid")
	queryType := ctx.Query("querytype")

	resp, err := channelQuery("queryAssetHistory", [][]byte{
		[]byte(assetId),
		[]byte(queryType),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	//ctx.JSON(http.StatusOK, resp)
	ctx.String(http.StatusOK, bytes.NewBuffer(resp.Payload).String())
}

type LoansEnrollRequest struct {
	LoanId   string `form:"loansid" binding:"required"`
	LoanNumber string `form:"loannumber" binding:"required"`
	Metadata  string `form:"metadata" binding:"required"`
	OwnerId   string `form:"ownerid" binding:"required"`
	LoanRate string `form:"loanrate" binding:"required"`

}

// 贷款登记
func loansEnroll(ctx *gin.Context) {
	req := new(LoansEnrollRequest)
	if err := ctx.ShouldBind(req); err != nil {
		ctx.AbortWithError(400, err)
		return
	}

	resp, err := channelExecute("loanEnroll", [][]byte{
		[]byte(req.LoanNumber),
		[]byte(req.LoanId),
		[]byte(req.Metadata),
		[]byte(req.OwnerId),
		[]byte(req.LoanRate),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

type LoansExchangeRequest struct {
	LoanId        string `form:"loansid" binding:"required"`
	OriginOwnerId  string `form:"originownerid" binding:"required"`
	CurrentOwnerId string `form:"currentownerid" binding:"required"`
}

// 贷款转让
func loansExchange(ctx *gin.Context) {
	req := new(LoansExchangeRequest)
	if err := ctx.ShouldBind(req); err != nil {
		ctx.AbortWithError(400, err)
		return
	}

	resp, err := channelExecute("loanExchange", [][]byte{
		[]byte(req.OriginOwnerId),
		[]byte(req.LoanId),
		[]byte(req.CurrentOwnerId),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// 贷款历史变更记录
func loansExchangeHistory(ctx *gin.Context) {
	loanId := ctx.Query("loanid")
	queryType := ctx.Query("querytype")

	resp, err := channelQuery("queryLoanHistory", [][]byte{
		[]byte(loanId),
		[]byte(queryType),
	})

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	//ctx.JSON(http.StatusOK, resp)
	ctx.String(http.StatusOK, bytes.NewBuffer(resp.Payload).String())
}



var (
	sdk           *fabsdk.FabricSDK
	channelName   = "assetschannel"
	chaincodeName = "assets"
	org           = "org1"
	user          = "Admin"
	//configPath    = "$GOPATH/src/github.com/hyperledger/fabric/imocc/application/config.yaml"
	configPath = "./config.yaml"
)

func init() {
	var err error
	sdk, err = fabsdk.New(config.FromFile(configPath))
	if err != nil {
		panic(err)
	}
}

// 区块链管理
func manageBlockchain() {
	// 表明身份
	ctx := sdk.Context(fabsdk.WithOrg(org), fabsdk.WithUser(user))

	cli, err := resmgmt.New(ctx)
	if err != nil {
		panic(err)
	}

	// 具体操作
	cli.SaveChannel(resmgmt.SaveChannelRequest{}, resmgmt.WithOrdererEndpoint("orderer.imocc.com"), resmgmt.WithTargetEndpoints())
}

// 区块链数据查询 账本的查询
func queryBlockchain() {
	ctx := sdk.ChannelContext(channelName, fabsdk.WithOrg(org), fabsdk.WithUser(user))

	cli, err := ledger.New(ctx)
	if err != nil {
		panic(err)
	}

	resp, err := cli.QueryInfo(ledger.WithTargetEndpoints("peer0.org1.imocc.com"))
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)

	// 1
	cli.QueryBlockByHash(resp.BCI.CurrentBlockHash)

	// 2
	for i := uint64(0); i <= resp.BCI.Height; i++ {
		cli.QueryBlock(i)
	}
}

// 区块链交互
func channelExecute(fcn string, args [][]byte) (channel.Response, error) {
	ctx := sdk.ChannelContext(channelName, fabsdk.WithOrg(org), fabsdk.WithUser(user))

	cli, err := channel.New(ctx)
	if err != nil {
		return channel.Response{}, err
	}

	// 状态更新，insert/update/delete
	resp, err := cli.Execute(channel.Request{
		ChaincodeID: chaincodeName,
		Fcn:         fcn,
		Args:        args,
	}, channel.WithTargetEndpoints("peer0.org1.imocc.com"))
	if err != nil {
		return channel.Response{}, err
	}

	// 链码事件监听
	go func() {
		// channel
		reg, ccevt, err := cli.RegisterChaincodeEvent(chaincodeName, "eventname")
		if err != nil {
			return
		}
		defer cli.UnregisterChaincodeEvent(reg)

		timeoutctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		for {
			select {
			case evt := <-ccevt:
				fmt.Printf("received event of tx %s: %+v", resp.TransactionID, evt)
			case <-timeoutctx.Done():
				fmt.Println("event timeout, exit!")
				return
			}
		}

		// event
		//eventcli, err := event.New(ctx)
		//if err != nil {
		//	return
		//}

		//eventcli.RegisterChaincodeEvent(chaincodeName, "eventname")
	}()

	// 交易状态事件监听
	go func() {
		eventcli, err := event.New(ctx)
		if err != nil {
			return
		}

		reg, status, err := eventcli.RegisterTxStatusEvent(string(resp.TransactionID))
		defer eventcli.Unregister(reg) // 注册必有注销

		timeoutctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		for {
			select {
			case evt := <-status:
				fmt.Printf("received event of tx %s: %+v", resp.TransactionID, evt)
			case <-timeoutctx.Done():
				fmt.Println("event timeout, exit!")
				return
			}
		}
	}()

	return resp, nil
}

func channelQuery(fcn string, args [][]byte) (channel.Response, error) {
	ctx := sdk.ChannelContext(channelName, fabsdk.WithOrg(org), fabsdk.WithUser(user))

	cli, err := channel.New(ctx)
	if err != nil {
		return channel.Response{}, err
	}

	// 状态的查询，select
	return cli.Query(channel.Request{
		ChaincodeID: chaincodeName,
		Fcn:         fcn,
		Args:        args,
	}, channel.WithTargetEndpoints("peer0.org1.imocc.com"))
}

// 事件监听
func eventHandle() {
	ctx := sdk.ChannelContext(channelName, fabsdk.WithOrg(org), fabsdk.WithUser(user))

	cli, err := event.New(ctx)
	if err != nil {
		panic(err)
	}

	// 交易状态事件
	// 链码事件 业务事件
	// 区块事件
	reg, blkevent, err := cli.RegisterBlockEvent()
	if err != nil {
		panic(err)
	}
	defer cli.Unregister(reg)

	timeoutctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	for {
		select {
		case evt := <-blkevent:
			fmt.Printf("received a block", evt)
		case <-timeoutctx.Done():
			fmt.Println("event timeout, exit!")
			return
		}
	}
}
