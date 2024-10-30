package controllor

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sushi/model"
	"sushi/service"
	"sushi/utils"
	"sushi/utils/config"
	"sushi/utils/custom_errors"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Controller struct {
	service *service.Service
	log     *logrus.Logger
	conf    *config.Config
}

func NewControllor(service *service.Service, log *logrus.Logger, conf *config.Config) *Controller {
	return &Controller{service: service, log: log, conf: conf}
}

func (con *Controller) HandlePing(c *gin.Context) {

	con.log.Debug("handling ping...")
	utils.SuccessResponse(c, "pong", "")
}

func (con *Controller) HandleNewPlayer(c *gin.Context) {

	user, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	err = con.service.NewPlayer(user.Mail, user.Sub)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "", "")
}

type UserTransJson struct {
	Amount float64 `json:"token_amount"`
	UUID   string  `json:"uuid"`
}

type EarnJson struct {
	SessionID       string             `json:"session_id"`
	SessionDuration int                `json:"session_duration"`
	Players         []model.EarnPlayer `json:"players"`
}
type UserInfo struct {
	Mail string `json:"mail"`
	Sub  string `json:"sub"`
	Name string `json:"name"`
}

func getUserInfo(c *gin.Context) (*UserInfo, error) {
	sub, f := c.Get("sub")
	if !f {
		fmt.Println("not find user" + sub.(string))
		return nil, custom_errors.GET_USERINFO_ERROR
	}
	mail, f := c.Get("mail")
	if !f {
		fmt.Println("not find user" + mail.(string))
		return nil, custom_errors.GET_USERINFO_ERROR
	}
	name, f := c.Get("name")
	if !f {
		fmt.Println("not find user" + name.(string))
		return nil, custom_errors.GET_USERINFO_ERROR
	}
	return &UserInfo{
		Mail: mail.(string),
		Sub:  sub.(string),
		Name: name.(string),
	}, nil
}

func (con *Controller) HandleEarn(c *gin.Context) {
	var json EarnJson
	if err := c.ShouldBindJSON(&json); err != nil {
		//con.log.Error(errors.BIND_JSON_ERROR)
		utils.ErrorResponse(c, 401, custom_errors.BIND_JSON_ERROR.Error(), "")
		return
	}
	if json.SessionID == "" {
		utils.ErrorResponse(c, 401, custom_errors.UNVALUABLE_SESSION_ID_ERROR.Error(), "")
		return
	}
	err := con.service.CheckSessionID(json.SessionID)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	err = con.service.Earn(json.Players, json.SessionID)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "", "")
}
func (con *Controller) HandleSwap(c *gin.Context) {
	userinfo, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	var json UserTransJson
	if err := c.ShouldBindJSON(&json); err != nil {
		//con.log.Error(errors.BIND_JSON_ERROR)
		utils.ErrorResponse(c, 401, custom_errors.BIND_JSON_ERROR.Error(), "")
		return
	}
	if json.Amount < 0.001 {
		//con.log.Error(errors.AMOUNT_ERROR)
		utils.ErrorResponse(c, 401, custom_errors.AMOUNT_ERROR.Error(), "")
		return
	}
	err = con.service.Swap(userinfo.Sub, json.Amount)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "", "")
}
func (con *Controller) HandleApplyWithdraw(c *gin.Context) {
	userinfo, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return

	}
	var json UserTransJson
	if err := c.ShouldBindJSON(&json); err != nil {
		//con.log.Error(errors.BIND_JSON_ERROR)
		utils.ErrorResponse(c, 401, custom_errors.BIND_JSON_ERROR.Error(), "")
		return
	}
	if json.Amount < 0.001 {
		//con.log.Error(errors.AMOUNT_ERROR)
		utils.ErrorResponse(c, 401, custom_errors.AMOUNT_ERROR.Error(), "")
		return
	}
	err = con.service.ApplyWithdraw(userinfo.Sub, json.Amount)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "", "")
}

type WithdrawJson struct {
	ID   uint
	Hash string
}

func (con *Controller) HandleHandleWithdraw(c *gin.Context) {
	var json WithdrawJson
	if err := c.ShouldBindJSON(&json); err != nil {
		//con.log.Error(errors.BIND_JSON_ERROR)
		utils.ErrorResponse(c, 401, custom_errors.BIND_JSON_ERROR.Error(), "")
		return
	}
	err := con.service.HandleWithdraw(json.ID)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "", "")
}
func (con *Controller) HandleConfirmWithdraw(c *gin.Context) {
	var json WithdrawJson
	if err := c.ShouldBindJSON(&json); err != nil {
		//con.log.Error(errors.BIND_JSON_ERROR)
		utils.ErrorResponse(c, 401, custom_errors.BIND_JSON_ERROR.Error(), "")
		return
	}
	if json.Hash == "" {
		err := con.service.WithDrawFail(json.ID)
		if err != nil {
			utils.ErrorResponse(c, 501, err.Error(), "")
			return
		}
	} else {
		err := con.service.ConfirmWithdraw(json.ID, json.Hash)
		if err != nil {
			utils.ErrorResponse(c, 501, err.Error(), "")
			return
		}
	}
	utils.SuccessResponse(c, "", "")
}
func (con *Controller) HandleGetMonthWithdraw(c *gin.Context) {
	userinfo, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return

	}

	limit, err := con.service.GetMonthWithdraw(userinfo.Sub)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "", limit)
}

func (con *Controller) HandleGetBalance(c *gin.Context) {
	userinfo, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}

	food, speak, err := con.service.GetBalance(userinfo.Sub)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	balance := Balance{
		Food:  food,
		Speak: speak,
	}
	utils.SuccessResponse(c, "", balance)
}

type Balance struct {
	Food  float64
	Speak float64
}

func (con *Controller) HandleGetEarnRecords(c *gin.Context) {
	userinfo, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}

	records, err := con.service.GetEarnRecord(userinfo.Sub)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "", records)
}
func (con *Controller) HandleGetSwapRecords(c *gin.Context) {
	userinfo, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	records, err := con.service.GetSwapRecord(userinfo.Sub)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "", records)
}
func (con *Controller) HandleGetWithdrawRecords(c *gin.Context) {
	userinfo, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}

	records, err := con.service.GetWithdrawRecord(userinfo.Sub)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "", records)
}
func (con *Controller) HandleGetEarnTotal(c *gin.Context) {
	userinfo, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}

	earn, err := con.service.GetEarnTotal(userinfo.Sub)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "", earn)
}
func (con *Controller) HandleGetWithdrawTotal(c *gin.Context) {
	userinfo, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	var json UserTransJson
	if err := c.ShouldBindJSON(&json); err != nil {
		utils.ErrorResponse(c, 401, custom_errors.BIND_JSON_ERROR.Error(), "")
		return
	}
	withdraw, err := con.service.GetWithdrawTotal(userinfo.Sub)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "", withdraw)
}

func (con *Controller) HandleGetUserInfo(c *gin.Context) {
	userinfo, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	player, err := con.service.GetUserInfo(userinfo.Mail, userinfo.Sub)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return

	}
	utils.SuccessResponse(c, "ok", player)
}

type FirebaseUserInfo struct {
	Tier   uint `json:"tier"`
	Region uint `json:"region"`
}

type EthAddress struct {
	EthAddress string `json:"eth_address"`
}

func (con *Controller) HandleEditEthAddress(c *gin.Context) {
	userinfo, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}

	var json EthAddress
	if err := c.ShouldBindJSON(&json); err != nil {
		utils.ErrorResponse(c, 401, custom_errors.BIND_JSON_ERROR.Error(), "")
		return
	}

	if !strings.HasPrefix(json.EthAddress, "0x") {
		utils.ErrorResponse(c, 410, "eth address must start with 0x", "")
		return
	}
	if len(json.EthAddress) != 42 {
		utils.ErrorResponse(c, 410, "eth address length must be 42", "")
		return
	}
	err = con.service.EditEthAddress(userinfo.Sub, json.EthAddress)
	if err != nil {
		if errors.Is(err, custom_errors.ETH_ADDRESS_EXIST_ERROR) {
			utils.ErrorResponse(c, 411, err.Error(), "")
			return
		}
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "ok", "")
}

func (con *Controller) HandleGetNfts(c *gin.Context) {
	userinfo, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	queryParams := c.Request.URL.Query()

	// Get a specific query parameter by key
	limitStr := queryParams.Get("limit")
	pageStr := queryParams.Get("page")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}

	nfts, err := con.service.GetUserNfts(userinfo.Sub, page, limit)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return

	}
	utils.SuccessResponse(c, "ok", nfts)
}
func (con *Controller) HandleEarnAllowFreebie(c *gin.Context) {
	var json EarnJson
	if err := c.ShouldBindJSON(&json); err != nil {
		utils.ErrorResponse(c, 401, custom_errors.BIND_JSON_ERROR.Error(), "")
		return
	}
	if json.SessionID == "" {
		utils.ErrorResponse(c, 401, custom_errors.UNVALUABLE_SESSION_ID_ERROR.Error(), "")
		return
	}
	err := con.service.CheckSessionID(json.SessionID)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	err = con.service.EarnAllowFreebie(json.Players, json.SessionID)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "", "")
}

func (con *Controller) HandleGetFreebieRecords(c *gin.Context) {
	userinfo, err := getUserInfo(c)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}

	records, err := con.service.GetFreebieRecord(userinfo.Sub)
	if err != nil {
		utils.ErrorResponse(c, 501, err.Error(), "")
		return
	}
	utils.SuccessResponse(c, "", records)
}
