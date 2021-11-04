package main

import (
	"flag"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	contractabi "github.com/synycboom/bsc-evm-compatible-bridge-core/abi"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/agent"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/client"
	engine "github.com/synycboom/bsc-evm-compatible-bridge-core/engine/erc721"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/model"
	observer "github.com/synycboom/bsc-evm-compatible-bridge-core/observer"
	recorder "github.com/synycboom/bsc-evm-compatible-bridge-core/recorder/erc721"
	token "github.com/synycboom/bsc-evm-compatible-bridge-core/token/erc721"
	"github.com/synycboom/bsc-evm-compatible-bridge-core/util"
)

const (
	flagConfigType         = "config-type"
	flagConfigAwsRegion    = "aws-region"
	flagConfigAwsSecretKey = "aws-secret-key"
	flagConfigPath         = "config-path"
)

const (
	ConfigTypeLocal = "local"
	ConfigTypeAws   = "aws"
)

func initFlags() {
	flag.String(flagConfigPath, "", "config path")
	flag.String(flagConfigType, "", "config type, local or aws")
	flag.String(flagConfigAwsRegion, "", "aws s3 region")
	flag.String(flagConfigAwsSecretKey, "", "aws s3 secret key")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		panic(fmt.Sprintf("bind flags error, err=%s", err))
	}
}

func printUsage() {
	fmt.Print("usage: ./swap --config-type [local or aws] --config-path config_file_path\n")
}

func main() {
	initFlags()

	configType := viper.GetString(flagConfigType)
	if configType == "" {
		printUsage()
		return
	}

	if configType != ConfigTypeAws && configType != ConfigTypeLocal {
		printUsage()
		return
	}

	var config *util.Config
	if configType == ConfigTypeAws {
		awsSecretKey := viper.GetString(flagConfigAwsSecretKey)
		if awsSecretKey == "" {
			printUsage()
			return
		}

		awsRegion := viper.GetString(flagConfigAwsRegion)
		if awsRegion == "" {
			printUsage()
			return
		}

		configContent, err := util.GetSecret(awsSecretKey, awsRegion)
		if err != nil {
			fmt.Printf("get aws config error, err=%s", err.Error())
			return
		}
		config = util.ParseConfigFromJson(configContent)
	} else {
		configFilePath := viper.GetString(flagConfigPath)
		if configFilePath == "" {
			printUsage()
			return
		}
		config = util.ParseConfigFromFile(configFilePath)
	}
	config.Validate()

	util.InitLogger(config.LogConfig)
	util.InitTgAlerter(config.AlertConfig)

	mysqlConn := mysql.New(mysql.Config{
		DSN: config.DBConfig.DSN,
	})
	db, err := gorm.Open(mysqlConn, &gorm.Config{
		Logger: logger.Default.LogMode(dbLogLevel(config.DBConfig.LogLevel)),
	})
	if err != nil {
		panic(errors.Wrap(err, "[main]: open db error"))
	}

	model.InitTables(db)

	swapAgents := make(map[string]agent.SwapAgent)
	swapAgentAddresses := make(map[string]common.Address)
	tokens := make(map[string]token.IToken)
	clients := make(map[string]client.ETHClient)
	for _, c := range config.ChainConfigs {
		ec, err := ethclient.Dial(c.Provider)
		if err != nil {
			panic(errors.Wrap(err, "[main]: new eth client error"))
		}

		swapAgentAddr := common.HexToAddress(c.SwapAgentAddr)
		swapAgent, err := contractabi.NewERC721SwapAgent(swapAgentAddr, ec)
		if err != nil {
			panic(errors.Wrap(err, "[main]: failed to create swap agent"))
		}

		tokens[c.ID] = token.NewToken(ec)
		clients[c.ID] = client.NewClient(ec)
		swapAgents[c.ID] = swapAgent
		swapAgentAddresses[c.ID] = swapAgentAddr
	}

	recorders := make(map[string]recorder.IRecorder)
	for _, c := range config.ChainConfigs {
		chainID := util.StrToBigInt(c.ID)
		if chainID.Cmp(big.NewInt(0)) == 0 {
			panic(errors.New("[main]: chain id is 0"))
		}

		recorders[c.ID] = recorder.NewRecorder(&recorder.Config{
			ChainID:   chainID,
			ChainName: c.Name,
			HMACKey:   config.KeyManagerConfig.HMACKey,
		}, &recorder.Dependencies{
			Client:    clients,
			DB:        db.Session(&gorm.Session{}),
			SwapAgent: swapAgents,
			Token:     tokens,
		})
	}

	for _, c := range config.ChainConfigs {
		chainID := util.StrToBigInt(c.ID)

		// TODO: implement SwapAgent instance and implement mutex lock to prevent multiple calls
		// TODO: send tg when logging has error
		ob := observer.NewObserver(&observer.Config{
			StartHeight:        c.StartHeight,
			ConfirmNum:         c.ConfirmNum,
			FetchInterval:      time.Duration(c.ObserverFetchInterval) * time.Second,
			BlockUpdateTimeout: time.Duration(config.AlertConfig.BlockUpdateTimeout) * time.Second,
		}, &observer.Dependencies{
			DB:       db.Session(&gorm.Session{}),
			Recorder: recorders[c.ID],
		})
		ob.Start()

		e := engine.NewEngine(&engine.Config{
			ChainID:            chainID,
			ConfirmNum:         c.ConfirmNum,
			ExplorerURL:        c.ExplorerUrl,
			PrivateKey:         c.PrivateKey,
			MaxTrackRetry:      c.MaxTrackRetry,
			SwapAgentAddresses: swapAgentAddresses,
		}, &engine.Dependencies{
			Client:    clients,
			DB:        db.Session(&gorm.Session{}),
			Recorder:  recorders,
			SwapAgent: swapAgents,
		})
		e.Start()
	}

	select {}
}

func dbLogLevel(level string) logger.LogLevel {
	switch level {
	case "SILENT":
		return logger.Silent
	case "ERROR":
		return logger.Error
	case "WARN":
		return logger.Warn
	case "INFO":
		return logger.Info
	}

	return logger.Warn
}
