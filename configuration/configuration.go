// Copyright 2020 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Modifications Copyright © 2022 Klaytn
// Modified and improved for the Klaytn development.
// Modifications Copyright 2024 Rosetta-kaia developers
// Modified and improved for the Kaia development

package configuration

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/kaiachain/rosetta-kaia/kaia"
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/params"

	"github.com/klaytn/rosetta-sdk-go-klaytn/types"
)

// Mode is the setting that determines if
// the implementation is "online" or "offline".
type Mode string

const (
	// Online is when the implementation is permitted
	// to make outbound connections.
	Online Mode = "ONLINE"

	// Offline is when the implementation is not permitted
	// to make outbound connections.
	Offline Mode = "OFFLINE"

	// Mainnet is the Klaytn Mainnet.
	Mainnet string = "MAINNET"

	// Testnet is the Klaytn Baobab testnet.
	Testnet string = "TESTNET"

	// Local is a local private network for testing purpose.
	Local string = "LOCAL"

	// DataDirectory is the default location for all
	// persistent data.
	DataDirectory = "/data"

	// ModeEnv is the environment variable read
	// to determine mode.
	ModeEnv = "MODE"

	// NetworkEnv is the environment variable
	// read to determine network.
	NetworkEnv = "NETWORK"

	// PortEnv is the environment variable
	// read to determine the port for the Rosetta
	// implementation.
	PortEnv = "PORT"

	// KENEnv is an optional environment variable
	// used to connect rosetta-kaia to an already
	// running kaia client node.
	KENEnv = "KEN"

	// DefaultKENURL is the default URL for
	// a running ken node. This is used
	// when KENEnv is not populated.
	DefaultKENURL = "http://localhost:8551"

	// SkipAdminEnv is an optional environment variable
	// to skip `admin` calls which are typically not supported
	// by hosted node services. When not set, defaults to false.
	SkipAdminEnv = "SKIP_ADMIN"

	// MiddlewareVersion is the version of rosetta-kaia.
	MiddlewareVersion = "1.0.7"
)

// Configuration determines how
type Configuration struct {
	Mode                   Mode
	Network                *types.NetworkIdentifier
	GenesisBlockIdentifier *types.BlockIdentifier
	NodeURL                string
	RemoteNode             bool
	Port                   int
	NodeArguments          string
	SkipAdmin              bool

	// Block Reward Data
	Params *params.ChainConfig
}

// LoadConfiguration attempts to create a new Configuration
// using the ENVs in the environment.
func LoadConfiguration() (*Configuration, error) {
	config := &Configuration{}

	modeValue := Mode(os.Getenv(ModeEnv))
	switch modeValue {
	case Online:
		config.Mode = Online
	case Offline:
		config.Mode = Offline
	case "":
		return nil, errors.New("MODE must be populated")
	default:
		return nil, fmt.Errorf("%s is not a valid mode", modeValue)
	}

	networkValue := os.Getenv(NetworkEnv)
	switch networkValue {
	case Mainnet:
		config.Network = &types.NetworkIdentifier{
			Blockchain: kaia.Blockchain,
			Network:    kaia.MainnetNetwork,
		}
		config.GenesisBlockIdentifier = kaia.MainnetGenesisBlockIdentifier
		config.Params = params.CypressChainConfig
		config.NodeArguments = kaia.MainnetNodeArguments
	case Testnet:
		config.Network = &types.NetworkIdentifier{
			Blockchain: kaia.Blockchain,
			Network:    kaia.TestnetNetwork,
		}
		config.GenesisBlockIdentifier = kaia.TestnetGenesisBlockIdentifier
		config.Params = params.BaobabChainConfig
		config.NodeArguments = kaia.TestnetNodeArguments
	case Local:
		config.Network = &types.NetworkIdentifier{
			Blockchain: kaia.Blockchain,
			Network:    kaia.LocalNetwork,
		}
		config.GenesisBlockIdentifier = kaia.LocalGenesisBlockIdentifier
		mintingAmount, _ := new(big.Int).SetString("9600000000000000000", 10) // nolint
		config.Params = &params.ChainConfig{
			ChainID:                  big.NewInt(int64(940625)), // nolint
			IstanbulCompatibleBlock:  big.NewInt(0),
			LondonCompatibleBlock:    big.NewInt(0),
			EthTxTypeCompatibleBlock: big.NewInt(0),
			DeriveShaImpl:            2, // nolint:gomnd
			Governance: &params.GovernanceConfig{
				GoverningNode:  common.HexToAddress("0x9712f943b296758aaae79944ec975884188d3a96"),
				GovernanceMode: "single",
				Reward: &params.RewardConfig{
					MintingAmount:          mintingAmount,
					Ratio:                  "34/54/12",
					UseGiniCoeff:           false,
					DeferredTxFee:          true,
					StakingUpdateInterval:  60,                  // nolint:gomnd
					ProposerUpdateInterval: 30,                  // nolint:gomnd
					MinimumStake:           big.NewInt(5000000), // nolint:gomnd
				},
			},
			Istanbul: &params.IstanbulConfig{
				Epoch:          30, // nolint:gomnd
				ProposerPolicy: 2,  // nolint:gomnd
				SubGroupSize:   1,
			},
			UnitPrice: 25000000000, // nolint:gomnd
		}
	case "":
		return nil, errors.New("NETWORK must be populated")
	default:
		return nil, fmt.Errorf("%s is not a valid network", networkValue)
	}

	config.NodeURL = DefaultKENURL
	envNodeURL := os.Getenv(KENEnv)
	if len(envNodeURL) > 0 {
		config.RemoteNode = true
		config.NodeURL = envNodeURL
	}

	config.SkipAdmin = false
	envSkipAdmin := os.Getenv(SkipAdminEnv)
	if len(envSkipAdmin) > 0 {
		val, err := strconv.ParseBool(envSkipAdmin)
		if err != nil {
			return nil, fmt.Errorf("%w: unable to parse SKIP_ADMIN %s", err, envSkipAdmin)
		}
		config.SkipAdmin = val
	}

	portValue := os.Getenv(PortEnv)
	if len(portValue) == 0 {
		return nil, errors.New("PORT must be populated")
	}

	port, err := strconv.Atoi(portValue)
	if err != nil || len(portValue) == 0 || port <= 0 {
		return nil, fmt.Errorf("%w: unable to parse port %s", err, portValue)
	}
	config.Port = port

	return config, nil
}
