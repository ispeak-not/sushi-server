package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	config configData
}

type configData struct {
	// http
	HTTPPort int    `mapstructure:"http_port"`
	GinMode  string `mapstructure:"gin_mode"`

	// log
	LogLevel        logrus.Level `mapstructure:"log_level"`
	LogFileLocation string       `mapstructure:"log_file_location"`

	// database
	DBType           string `mapstructure:"db_type"`
	DBConnectionPath string `mapstructure:"db_path"`

	// worker
	APIKey                 string `mapstructure:"api_key"`
	NFTContractAddress     string `mapstructure:"nft_contract_address"`
	TokenType              string `mapstructure:"token_type"`
	WorkerPort             int    `mapstructure:"worker_port"`
	SyncBlockNumber        uint64 `mapstructure:"sync_block_number"`
	Network                string `mapstructure:"network"`
	PaymentContractAddress string `mapstructure:"payment_contract_address"`
	SpecSchedule           string `mapstructure:"spec_schedule"`

	// nft expiry
	NFTExpiryTime int `mapstructure:"nft_expiry_time"`
	//TxProcessorConfig TxProcessorConfig `mapstructure:"tx_processor_config"`
}

func NewConfig() (*Config, error) {

	viper.SetConfigName("config") // name of config.yaml file (without extension)
	viper.SetConfigType("yaml")

	// where to look for
	//viper.AddConfigPath("/etc/bitspawn/api/") // production config.yaml path
	//viper.AddConfigPath("./config.yaml")           // dev config.yaml path
	//viper.AddConfigPath("../config.yaml")          // dev config.yaml path
	viper.AddConfigPath(".")
	err := viper.ReadInConfig() // Find and read the config.yaml file
	if err != nil {             // Handle errors reading the config.yaml file
		return nil, err
	}

	// Override config.yaml file based on ENV variables e.g. DB_TYPE=postgres
	viper.AutomaticEnv()

	config := configData{}
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return &Config{config: config}, nil
}

func (c *Config) HTTPPort() int {
	return c.config.HTTPPort
}

func (c *Config) DBConnectionPath() string {
	return c.config.DBConnectionPath
}

func (c *Config) GinMode() string {
	return c.config.GinMode
}

func (c *Config) LogFileLocation() string {
	return c.config.LogFileLocation
}

func (c *Config) APIKey() string {
	return c.config.APIKey
}

func (c *Config) NFTContractAddress() string {
	return c.config.NFTContractAddress
}

func (c *Config) PaymentContractAddress() string {
	return c.config.PaymentContractAddress
}

func (c *Config) Network() string {
	return c.config.Network
}

func (c *Config) WorkerPort() int {
	return c.config.WorkerPort
}

func (c *Config) NFTExpiryTime() int {
	if c.config.NFTExpiryTime == 0 {
		return 10 * 24 * 60 * 60 //10 days (unix)
	}
	return c.config.NFTExpiryTime
}

func (c *Config) SpecSchedule() string {
	if c.config.SpecSchedule == "" {
		return "0 * * * *" // At minute 0 every hour
	}
	return c.config.SpecSchedule
}

func (c *Config) SyncBlockNumber() uint64 {
	return c.config.SyncBlockNumber
}

func (c *Config) TokenType() string {
	if c.config.TokenType != "ERC721" && c.config.TokenType != "ERC1155" {
		return "ERC721" // default ERC721
	}
	return c.config.TokenType
}

func (c *Config) LogLevel() logrus.Level {
	return c.config.LogLevel
}

//
//func (c *config) DBCredentials() []string {
//	return []string{
//		c.config.yaml.DBType,
//		c.config.yaml.DBConnectionPath,
//	}
//}
