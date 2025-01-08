package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"sushi/model"
	"sushi/utils"
	"sushi/utils/DB"
	"sushi/utils/config"
	"sushi/utils/custom_errors"
	"time"

	"github.com/jinzhu/now"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Service struct {
	db        *DB.DB
	log       *logrus.Logger
	conf      *config.Config
	rate      float64
	swapLimit float64
	Firebase  *utils.Firebase
	Ctx       *context.Context
}

func (svc *Service) GetRate() float64 {
	return svc.rate
}

func (svc *Service) SetRate(rate float64) {
	svc.log.Info("rate set to ", rate)
	svc.rate = rate
}

func (svc *Service) GetSwapLimit() float64 {
	return svc.swapLimit
}

func (svc *Service) SetSwapLimit(swapLimit float64) {
	svc.swapLimit = swapLimit
	svc.log.Info("swap limit set to ", swapLimit)
}

func (svc *Service) GetBalance(sub string) (float64, float64, error) {
	var player model.Player
	var err error
	var foodTotal model.EarnTotal
	var swapTotal model.SwapTotal
	var speakTotal model.WithdrawTotal
	err = svc.checkPlayer(sub)
	if err != nil {
		return 0.0, 0.0, err
	}
	player, err = svc.getPlayerBySub(sub)
	if err != nil {
		return 0, 0, err
	}
	err = svc.db.DB.Transaction(func(tx *gorm.DB) error {

		result := svc.db.DB.Where("user_id=?", player.UserId).First(&foodTotal)
		if result.Error != nil {
			return result.Error
		}
		result = svc.db.DB.Where("user_id=?", player.UserId).First(&swapTotal)
		if result.Error != nil {
			return result.Error
		}
		result = svc.db.DB.Where("user_id=?", player.UserId).First(&speakTotal)
		if result.Error != nil {
			return result.Error
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	freebieEarnTotal := svc.getFreebieEarnTotal(player.UserId)

	return foodTotal.EarnTotal - swapTotal.SwappedFood + freebieEarnTotal, swapTotal.SwappedSpeak - speakTotal.WithdrawTotal, nil
}

func NewService(db *DB.DB, log *logrus.Logger, conf *config.Config, ctx context.Context) *Service {
	_firebase := utils.NewFirebase(ctx)
	return &Service{
		db:        db,
		log:       log,
		conf:      conf,
		rate:      1000,
		swapLimit: 10,
		Firebase:  _firebase,
		Ctx:       &ctx,
	}
}

func (svc *Service) NewPlayer(mail string, sub string) error {
	var er error
	var player model.Player

	err := svc.checkPlayer(mail)
	if !errors.Is(err, custom_errors.PLAYER_NOT_EXIST_ERROR) {
		//Player exist
		return custom_errors.PLAYER_EXIST_ERROR
	}
	err = svc.db.DB.Transaction(func(tx *gorm.DB) error {

		er = svc.createPlayer(mail, sub)
		if er != nil {
			return er
		}

		player, er = svc.getPlayerBySub(sub)
		if er != nil {
			return er
		}

		er = svc.createSpeakTotal(player.UserId)
		if er != nil {
			return er
		}

		er = svc.createFoodTotal(player.UserId)
		if er != nil {
			return er
		}
		er = svc.createSwapTotal(player.UserId)
		if er != nil {
			return er
		}

		return nil
	})
	if err != nil {
		svc.log.Error("Failed to create new player: %d", mail)
		return err
	}
	svc.log.Info("New player:", mail)
	return nil
}

func (svc *Service) Earn(players []model.EarnPlayer, sessionId string) error {
	for _, player := range players {
		err := svc.checkPlayer(player.Sub)
		if err != nil {
			return custom_errors.PLAYER_NOT_EXIST_ERROR
		}
	}

	err := svc.db.DB.Transaction(func(tx *gorm.DB) error {
		for _, player := range players {
			temPlayer, err := svc.getPlayerBySub(player.Sub)
			if err != nil {
				return err
			}
			amount := player.Amount * player.Rarity
			amount64 := float64(amount)
			err = svc.addFoodTotal(temPlayer.UserId, amount64)
			if err != nil {
				return err
			}
			err = svc.addEarnRecord(temPlayer.UserId, amount64, sessionId)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		svc.log.Error("Failed to earn", sessionId)
		return err
	}
	return nil

}

func (svc *Service) Swap(sub string, speakAmount float64) error {

	var err error
	var player model.Player
	err = svc.checkPlayer(sub)
	if err != nil {
		return custom_errors.PLAYER_NOT_EXIST_ERROR
	}
	food, _, err := svc.GetBalance(sub)
	if err != nil {
		return err
	}
	if speakAmount*svc.rate > food {
		return custom_errors.FOOD_NOT_ENOUGH_ERROR
	}
	player, err = svc.getPlayerBySub(sub)
	if err != nil {
		return err
	}

	err = svc.db.DB.Transaction(func(tx *gorm.DB) error {
		var er error
		er = svc.modifySwapTotal(player.UserId, speakAmount*svc.rate, speakAmount)
		if er != nil {
			return er
		}
		er = svc.addSwapRecord(player.UserId, speakAmount*svc.rate, speakAmount)
		if er != nil {
			return er
		}

		return nil

	})
	if err != nil {
		return err
	}
	return nil
}
func (svc *Service) ApplyWithdraw(sub string, speakAmount float64) error {
	var err error
	var player model.Player
	err = svc.checkPlayer(sub)
	if err != nil {
		return custom_errors.PLAYER_NOT_EXIST_ERROR
	}
	player, err = svc.getPlayerBySub(sub)
	if err != nil {
		return err
	}
	_, speak, err := svc.GetBalance(sub)
	if err != nil {
		return err
	}
	if speak < speakAmount {
		return custom_errors.SPEAK_NOT_ENOUGH_ERROR
	}

	err = svc.db.DB.Transaction(func(tx *gorm.DB) error {
		var er error

		er = svc.addWithdrawRecord(player.UserId, speakAmount)
		if er != nil {
			return er
		}
		er = svc.addSpeakTotal(player.UserId, speakAmount)
		if er != nil {
			return er
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
func (svc *Service) HandleWithdraw(withdrawId uint) error {
	var withdrawRecord model.WithdrawRecord

	result := svc.db.DB.Where("withdraw_id=?", withdrawId).First(&withdrawRecord)
	if result.Error != nil {
		return result.Error
	}
	if withdrawRecord.State != 0 {
		return custom_errors.WITHDRAW_HANDLE_ERROR

	}
	currentTime := time.Now().UTC()
	withdrawRecord.HandleTimestamp = &currentTime
	withdrawRecord.State = 1
	err := svc.db.DB.Save(&withdrawRecord).Error
	if err != nil {
		svc.log.Error(err)
		return err
	}
	return nil
}

func (svc *Service) ConfirmWithdraw(withdrawId uint, hash string) error {
	var withdrawRecord model.WithdrawRecord

	result := svc.db.DB.Where("withdraw_id=?", withdrawId).First(&withdrawRecord)
	if result.Error != nil {
		return result.Error
	}
	if withdrawRecord.State != 1 {
		return custom_errors.WITHDRAW_HANDLE_ERROR

	}
	withdrawRecord.Hash = hash
	currentTime := time.Now().UTC()
	withdrawRecord.ConfirmTimestamp = &currentTime
	withdrawRecord.State = 2
	err := svc.db.DB.Save(&withdrawRecord).Error
	if err != nil {
		svc.log.Error(err)
		return err
	}
	return nil
}

func (svc *Service) WithDrawFail(withdrawId uint) error {
	var err error
	err = svc.db.DB.Transaction(func(tx *gorm.DB) error {
		var withdrawRecord model.WithdrawRecord
		result := svc.db.DB.Where("withdraw_id=?", withdrawId).First(&withdrawRecord)
		if result.Error != nil {
			return result.Error
		}
		if withdrawRecord.State != 1 {
			return custom_errors.WITHDRAW_HANDLE_ERROR
		}
		var speakTotal model.WithdrawTotal
		result = svc.db.DB.Where("user_id=?", withdrawRecord.UserID).First(&speakTotal)
		if result.Error != nil {
			return result.Error
		}
		speakTotal.WithdrawTotal -= withdrawRecord.Amount
		er := svc.db.DB.Save(&speakTotal).Error
		if err != nil {
			svc.log.Error(er)
			return er
		}
		currentTime := time.Now().UTC()
		withdrawRecord.ConfirmTimestamp = &currentTime
		withdrawRecord.State = 3
		err := svc.db.DB.Save(&withdrawRecord).Error
		if err != nil {
			svc.log.Error(err)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil

}

func (svc *Service) createPlayer(mail string, sub string) error {
	player := model.Player{
		Mail:       mail,
		Sub:        sub,
		EthAddress: nil,
		LoginAt:    time.Now(),
	}
	result := svc.db.DB.Create(&player)
	if result.Error != nil {
		//svc.log.Error("CreatePlayerError:")
		return result.Error
	}
	return nil
}

func (svc *Service) createFoodTotal(userId uint) error {
	foodTotal := model.EarnTotal{
		UserID:    userId,
		EarnTotal: 0,
	}
	result := svc.db.DB.Create(&foodTotal)

	if result.Error != nil {
		//svc.log.Error("CreateBalanceError:")
		return result.Error
	}
	return nil
}

func (svc *Service) createSpeakTotal(userId uint) error {
	speakTotal := model.WithdrawTotal{
		UserID:        userId,
		WithdrawTotal: 0,
	}
	result := svc.db.DB.Create(&speakTotal)
	if result.Error != nil {
		//svc.log.Error("CreateBalanceError:")
		return result.Error
	}
	return nil
}
func (svc *Service) createSwapTotal(userId uint) error {
	swapTotal := model.SwapTotal{
		UserID: userId,
	}
	result := svc.db.DB.Create(&swapTotal)
	if result.Error != nil {
		//svc.log.Error("CreateBalanceError:")
		return result.Error
	}
	return nil
}

func (svc *Service) getPlayerBySub(sub string) (model.Player, error) {
	var player model.Player
	result := svc.db.DB.Where("sub = ?", sub).First(&player)
	if result.Error != nil {
		return model.Player{}, result.Error
	}
	return player, nil
}

func (svc *Service) addFoodTotal(userId uint, amount float64) error {
	var foodTotal model.EarnTotal
	err := svc.db.DB.Where("user_id = ?", userId).First(&foodTotal).Error
	if err != nil {
		svc.log.Error(err)
		return err
	}
	if amount < 0 {
		return custom_errors.AMOUNT_ERROR
	}

	foodTotal.EarnTotal += amount
	err = svc.db.DB.Save(&foodTotal).Error
	if err != nil {
		svc.log.Error(err)
		return err
	}
	return nil
}

func (svc *Service) addSpeakTotal(userId uint, amount float64) error {
	var speakTotal model.WithdrawTotal
	err := svc.db.DB.Where("user_id = ?", userId).First(&speakTotal).Error
	if err != nil {
		svc.log.Error(err)
		return err
	}

	if amount < 0 {
		return custom_errors.AMOUNT_ERROR
	}

	speakTotal.WithdrawTotal += amount
	err = svc.db.DB.Save(&speakTotal).Error
	if err != nil {
		svc.log.Error(err)
		return err
	}
	return nil
}

func (svc *Service) modifySwapTotal(userId uint, FoodAmount float64, SpeakAmount float64) error {
	var swapTotal model.SwapTotal
	err := svc.db.DB.Where("user_id = ?", userId).First(&swapTotal).Error
	if err != nil {
		return err
	}
	swapTotal.SwappedFood += FoodAmount
	swapTotal.SwappedSpeak += SpeakAmount
	err = svc.db.DB.Save(&swapTotal).Error
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) addEarnRecord(userId uint, amount float64, sessionId string) error {

	earnRecord := model.EarnRecord{
		UserID:    userId,
		Amount:    amount,
		SessionID: sessionId,
	}
	result := svc.db.DB.Create(&earnRecord)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (svc *Service) addSwapRecord(userId uint, foodAmount float64, speakAmount float64) error {

	swapRecord := model.SwapRecord{
		UserID:      userId,
		FoodAmount:  foodAmount,
		SpeakAmount: speakAmount,
	}

	result := svc.db.DB.Create(&swapRecord)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (svc *Service) addWithdrawRecord(userId uint, speakAmount float64) error {

	withdrawRecord := model.WithdrawRecord{
		UserID:           userId,
		Amount:           speakAmount,
		State:            0, //0pending,1success,2fail
		Hash:             "",
		HandleTimestamp:  nil,
		ConfirmTimestamp: nil,
	}

	result := svc.db.DB.Create(&withdrawRecord)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (svc *Service) CheckSessionID(session_id string) error {
	uuidRecord := model.EarnRecord{}
	result := svc.db.DB.Where("session_id = ?", session_id).Find(&uuidRecord)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected >= 1 {
		return custom_errors.SESSION_ID_EXIST_ERROR
	}
	return nil
}

func (svc *Service) checkPlayer(sub string) error {
	var player model.Player
	result := svc.db.DB.Where("sub = ?", sub).Find(&player)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected >= 1 {

		//exist
		return nil
	}
	//not exist
	return custom_errors.PLAYER_NOT_EXIST_ERROR
}

func (svc *Service) GetMonthWithdraw(sub string) (float64, error) {
	var err error
	var player model.Player
	player, err = svc.getPlayerBySub(sub)
	if err != nil {
		return 0, custom_errors.PLAYER_NOT_EXIST_ERROR
	}
	monthBeginning := now.BeginningOfMonth()
	var withdrawRecords []model.WithdrawRecord
	err = svc.db.DB.Where("user_id = ?", player.UserId).
		Where("created_at > ?", monthBeginning).
		Where("state <> 3").
		Find(&withdrawRecords).Error
	if err != nil {
		return 0, err
	}
	if len(withdrawRecords) == 0 {
		return 0, nil
	}
	sum := 0.00
	for _, record := range withdrawRecords {
		sum += record.Amount

	}
	return sum, nil
}

func (svc *Service) GetEarnRecord(sub string) ([]model.EarnRecord, error) {
	var player model.Player
	err := svc.checkPlayer(sub)
	if err != nil {
		return nil, custom_errors.PLAYER_NOT_EXIST_ERROR
	}
	player, err = svc.getPlayerBySub(sub)
	if err != nil {
		return nil, err
	}

	var EarnRecords []model.EarnRecord
	err = svc.db.DB.Where("user_id = ?", player.UserId).Find(&EarnRecords).Error
	if err != nil {
		return nil, err
	}
	return EarnRecords, nil
}
func (svc *Service) GetSwapRecord(sub string) ([]model.SwapRecord, error) {
	var player model.Player
	err := svc.checkPlayer(sub)
	if err != nil {
		return nil, custom_errors.PLAYER_NOT_EXIST_ERROR
	}
	player, err = svc.getPlayerBySub(sub)
	if err != nil {
		return nil, err
	}
	var SwapRecords []model.SwapRecord
	err = svc.db.DB.Where("user_id = ?", player.UserId).Find(&SwapRecords).Error
	if err != nil {
		return nil, err
	}
	return SwapRecords, nil
}
func (svc *Service) GetWithdrawRecord(sub string) ([]model.WithdrawRecord, error) {
	var player model.Player
	err := svc.checkPlayer(sub)
	if err != nil {
		return nil, custom_errors.PLAYER_NOT_EXIST_ERROR
	}
	player, err = svc.getPlayerBySub(sub)
	if err != nil {
		return nil, err
	}
	var withdrawRecord []model.WithdrawRecord
	err = svc.db.DB.Where("user_id = ?", player.UserId).Find(&withdrawRecord).Error
	if err != nil {
		return nil, err
	}
	return withdrawRecord, nil
}
func (svc *Service) GetEarnTotal(sub string) (float64, error) {
	var player model.Player
	err := svc.checkPlayer(sub)
	if err != nil {
		return 0, custom_errors.PLAYER_NOT_EXIST_ERROR
	}
	player, err = svc.getPlayerBySub(sub)
	if err != nil {
		return 0, err
	}
	var earnTotal model.EarnTotal
	err = svc.db.DB.Where("user_id = ?", player.UserId).First(&earnTotal).Error
	if err != nil {
		return 0, err
	}
	return earnTotal.EarnTotal, nil
}
func (svc *Service) GetSwapTotal(sub string) (float64, float64, error) {
	var player model.Player
	err := svc.checkPlayer(sub)
	if err != nil {
		return 0, 0, custom_errors.PLAYER_NOT_EXIST_ERROR
	}
	player, err = svc.getPlayerBySub(sub)
	if err != nil {
		return 0, 0, err
	}

	return svc.getSwapTotalByUserid(player.UserId)
}
func (svc *Service) GetWithdrawTotal(sub string) (float64, error) {
	var player model.Player
	err := svc.checkPlayer(sub)
	if err != nil {
		return 0, custom_errors.PLAYER_NOT_EXIST_ERROR
	}
	player, err = svc.getPlayerBySub(sub)
	if err != nil {
		return 0, err
	}
	var withdrawTotal model.WithdrawTotal
	err = svc.db.DB.Where("user_id = ?", player.UserId).First(&withdrawTotal).Error
	if err != nil {
		return 0, err
	}
	return withdrawTotal.WithdrawTotal, nil
}

func (svc *Service) getSwapTotalByUserid(userid uint) (float64, float64, error) {
	var swapTotal model.SwapTotal
	err := svc.db.DB.Where("user_id = ?", userid).First(&swapTotal).Error
	if err != nil {
		return 0, 0, err
	}
	return swapTotal.SwappedFood, swapTotal.SwappedSpeak, nil
}

func (svc *Service) GetUserInfo(mail string, sub string) (*model.Player, error) {
	var player model.Player

	err := svc.checkPlayer(sub)
	if errors.Is(err, custom_errors.PLAYER_NOT_EXIST_ERROR) {
		err = svc.NewPlayer(mail, sub)
		if err != nil {
			return nil, err
		}
	}
	player, err = svc.getPlayerBySub(sub)
	if err != nil {
		return nil, err
	}
	return &player, nil
}

func (svc *Service) EditEthAddress(sub string, ethAddress string) error {
	var player model.Player
	player, err := svc.getPlayerBySub(sub)
	if err != nil {
		return err
	}
	err = svc.db.DB.Model(&player).Update("eth_address", ethAddress).Error
	if err != nil {
		if strings.HasPrefix(err.Error(), "Error 1062 (23000): Duplicate entry") {
			return custom_errors.ETH_ADDRESS_EXIST_ERROR
		}
		return err
	}
	return nil
}

type Data struct {
	Nfts       []NFT `json:"nfts"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"totalPages"`
	TotalItems int   `json:"totalItems"`
}

type NFTWithRecharge struct {
	model.NFT
	Balance    int64 `json:"balance"`
	Amount     int64 `json:"rechargeAmount"`
	ExpiryDate int64 `json:"expiryDate"`
}

type NFT struct {
	NFTWithRecharge
	Image      model.NFTImage     `json:"image"`
	Attributes []model.Attributes `json:"attributes"`
	Contract   model.Contract     `json:"contract"`
	Collection model.Collection   `json:"collection"`
}

func (svc *Service) GetUserNfts(sub string, page int, limit int) (*Data, error) {
	var player model.Player
	var result Data
	var nfts []NFT = make([]NFT, 0)
	var nftWithRecharges []NFTWithRecharge

	var count int64

	player, err := svc.getPlayerBySub(sub)
	if err != nil {
		return nil, err
	}
	if player.EthAddress == nil {
		return &Data{}, nil
	}

	queryNFTs := svc.db.DB.Table("nfts").
		Select("*").
		Joins("LEFT JOIN owners ON owners.token_id = nfts.token_id AND owners.token_type = nfts.token_type AND lower(owners.address) = lower(?)", *player.EthAddress).
		Joins("LEFT JOIN (SELECT token_id, MAX(created_at) AS latest_created_at FROM recharge_nfts WHERE payer = ? AND status = ? GROUP BY token_id) AS latest_recharge ON nfts.token_id = latest_recharge.token_id", *player.EthAddress, model.Confirmed).
		Joins("LEFT JOIN recharge_nfts ON nfts.token_id = recharge_nfts.token_id AND recharge_nfts.created_at = latest_recharge.latest_created_at").
		Where("lower(owners.address) = lower(?) AND owners.token_type= ?", *player.EthAddress, svc.conf.TokenType()).
		Or("lower(recharge_nfts.payer) = lower(?) AND nfts.token_type= ?", *player.EthAddress, svc.conf.TokenType()).
		Count(&count).
		Offset(int((page - 1) * limit)).
		Limit(limit).
		Find(&nftWithRecharges)
	if queryNFTs.Error != nil {
		return nil, err
	}

	for _, nft := range nftWithRecharges {
		var image model.NFTImage
		var contract model.Contract
		var collection model.Collection
		var attributes []model.Attributes

		query := svc.db.DB.Where("contract_id = ?", nft.ContractID).First(&contract)
		if query.Error != nil {
			return nil, err
		}
		query = svc.db.DB.Where("collection_id = ?", nft.CollectionID).First(&collection)
		if query.Error != nil {
			return nil, err
		}
		query = svc.db.DB.Where("token_id = ? and token_type = ?", nft.TokenId, svc.conf.TokenType()).First(&image)
		if query.Error != nil {
			return nil, err
		}
		query = svc.db.DB.Where("token_id = ? and token_type = ?", nft.TokenId, svc.conf.TokenType()).Find(&attributes)
		if query.Error != nil {
			return nil, err
		}
		var data = NFT{
			NFTWithRecharge: nft,
			Image:           image,
			Attributes:      attributes,
			Contract:        contract,
			Collection:      collection,
		}
		nfts = append(nfts, data)
	}
	result.Nfts = nfts
	result.Page = int(page)
	result.Limit = limit
	result.TotalItems = int(count)
	result.TotalPages = int(math.Ceil(float64(count) / float64(limit)))
	return &result, nil
}

func (svc *Service) CheckPaidPlayer(player model.Player) (err error) {
	if player.EthAddress == nil {
		return custom_errors.PLAYER_ETH_ADDRESS_EXIST_ERROR
	}
	var count int64
	queryNFTs := svc.db.DB.Table("nfts").
		// Joins("LEFT JOIN owners ON owners.token_id = nfts.token_id").
		Joins("LEFT JOIN recharge_nfts ON nfts.token_id = recharge_nfts.token_id AND recharge_nfts.payer = ? ", *player.EthAddress).
		// Where("lower(owners.address) = lower(?)", *player.EthAddress).
		Where("lower(recharge_nfts.payer) = lower(?) AND recharge_nfts.status = ? AND recharge_nfts.expiry_date > UNIX_TIMESTAMP(NOW())", *player.EthAddress, model.Confirmed).
		Count(&count)
	if queryNFTs.Error != nil {
		return queryNFTs.Error
	}
	if count <= 0 {
		return custom_errors.FREE_BIE_USER_ERROR
	}
	return nil
}

func (svc *Service) addFoodFreebieTotal(userId uint, amount float64) error {
	var foodTotal model.FreebieEarnTotal
	err := svc.db.DB.Where("user_id = ?", userId).Last(&foodTotal).Error
	if err != nil {
		foodTotal = model.FreebieEarnTotal{
			UserID:     userId,
			EarnTotal:  0,
			ExpiryDate: uint64(time.Now().Unix()) + uint64(svc.conf.NFTExpiryTime()),
		}
		svc.db.DB.Create(&foodTotal)
	}
	if amount < 0 {
		return custom_errors.AMOUNT_ERROR
	}

	if foodTotal.ExpiryDate >= uint64(time.Now().Unix()) {
		foodTotal.EarnTotal += amount
		err = svc.db.DB.Save(&foodTotal).Error
		if err != nil {
			svc.log.Error(err)
			return err
		}
	} else {
		foodTotal = model.FreebieEarnTotal{
			UserID:     userId,
			EarnTotal:  amount,
			ExpiryDate: uint64(time.Now().Unix()) + uint64(svc.conf.NFTExpiryTime()),
		}
		err = svc.db.DB.Create(&foodTotal).Error
		if err != nil {
			svc.log.Error(err)
			return err
		}
	}
	return nil
}

func (svc *Service) addFreebieRecord(userId uint, amount float64, sessionId string) error {

	freeBieRecord := model.FreeBieRecord{
		UserID:    userId,
		Amount:    amount,
		SessionID: sessionId,
	}
	result := svc.db.DB.Create(&freeBieRecord)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (svc *Service) EarnAllowFreebie(players []model.EarnPlayer, sessionId string) error {
	for _, player := range players {
		err := svc.checkPlayer(player.Sub)
		if err != nil {
			return custom_errors.PLAYER_NOT_EXIST_ERROR
		}
	}

	err := svc.db.DB.Transaction(func(tx *gorm.DB) error {
		for _, player := range players {
			temPlayer, err := svc.getPlayerBySub(player.Sub)
			if err != nil {
				return err
			}
			err = svc.CheckPaidPlayer(temPlayer)
			if err != nil {
				amount64 := float64(player.Amount)
				err = svc.addFoodFreebieTotal(temPlayer.UserId, amount64)
				if err != nil {
					return err
				}
				err = svc.addFreebieRecord(temPlayer.UserId, amount64, sessionId)
				if err != nil {
					return err
				}
			} else {
				amount := player.Amount * player.Rarity
				amount64 := float64(amount)
				err = svc.addFoodTotal(temPlayer.UserId, amount64)
				if err != nil {
					return err
				}
				err = svc.addEarnRecord(temPlayer.UserId, amount64, sessionId)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		svc.log.Error("Failed to earn", sessionId)
		return err
	}
	return nil
}

func (svc *Service) getFreebieEarnTotal(userId uint) float64 {
	var foodTotals []model.FreebieEarnTotal
	err := svc.db.DB.Where("user_id =? AND charge_date > 0", userId).Find(&foodTotals).Error
	if err != nil {
		svc.log.Error(err)
		return 0
	}
	total := float64(0)
	for _, foodTotal := range foodTotals {
		total += foodTotal.EarnTotal
	}
	return total
}

func (svc *Service) GetFreebieRecord(sub string) ([]model.FreeBieRecord, error) {
	var player model.Player
	err := svc.checkPlayer(sub)
	if err != nil {
		return nil, custom_errors.PLAYER_NOT_EXIST_ERROR
	}
	player, err = svc.getPlayerBySub(sub)
	if err != nil {
		return nil, err
	}

	var freeBieRecords []model.FreeBieRecord
	err = svc.db.DB.Where("user_id = ?", player.UserId).Find(&freeBieRecords).Error
	if err != nil {
		return nil, err
	}
	return freeBieRecords, nil
}
