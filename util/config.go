package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	ethcom "github.com/ethereum/go-ethereum/common"

	"github.com/synycboom/bsc-evm-compatible-bridge-core/common"
)

type Config struct {
	KeyManagerConfig KeyManagerConfig `json:"key_manager_config"`
	DBConfig         DBConfig         `json:"db_config"`
	ChainConfig      ChainConfig      `json:"chain_config"`
	LogConfig        LogConfig        `json:"log_config"`
	AlertConfig      AlertConfig      `json:"alert_config"`
	AdminConfig      AdminConfig      `json:"admin_config"`
}

func (cfg *Config) Validate() {
	cfg.DBConfig.Validate()
	cfg.ChainConfig.Validate()
	cfg.LogConfig.Validate()
	cfg.AlertConfig.Validate()
}

type AlertConfig struct {
	TelegramBotId  string `json:"telegram_bot_id"`
	TelegramChatId string `json:"telegram_chat_id"`

	BlockUpdateTimeout int64 `json:"block_update_timeout"`
}

func (cfg AlertConfig) Validate() {
	if cfg.BlockUpdateTimeout <= 0 {
		panic("block_update_timeout should be larger than 0")
	}
}

type KeyManagerConfig struct {
	KeyType       string `json:"key_type"`
	AWSRegion     string `json:"aws_region"`
	AWSSecretName string `json:"aws_secret_name"`

	// local keys
	LocalHMACKey                    string `json:"local_hmac_key"`
	LocalSourceChainPrivateKey      string `json:"local_source_chain_private_key"`
	LocalDestinationChainPrivateKey string `json:"local_destination_chain_private_key"`
	LocalAdminApiKey                string `json:"local_admin_api_key"`
	LocalAdminSecretKey             string `json:"local_admin_secret_key"`
}

type KeyConfig struct {
	HMACKey                    string `json:"hmac_key"`
	SourceChainPrivateKey      string `json:"source_chain_private_key"`
	DestinationChainPrivateKey string `json:"destination_chain_private_key"`
	AdminApiKey                string `json:"admin_api_key"`
	AdminSecretKey             string `json:"admin_secret_key"`
}

func (cfg KeyManagerConfig) Validate() {
	if cfg.KeyType == common.LocalPrivateKey && len(cfg.LocalHMACKey) == 0 {
		panic("missing local hmac key")
	}
	if cfg.KeyType == common.LocalPrivateKey && len(cfg.LocalSourceChainPrivateKey) == 0 {
		panic("missing local source chain private key")
	}
	if cfg.KeyType == common.LocalPrivateKey && len(cfg.LocalDestinationChainPrivateKey) == 0 {
		panic("missing local destination chain private key")
	}

	if cfg.KeyType == common.LocalPrivateKey && len(cfg.LocalAdminApiKey) == 0 {
		panic("missing local admin api key")
	}

	if cfg.KeyType == common.LocalPrivateKey && len(cfg.LocalAdminSecretKey) == 0 {
		panic("missing local admin secret key")
	}

	if cfg.KeyType == common.AWSPrivateKey && (cfg.AWSRegion == "" || cfg.AWSSecretName == "") {
		panic("Missing aws key region or name")
	}
}

type TokenSecretKey struct {
	Symbol                     string `json:"symbol"`
	SourceChainPrivateKey      string `json:"source_chain_private_key"`
	DestinationChainPrivateKey string `json:"destination_private_key"`
}

type DBConfig struct {
	Dialect string `json:"dialect"`
	DBPath  string `json:"db_path"`
}

func (cfg DBConfig) Validate() {
	if cfg.Dialect != common.DBDialectMysql && cfg.Dialect != common.DBDialectSqlite3 {
		panic(fmt.Sprintf("only %s and %s supported", common.DBDialectMysql, common.DBDialectSqlite3))
	}
	if cfg.DBPath == "" {
		panic("db path should not be empty")
	}
}

type ChainConfig struct {
	BalanceMonitorInterval int64 `json:"balance_monitor_interval"`

	SourceChainObserverFetchInterval    int64  `json:"source_chain_observer_fetch_interval"`
	SourceChainStartHeight              int64  `json:"source_chain_start_height"`
	SourceChainProvider                 string `json:"source_chain_provider"`
	SourceChainConfirmNum               int64  `json:"source_chain_confirm_num"`
	SourceChainSwapAgentAddr            string `json:"source_chain_swap_agent_addr"`
	SourceChainExplorerUrl              string `json:"source_chain_explorer_url"`
	SourceChainMaxTrackRetry            int64  `json:"source_chain_max_track_retry"`
	SourceChainAlertThreshold           string `json:"source_chain_alert_threshold"`
	SourceChainWaitMilliSecBetweenSwaps int64  `json:"source_chain_wait_milli_sec_between_swaps"`

	DestinationChainObserverFetchInterval    int64  `json:"destination_chain_observer_fetch_interval"`
	DestinationChainStartHeight              int64  `json:"destination_chain_start_height"`
	DestinationChainProvider                 string `json:"destination_chain_provider"`
	DestinationChainConfirmNum               int64  `json:"destination_chain_confirm_num"`
	DestinationChainSwapAgentAddr            string `json:"destination_chain_swap_agent_addr"`
	DestinationChainExplorerUrl              string `json:"destination_chain_explorer_url"`
	DestinationChainMaxTrackRetry            int64  `json:"destination_chain_max_track_retry"`
	DestinationChainAlertThreshold           string `json:"destination_chain_alert_threshold"`
	DestinationChainWaitMilliSecBetweenSwaps int64  `json:"destination_chain_wait_milli_sec_between_swaps"`
}

func (cfg ChainConfig) Validate() {
	if cfg.SourceChainStartHeight < 0 {
		panic("source_chain_start_height should not be less than 0")
	}
	if cfg.SourceChainProvider == "" {
		panic("source_chain_provider should not be empty")
	}
	if cfg.SourceChainConfirmNum <= 0 {
		panic("source_chain_confirm_num should be larger than 0")
	}
	if !ethcom.IsHexAddress(cfg.SourceChainSwapAgentAddr) {
		panic(fmt.Sprintf("invalid source_chain_swap_contract_addr: %s", cfg.SourceChainSwapAgentAddr))
	}
	if cfg.SourceChainMaxTrackRetry <= 0 {
		panic("source_chain_max_track_retry should be larger than 0")
	}

	if cfg.DestinationChainStartHeight < 0 {
		panic("source_chain_start_height should not be less than 0")
	}
	if cfg.DestinationChainProvider == "" {
		panic("source_chain_provider should not be empty")
	}
	if !ethcom.IsHexAddress(cfg.DestinationChainSwapAgentAddr) {
		panic(fmt.Sprintf("invalid destination_chain_swap_contract_addr: %s", cfg.DestinationChainSwapAgentAddr))
	}
	if cfg.DestinationChainConfirmNum <= 0 {
		panic("source_chain_confirm_num should be larger than 0")
	}
	if cfg.DestinationChainMaxTrackRetry <= 0 {
		panic("destination_chain_max_track_retry should be larger than 0")
	}
}

type LogConfig struct {
	Level                        string `json:"level"`
	Filename                     string `json:"filename"`
	MaxFileSizeInMB              int    `json:"max_file_size_in_mb"`
	MaxBackupsOfLogFiles         int    `json:"max_backups_of_log_files"`
	MaxAgeToRetainLogFilesInDays int    `json:"max_age_to_retain_log_files_in_days"`
	UseConsoleLogger             bool   `json:"use_console_logger"`
	UseFileLogger                bool   `json:"use_file_logger"`
	Compress                     bool   `json:"compress"`
}

func (cfg LogConfig) Validate() {
	if cfg.UseFileLogger {
		if cfg.Filename == "" {
			panic("filename should not be empty if use file logger")
		}
		if cfg.MaxFileSizeInMB <= 0 {
			panic("max_file_size_in_mb should be larger than 0 if use file logger")
		}
		if cfg.MaxBackupsOfLogFiles <= 0 {
			panic("max_backups_off_log_files should be larger than 0 if use file logger")
		}
	}
}

type AdminConfig struct {
	ListenAddr string `json:"listen_addr"`
}

func ParseConfigFromFile(filePath string) *Config {
	bz, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	var config Config
	if err := json.Unmarshal(bz, &config); err != nil {
		panic(err)
	}

	return &config
}

func ParseConfigFromJson(content string) *Config {
	var config Config
	if err := json.Unmarshal([]byte(content), &config); err != nil {
		panic(err)
	}

	return &config
}
