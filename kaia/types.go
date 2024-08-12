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

package kaia

import (
	"context"
	"fmt"

	"github.com/klaytn/klaytn/networks/rpc"
	"github.com/klaytn/klaytn/params"

	"github.com/klaytn/rosetta-sdk-go-klaytn/types"
)

const (
	// NodeVersion is the version of kaia we are using.
	NodeVersion = "1.8.4"

	// Blockchain is Kaia.
	Blockchain string = "Kaia"

	// MainnetNetwork is the value of the network
	// in MainnetNetworkIdentifier (which is Cypress).
	MainnetNetwork string = "Mainnet"

	// TestnetNetwork is the value of the network
	// in TestnetNetworkIdentifier.
	TestnetNetwork string = "Testnet"

	// LocalNetwork is the value of the network
	// in LocalNetworkIdentifier.
	LocalNetwork string = "Local"

	// Symbol is the symbol value
	// used in Currency.
	Symbol = "KLAY"

	// Decimals is the decimals value
	// used in Currency.
	Decimals = 18

	// BlockRewardOpType is used to describe
	// a block reward.
	BlockRewardOpType = "BLOCK_REWARD"

	// FeeOpType is used to represent fee operations.
	FeeOpType = "FEE"

	// CallOpType is used to represent CALL trace operations.
	CallOpType = "CALL"

	// CreateOpType is used to represent CREATE trace operations.
	CreateOpType = "CREATE"

	// Create2OpType is used to represent CREATE2 trace operations.
	Create2OpType = "CREATE2"

	// SelfDestructOpType is used to represent SELFDESTRUCT trace operations.
	SelfDestructOpType = "SELFDESTRUCT"

	// CallCodeOpType is used to represent CALLCODE trace operations.
	CallCodeOpType = "CALLCODE"

	// DelegateCallOpType is used to represent DELEGATECALL trace operations.
	DelegateCallOpType = "DELEGATECALL"

	// StaticCallOpType is used to represent STATICCALL trace operations.
	StaticCallOpType = "STATICCALL"

	// DestructOpType is a synthetic operation used to represent the
	// deletion of suicided accounts that still have funds at the end
	// of a transaction.
	DestructOpType = "DESTRUCT"

	// SuccessStatus is the status of any
	// Kaia operation considered successful.
	SuccessStatus = "SUCCESS"

	// FailureStatus is the status of any
	// Kaia operation considered unsuccessful.
	FailureStatus = "FAILURE"

	// HistoricalBalanceSupported is whether
	// historical balance is supported.
	HistoricalBalanceSupported = true

	// GenesisBlockIndex is the index of the
	// genesis block.
	GenesisBlockIndex = int64(0)

	// TransferGasLimit is the gas limit
	// of a transfer.
	TransferGasLimit = int64(21000) //nolint:gomnd

	// NodeArguments are the arguments to start a kaia node instance.
	NodeArguments = `--conf /app/kaia/ken.yaml --gcmode archive --rpc --rpcapi debug,admin,txpool,klay,governance --rpcport 8551`

	// IncludeMempoolCoins does not apply to rosetta-kaia as it is not UTXO-based.
	IncludeMempoolCoins = false
)

var (
	// MainnetNodeArguments are the arguments to start a Mainnet node
	MainnetNodeArguments = fmt.Sprintf("%s --cypress", NodeArguments)

	// TestnetNodeArguments are the arguments to start a Testnet node
	TestnetNodeArguments = fmt.Sprintf("%s --baobab", NodeArguments)

	// MainnetGenesisBlockIdentifier is the *types.BlockIdentifier
	// of the mainnet genesis block.
	MainnetGenesisBlockIdentifier = &types.BlockIdentifier{
		Hash:  params.CypressGenesisHash.Hex(),
		Index: GenesisBlockIndex,
	}

	// TestnetGenesisBlockIdentifier is the *types.BlockIdentifier
	// of the Testnet(Baobab) genesis block.
	TestnetGenesisBlockIdentifier = &types.BlockIdentifier{
		Hash:  params.BaobabGenesisHash.Hex(),
		Index: GenesisBlockIndex,
	}

	// LocalGenesisBlockIdentifier is the *types.BlockIdentifier
	// of the Local network genesis block.
	LocalGenesisBlockIdentifier = &types.BlockIdentifier{
		Hash:  "0xb9d1d87259d7c7badd0a4ce2268a2fce81c7fe944905ac04dd2e7872a20b2087",
		Index: GenesisBlockIndex,
	}

	// Currency is the *types.Currency for all
	// Kaia networks.
	Currency = &types.Currency{
		Symbol:   Symbol,
		Decimals: Decimals,
	}

	// OperationTypes are all suppoorted operation types.
	OperationTypes = []string{
		BlockRewardOpType,
		FeeOpType,
		CallOpType,
		CreateOpType,
		Create2OpType,
		SelfDestructOpType,
		CallCodeOpType,
		DelegateCallOpType,
		StaticCallOpType,
		DestructOpType,
	}

	// OperationStatuses are all supported operation statuses.
	OperationStatuses = []*types.OperationStatus{
		{
			Status:     SuccessStatus,
			Successful: true,
		},
		{
			Status:     FailureStatus,
			Successful: false,
		},
	}

	// CallMethods are all supported call methods.
	CallMethods = []string{
		"klay_getBlockByNumber",
		"klay_getTransactionReceipt",
		"klay_call",
		"klay_estimateGas",
	}
)

// JSONRPC is the interface for accessing Kaia's JSON RPC endpoint.
type JSONRPC interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
	Close()
}

// GraphQL is the interface for accessing Kaia's GraphQL endpoint.
type GraphQL interface {
	Query(ctx context.Context, input string) (string, error)
}

// CallType returns a boolean indicating
// if the provided trace type is a call type.
func CallType(t string) bool {
	callTypes := []string{
		CallOpType,
		CallCodeOpType,
		DelegateCallOpType,
		StaticCallOpType,
	}

	for _, callType := range callTypes {
		if callType == t {
			return true
		}
	}

	return false
}

// CreateType returns a boolean indicating
// if the provided trace type is a create type.
func CreateType(t string) bool {
	createTypes := []string{
		CreateOpType,
		Create2OpType,
	}

	for _, createType := range createTypes {
		if createType == t {
			return true
		}
	}

	return false
}
