package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
	"sushi/model"
	"sushi/utils/DB"
	"sushi/utils/config"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Handler struct {
	db   *DB.DB
	log  *logrus.Logger
	conf *config.Config
	Ctx  *context.Context
}

type GetOwnersForContractResponse struct {
	Owners  []OwnerResponse `json:"owners"`
	PageKey *string         `json:"pageKey"`
}

type OwnerResponse struct {
	OwnerAddress  string         `json:"ownerAddress"`
	TokenBalances []TokenBalance `json:"tokenBalances"`
}

type TokenBalance struct {
	TokenId string `json:"tokenId"`
	Balance string `json:"balance"`
}

type NftsResponse struct {
	OwnedNfts  []NFTMetaData `json:"ownedNfts"`
	PageKey    *string       `json:"pageKey"`
	TotalCount int           `json:"totalCount"`
}
type NFTMetaData struct {
	Contract        model.Contract   `json:"contract"`
	TokenId         string           `json:"tokenId"`
	TokenType       string           `json:"tokenType"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	TokenUri        string           `json:"tokenUri"`
	Image           model.NFTImage   `json:"image"`
	Collection      model.Collection `json:"collection"`
	TimeLastUpdated string           `json:"timeLastUpdated"`
	Balance         string           `json:"balance"`
	Raw             Raw              `json:"raw"`
}

type Raw struct {
	TokenUri string   `json:"tokenUri"`
	Metadata Metadata `json:"metadata"`
}
type Metadata struct {
	Name        string             `json:"name"`
	Image       string             `json:"image"`
	Description string             `json:"description"`
	ExternalUrl string             `json:"external_url"`
	Attributes  []model.Attributes `json:"attributes"`
}

const AVG_BLOCK_CONFIRM = 5
const AVG_BLOCK_TIME = 2

func NewHandler(worker *Worker) *Handler {
	ctx, _ := context.WithCancel(context.Background())
	return &Handler{
		log:  worker.log,
		conf: worker.config,
		db:   worker.db,
		Ctx:  &ctx,
	}
}
func (handler *Handler) HandleLog() {
	fmt.Printf("welcome %s", handler.conf.APIKey())
}

func (handler *Handler) GetOwnersForContract() {
	fmt.Println("Get Owner For Contract Job Started")
	var pageKey *string
	res := handler.db.DB.Where("1 = 1").Delete(&model.Owner{})
	if res.Error != nil {
		handler.log.Fatalf("Failed to delete old owner: %v", res.Error)
	}
	for {
		params := ""
		if pageKey != nil {
			params = fmt.Sprintf("&pageKey=%s", *pageKey)
		}

		url := fmt.Sprintf("https://%s.g.alchemy.com/nft/v3/%s/getOwnersForContract?contractAddress=%s&withTokenBalances=true%s", handler.conf.Network(), handler.conf.APIKey(), handler.conf.NFTContractAddress(), params)
		resp, err := http.Get(url)
		if err != nil {
			handler.log.Fatalf("Failed to make a request: %v", err)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			handler.log.Fatalf("Failed to read the response body: %v", err)
		}

		var result GetOwnersForContractResponse
		err = json.Unmarshal([]byte(body), &result)
		if err != nil {
			handler.log.Fatalf("Failed to json unmarshal: %v", err)
		}
		if result.Owners != nil {
			for _, v := range result.Owners {
				err := handler.createOwner(v)
				if err != nil {
					continue
				}
			}
		}
		if result.PageKey != nil {
			pageKey = result.PageKey
		} else {
			break
		}
		time.Sleep(10 * time.Second)
	}
	fmt.Println("Get Owner For Contract Job Done")
	handler.getNFTsForOwners()
}

func (handler *Handler) getNFTsForOwners() {
	fmt.Println("Get NFTs For Owners Job Started")

	owners, err := handler.findAllOwners()
	if err != nil {
		handler.log.Fatalf("Failed to get Owners: %v", err)
	}
	for _, owner := range *owners {

		var pageKey *string
		for {
			params := ""
			if pageKey != nil {
				params = fmt.Sprintf("&pageKey=%s", *pageKey)
			}

			url := fmt.Sprintf("https://%s.g.alchemy.com/nft/v3/%s/getNFTsForOwner?owner=%s&contractAddresses[]=%s&withMetadata=true&pageSize=100%s", handler.conf.Network(), handler.conf.APIKey(), owner.Address, handler.conf.NFTContractAddress(), params)
			resp, err := http.Get(url)
			if err != nil {
				handler.log.Fatalf("Failed to make a request: %v", err)
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				handler.log.Fatalf("Failed to read the response body: %v", err)
			}

			var result NftsResponse
			err = json.Unmarshal([]byte(body), &result)
			if err != nil {
				handler.log.Fatalf("Failed to json unmarshal: %v", err)
			}
			if result.OwnedNfts != nil {
				for _, v := range result.OwnedNfts {
					err := handler.updateOrCreateNFT(&v)
					if err != nil {
						continue
					}
				}
			}
			if result.PageKey != nil {
				pageKey = result.PageKey
			} else {
				break
			}
			time.Sleep(10 * time.Second)
		}
	}
	fmt.Println("Get NFTs For Owners Job Done")
}

func (handler *Handler) findAllOwners() (*[]model.Owner, error) {
	var owners []model.Owner
	err := handler.db.DB.Distinct("address").Select("address").Group("address").Find(&owners).Error
	if err != nil {
		return nil, err
	}
	return &owners, nil
}

func (handler *Handler) findOrCreateContract(contract *model.Contract) (*model.Contract, error) {

	result := handler.db.DB.Where("address = ?", contract.Address).Assign(model.Contract{Address: contract.Address}).FirstOrCreate(&contract)

	if result.Error != nil {
		return nil, result.Error
	}
	return contract, nil
}

func (handler *Handler) findOrCreateCollection(collection *model.Collection) (*model.Collection, error) {

	result := handler.db.DB.Where(model.Collection{Slug: collection.Slug}).Assign(model.Collection{Slug: collection.Slug}).FirstOrCreate(&collection)

	if result.Error != nil {
		return nil, result.Error
	}
	return collection, nil
}

func (handler *Handler) createOrUpdateNFT(nftMeta *NFTMetaData, contract *model.Contract, collection *model.Collection) error {

	var nft model.NFT
	result := handler.db.DB.Where(model.NFT{TokenId: nftMeta.TokenId}).First(&nft)

	nft = model.NFT{
		ContractID:      contract.ContractID,
		TokenId:         nftMeta.TokenId,
		TokenType:       nftMeta.TokenType,
		Name:            nftMeta.Name,
		Description:     nftMeta.Description,
		TokenUri:        nftMeta.TokenUri,
		CollectionID:    collection.CollectionID,
		TimeLastUpdated: nftMeta.TimeLastUpdated,
		Balance:         nftMeta.Balance,
	}

	if result.Error != nil {
		result := handler.db.DB.Create(&nft)
		if result.Error != nil {
			return result.Error
		}
		err := handler.createOrUpdateNftImage(nft.TokenId, nftMeta.Image)
		if err != nil {
			return err
		}
		err = handler.createAttributes(nft.TokenId, nftMeta.Raw.Metadata.Attributes)
		if err != nil {
			return err
		}
		return nil
	}

	if result.RowsAffected >= 1 {
		//nft exits
		result := handler.db.DB.Where(model.NFT{TokenId: nft.TokenId}).Updates(nft)
		if result.Error != nil {
			return result.Error
		}
		err := handler.createOrUpdateNftImage(nft.TokenId, nftMeta.Image)
		if err != nil {
			return err
		}
		err = handler.createAttributes(nft.TokenId, nftMeta.Raw.Metadata.Attributes)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (handler *Handler) createOrUpdateNftImage(tokenId string, nftImage model.NFTImage) error {
	var image model.NFTImage
	result := handler.db.DB.Where(model.NFTImage{TokenId: tokenId}).First(&image)

	image = model.NFTImage{
		TokenId:      tokenId,
		CachedUrl:    nftImage.CachedUrl,
		ThumbnailUrl: nftImage.ThumbnailUrl,
		PngUrl:       nftImage.PngUrl,
		ContentType:  nftImage.ContentType,
		Size:         nftImage.Size,
		OriginalUrl:  nftImage.OriginalUrl,
	}

	if result.Error != nil {
		result := handler.db.DB.Create(&image)
		if result.Error != nil {
			return result.Error
		}
		return nil
	}

	if result.RowsAffected >= 1 {
		//nft exits
		result := handler.db.DB.Where(model.NFTImage{TokenId: image.TokenId}).Updates(&image)
		if result.Error != nil {
			return result.Error
		}
		return nil
	}

	return nil
}

func (handler *Handler) createAttributes(tokenId string, attributes []model.Attributes) error {
	handler.db.DB.Where(model.Attributes{TokenId: tokenId}).Delete(&model.Attributes{})

	for _, v := range attributes {
		var attribute = model.Attributes{
			Type:    v.Type,
			Rarity:  v.Rarity,
			TokenId: tokenId,
		}
		result := handler.db.DB.Create(&attribute)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func (handler *Handler) updateOrCreateNFT(nftMetaData *NFTMetaData) error {

	er := handler.db.DB.Transaction(func(tx *gorm.DB) error {

		contract, err := handler.findOrCreateContract(&nftMetaData.Contract)
		if err != nil {
			return err
		}

		collection, err := handler.findOrCreateCollection(&nftMetaData.Collection)
		if err != nil {
			return err
		}

		err = handler.createOrUpdateNFT(nftMetaData, contract, collection)
		if err != nil {
			return err
		}

		return nil
	})
	if er != nil {
		handler.log.Error("Failed to update or create NFT: ", nftMetaData.TokenId)
		return er
	}
	handler.log.Info("New NFT:", nftMetaData.TokenId)
	return nil
}

func (handler *Handler) createOwner(owner OwnerResponse) error {
	var owners []model.Owner

	for _, v := range owner.TokenBalances {
		owners = append(owners, model.Owner{
			Address: owner.OwnerAddress,
			TokenId: v.TokenId,
		})
	}

	result := handler.db.DB.Create(&owners)

	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (handler *Handler) CrawlFromWeb3() {
	go func() {
		network, err := handler.getNetwork()
		if err != nil {
			handler.log.Printf("Failed to get network from database: %v", err)
		}

		client, err := ethclient.Dial(network.RpcUrl)
		if err != nil {
			handler.log.Printf("Failed to connect to the %s client: %v", network.Name, err)
		}
		defer client.Close()

		handler.listenPastEvents(client)
		handler.subscribeRealTimeEvents(client)
	}()
}

func (handler *Handler) createRecharge(payer string, received string, tokenAddress string, tokenId string, expiryDate uint64, amount uint64, status model.ConfirmStatus) error {
	recharge := model.RechargeNFT{
		Payer:        payer,
		Received:     received,
		TokenAddress: tokenAddress,
		TokenID:      tokenId,
		ExpiryDate:   expiryDate,
		Amount:       amount,
		Status:       status,
	}

	var exist model.RechargeNFT

	result := handler.db.DB.Where(model.RechargeNFT{Payer: payer, TokenID: tokenId}).Last(&exist)
	if result.Error != nil {
		result = handler.db.DB.Create(&recharge)
		if result.Error != nil {
			return result.Error
		}
	} else if status != model.Confirming {
		if exist.ExpiryDate < uint64(time.Now().Unix()) {
			exist.ExpiryDate = expiryDate
		}
		exist.Amount = amount
		exist.TokenAddress = tokenAddress
		exist.Status = status
		result := handler.db.DB.Where(model.RechargeNFT{Payer: payer, TokenID: tokenId}).Updates(&exist)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func (handler *Handler) updateLatestBlock(latestBlock uint64, block *model.LatestBlock) error {
	var latestBlockInDB model.LatestBlock

	result := handler.db.DB.Where(model.LatestBlock{CrawlKey: block.CrawlKey}).First(&latestBlockInDB)
	latestBlockInDB = model.LatestBlock{
		CrawlKey:          block.CrawlKey,
		LatestBlockNumber: latestBlock,
	}

	if result.Error != nil {
		result := handler.db.DB.Create(&latestBlockInDB)
		if result.Error != nil {
			return result.Error
		}
	} else if result.RowsAffected >= 1 {
		result := handler.db.DB.Where(model.LatestBlock{CrawlKey: block.CrawlKey}).Updates(&latestBlockInDB)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func (handler *Handler) getLatestBlock(isTemp bool) (*model.LatestBlock, error) {
	var key string
	if isTemp {
		key = fmt.Sprintf("crawl_temp_%s", handler.conf.PaymentContractAddress())
	} else {
		key = fmt.Sprintf("crawl_polygon_%s", handler.conf.PaymentContractAddress())
	}

	latestBlock := model.LatestBlock{
		CrawlKey:          key,
		LatestBlockNumber: handler.conf.SyncBlockNumber(),
	}
	result := handler.db.DB.Where(model.LatestBlock{CrawlKey: key}).FirstOrCreate(&latestBlock)
	if result.Error != nil {
		return nil, result.Error

	}
	return &latestBlock, nil
}

func (handler *Handler) getNetwork() (*model.Network, error) {
	var network model.Network

	result := handler.db.DB.Where(model.Network{ContractAddress: handler.conf.PaymentContractAddress()}).First(&network)
	if result.Error != nil {
		return nil, result.Error

	}
	return &network, nil
}

func (handler *Handler) getTimeStamp(blocknumber int64, client *ethclient.Client) uint64 {
	block, err := client.BlockByNumber(*handler.Ctx, big.NewInt(blocknumber))
	if err != nil {
		handler.log.Printf("Failed to fetch block: %v", err)
		return 0
	}
	return block.Time()
}

func (handler *Handler) listenPastEvents(client *ethclient.Client) {
	latestBlock, err := handler.getLatestBlock(false)
	if err != nil {
		handler.log.Printf("Failed to get contract from database: %v", err)
	}

	network, err := handler.getNetwork()
	if err != nil {
		handler.log.Printf("Failed to get network from database: %v", err)
	}

	var fromBlock = latestBlock.LatestBlockNumber + 1
	var toBlock = fromBlock + 10000

	latestBlockNumber, err := client.BlockNumber(*handler.Ctx)
	if err != nil {
		handler.log.Fatal(err)
	}

	for ok := true; ok; ok = toBlock < latestBlockNumber-AVG_BLOCK_CONFIRM {
		query := ethereum.FilterQuery{
			Addresses: []common.Address{common.HexToAddress(handler.conf.PaymentContractAddress())},
			FromBlock: big.NewInt(int64(fromBlock)),
			ToBlock:   big.NewInt(int64(toBlock)),
		}

		logs, err := client.FilterLogs(*handler.Ctx, query)
		if err != nil {
			handler.log.Fatal(err)
		}
		for _, log := range logs {
			handler.handleLog(log, client, network.ABI, latestBlock, model.Confirmed)
		}
		fromBlock = toBlock + 1
		toBlock += 10000
	}
	err = handler.updateLatestBlock(latestBlockNumber, latestBlock)
	if err != nil {
		handler.log.Println("error to update Latest Block")
	}
}

func (handler *Handler) subscribeRealTimeEvents(client *ethclient.Client) {
	latestBlock, err := handler.getLatestBlock(true)
	if err != nil {
		handler.log.Printf("Failed to get contract from database: %v", err)
	}

	network, err := handler.getNetwork()
	if err != nil {
		handler.log.Printf("Failed to get network from database: %v", err)
	}

	var fromBlock = latestBlock.LatestBlockNumber + 1

	query := ethereum.FilterQuery{
		Addresses: []common.Address{common.HexToAddress(handler.conf.PaymentContractAddress())},
		FromBlock: big.NewInt(int64(fromBlock)),
	}

	logs := make(chan types.Log)
	sub := event.Resubscribe(2*time.Second, func(ctx context.Context) (event.Subscription, error) {
		return client.SubscribeFilterLogs(ctx, query, logs)
	})
	defer sub.Unsubscribe()

	for {
		select {
		case err := <-sub.Err():
			handler.log.Fatal(err)
		case log := <-logs:
			fmt.Printf("Received log %s \n", log.BlockHash)
			handler.handleLog(log, client, network.ABI, latestBlock, model.Confirming)
			go func() {
				time.Sleep(AVG_BLOCK_CONFIRM * AVG_BLOCK_TIME * time.Second)
				handler.listenPastEvents(client)
			}()
		}
	}
}

func (handler *Handler) handleLog(log types.Log, client *ethclient.Client, ABI string, latestBlock *model.LatestBlock, status model.ConfirmStatus) {
	event := struct {
		Payer        common.Address
		Receiver     common.Address
		TokenAddress common.Address
		NftId        *big.Int
		Amount       *big.Int
	}{}
	contractAbi, err := abi.JSON(strings.NewReader(ABI))
	if err != nil {
		handler.log.Printf("Failed to parse contract ABI: %v \n", err)
	}

	err = contractAbi.UnpackIntoInterface(&event, "PaymentReceived", log.Data)
	if err == nil {
		if len(log.Topics) > 0 {
			event.Payer = common.HexToAddress(log.Topics[1].Hex())
		}

		handler.log.Println("Event:", event)

		expiryDate := handler.getTimeStamp(int64(log.BlockNumber), client) + uint64(handler.conf.NFTExpiryTime())

		err = handler.createRecharge(event.Payer.Hex(), event.Receiver.Hex(), event.TokenAddress.Hex(), event.NftId.String(), expiryDate, event.Amount.Uint64(), status)
		if err != nil {
			handler.log.Fatalf(err.Error())
		}
	}

	err = handler.updateLatestBlock(log.BlockNumber, latestBlock)
	if err != nil {
		handler.log.Println("error to update Latest Block")
	}
}
