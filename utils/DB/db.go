package DB

import (
	"fmt"
	"sushi/model"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type DB struct {
	DB  *gorm.DB
	log *logrus.Logger
}

func NewDB_MySQL(log *logrus.Logger, DBPath string) *DB {
	dsn := fmt.Sprintf(DBPath)
	var _db *gorm.DB
	var err error

	_db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("connection error: " + err.Error())
	}

	sqlDB, _ := _db.DB()
	err = _db.AutoMigrate(model.Player{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.SwapTotal{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.WithdrawTotal{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.EarnTotal{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.EarnRecord{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.SwapRecord{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.WithdrawRecord{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.Contract{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.Collection{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.NFTImage{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.NFT{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.Owner{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.Attributes{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.LatestBlock{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.Network{})
	if err != nil {
		return nil
	}
	err = _db.AutoMigrate(model.RechargeNFT{})
	if err != nil {
		return nil
	}

	sqlDB.SetMaxOpenConns(100) //连接池最大连接数
	sqlDB.SetMaxIdleConns(20)  //最大允许的空闲连接数
	return &DB{
		log: log,
		DB:  _db,
	}
}
