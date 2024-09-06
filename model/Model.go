package model

import (
	"time"
)

type Player struct {
	UserId     uint    `gorm:"primaryKey" json:"-"`
	Tier       uint    `json:"tier"`
	Region     uint    `json:"region"`
	EthAddress *string `gorm:"unique" json:"eth_address"`
	//Blobs     []Blob `json:"blobs"`
	Mail      string    `gorm:"index" json:"mail"`
	Sub       string    `gorm:"index" json:"sub"`
	CreatedAt time.Time `json:"created_at"`
	LoginAt   time.Time `json:"login_at"`
}

//type Balance struct {
//	UserId uint `gorm:"primaryKey"`
//	Food   float64
//	Speak  float64
//}

type SwapTotal struct {
	UserID       uint `gorm:"primaryKey"`
	SwappedFood  float64
	SwappedSpeak float64
}

type EarnTotal struct {
	UserID    uint `gorm:"primaryKey"`
	EarnTotal float64
}

type WithdrawTotal struct {
	UserID        uint `gorm:"primaryKey"`
	WithdrawTotal float64
}

type EarnRecord struct {
	EarnId    uint      `gorm:"primaryKey" json:"earn_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	SessionID string    `json:"session_id"`
	Amount    float64   `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}
type EarnPlayer struct {
	Sub    string `json:"sub"`
	Amount uint   `json:"amount"`
	Rarity uint   `json:"rarity"`
}
type WithdrawRecord struct {
	WithdrawId       uint `gorm:"primaryKey"`
	UserID           uint `gorm:"index"`
	Amount           float64
	CreatedAt        time.Time
	Address          string
	State            uint
	HandleTimestamp  *time.Time
	ConfirmTimestamp *time.Time
	Hash             string
}
type SwapRecord struct {
	SwapId      uint `gorm:"primaryKey"`
	UserID      uint `gorm:"index" `
	FoodAmount  float64
	SpeakAmount float64
	CreatedAt   time.Time
}

type Blob struct {
	BlobID       uint64 `json:"blob_id"`
	BlobName     string `json:"blob_name"`
	BlobUniqueID uint64 `json:"blob_unique_id"`
}

type Contract struct {
	ContractID          uint64 `gorm:"primaryKey" json:"-"`
	Address             string `json:"address"`
	Name                string `json:"name"`
	Symbol              string `json:"symbol"`
	TotalSupply         uint64 `json:"totalSupply"`
	TokenType           string `json:"tokenType"`
	ContractDeployer    string `json:"contractDeployer"`
	DeployedBlockNumber uint64 `json:"deployedBlockNumber"`
}

type Collection struct {
	CollectionID   uint64 `gorm:"primaryKey" json:"-"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	ExternalUrl    string `json:"externalUrl"`
	BannerImageUrl string `json:"bannerImageUrl"`
}

type NFT struct {
	NftID           uint64 `gorm:"primaryKey" json:"-"`
	ContractID      uint64 `gorm:"index" json:"-"`
	TokenId         string `json:"tokenId"`
	TokenType       string `json:"tokenType"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	TokenUri        string `json:"tokenUri"`
	CollectionID    uint64 `json:"-"`
	Balance         string `json:"balance"`
	TimeLastUpdated string `json:"timeLastUpdated"`
}

type NFTImage struct {
	TokenId      string `gorm:"index" json:"-"`
	CachedUrl    string `json:"cachedUrl"`
	ThumbnailUrl string `json:"thumbnailUrl"`
	PngUrl       string `json:"pngUrl"`
	ContentType  string `json:"contentType"`
	Size         uint64 `json:"size"`
	OriginalUrl  string `json:"originalUrl"`
}

type Owner struct {
	Address string `json:"address"`
	TokenId string `json:"tokenId"`
}

type Attributes struct {
	TokenId string `json:"-"`
	Type    string `json:"Type"`
	Rarity  string `json:"Rarity"`
}

func GetNameFromBlobID(id uint64) string {
	BlobMap := make(map[uint64]string)
	BlobMap[0] = "Tuna Roll"
	BlobMap[1] = "Natto Roll"
	BlobMap[2] = "Salmon Sushi"
	BlobMap[3] = "Shrimp Sushi"
	return BlobMap[id]
}
func GetBlobIDFromBlobUniqueID(uniqueId uint64) uint64 {
	return uniqueId / 10000 % 4
}
func GetBlobFromUniqueID(uniqueId uint64) Blob {
	blobId := GetBlobIDFromBlobUniqueID(uniqueId)
	return Blob{
		BlobID:       blobId,
		BlobName:     GetNameFromBlobID(blobId),
		BlobUniqueID: uniqueId,
	}
}
func GetBlobsFromUniqueIDs(ids []uint64) []Blob {
	var blobs []Blob
	for _, v := range ids {
		blobs = append(blobs, GetBlobFromUniqueID(v))
	}
	return blobs
}

type PlayerInfo struct {
	Tier   uint `json:"tier"`
	Region uint `json:"region"`
	//Blobs  []Blob `json:"blobs"`
	Mail string `json:"mail"`
	Sub  string `json:"sub"`
}

type LatestBlock struct {
	CrawlKey          string `gorm:"unique"`
	LatestBlockNumber uint64
}

type Network struct {
	ChainID         int64
	ContractAddress string `gorm:"unique"`
	Name            string
	Decimals        int64
	Symbol          string
	RpcUrl          string
	ABI             string
}

type ConfirmStatus string

const (
	Confirming ConfirmStatus = "confirming"
	Confirmed  ConfirmStatus = "confirmed"
)

type RechargeNFT struct {
	Payer        string
	Received     string
	TokenAddress string
	TokenID      string
	ExpiryDate   uint64
	Amount       uint64
	Status       ConfirmStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
