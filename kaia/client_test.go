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

package kaia

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	mocks "github.com/kaiachain/rosetta-kaia/mocks/kaia"
	"github.com/klaytn/klaytn"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/blockchain/types/account"
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/common/hexutil"
	"github.com/klaytn/klaytn/networks/p2p"
	"github.com/klaytn/klaytn/node/cn/tracers"
	"github.com/klaytn/klaytn/params"
	"github.com/klaytn/klaytn/reward"
	RosettaTypes "github.com/klaytn/rosetta-sdk-go-klaytn/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sync/semaphore"
)

func TestStatus_NotReady(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"latest",
		false,
	).Return(
		nil,
	).Once()

	block, timestamp, syncStatus, peers, err := c.Status(ctx)
	assert.Nil(t, block)
	assert.Equal(t, int64(-1), timestamp)
	assert.Nil(t, syncStatus)
	assert.Nil(t, peers)
	assert.True(t, errors.Is(err, klaytn.NotFound))

	mockJSONRPC.AssertExpectations(t)
}

func TestStatus_NotSyncing(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"latest",
		false,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			header := args.Get(1).(**types.Header)
			file, err := os.ReadFile("testdata/basic_header.json")
			assert.NoError(t, err)

			*header = new(types.Header)

			assert.NoError(t, (*header).UnmarshalJSON(file))
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_syncing",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			status := args.Get(1).(*json.RawMessage)

			*status = json.RawMessage("false")
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"admin_peers",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			info := args.Get(1).(*[]*p2p.PeerInfo)

			file, err := os.ReadFile("testdata/peers.json")
			assert.NoError(t, err)

			assert.NoError(t, json.Unmarshal(file, info))
		},
	).Once()

	block, timestamp, syncStatus, peers, err := c.Status(ctx)
	assert.Equal(t, &RosettaTypes.BlockIdentifier{
		Hash:  "0x9a65e7f5b277136a4600097687abbbd76ca07662aa297be5279605aa20b73a6a",
		Index: 86613352,
	}, block)
	assert.Equal(t, int64(1648183517000), timestamp)
	assert.Nil(t, syncStatus)
	assert.Equal(t, []*RosettaTypes.Peer{
		{
			PeerID: "5cb3a87e20d10d4721bd99a73a5683e7e83b8a518cf6cfdd4098e2b5b5b4f58f928e58ca89e87ec93e10601977733cab0514b9766bb26fce3af9e329c57053c3", // nolint:lll
			Metadata: map[string]interface{}{
				"caps": []string{"istanbul/64"},
				"name": "Klaytn/v1.8.2+556f2311f3/linux-amd64/go1.13.1",
				"protocols": map[string]interface{}{
					"istanbul": map[string]interface{}{
						"blockscore": float64(376),
						"head":       "0xd0291b3fdac3e4c3e6b3ece33fb43e79cfa7af74daafa37565d13e6e0b623c8e",
						"version":    float64(64),
					},
				},
			},
		},
		{
			PeerID: "9fa2e2c701f2492ac4950824903b402727ea3c19f8b86b5c399d546c5a7d9dc7b1cfdbcadf498acd18ffde617950947512fff70bd67e45e9482f3d4273e70230", // nolint:lll
			Metadata: map[string]interface{}{
				"caps": []string{"istanbul/64"},
				"name": "Klaytn/v1.8.2+556f2311f3/linux-amd64/go1.13.1",
				"protocols": map[string]interface{}{
					"istanbul": map[string]interface{}{
						"blockscore": float64(27),
						"head":       "0x2a74c861c70533fab2323d06f30d731baef985a1c569143e3a3b847ac0f3b638",
						"version":    float64(64),
					},
				},
			},
		},
	}, peers)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestStatus_NotSyncing_SkipAdminCalls(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
		skipAdminCalls: true,
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"latest",
		false,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			header := args.Get(1).(**types.Header)
			file, err := os.ReadFile("testdata/basic_header.json")
			assert.NoError(t, err)

			*header = new(types.Header)

			assert.NoError(t, (*header).UnmarshalJSON(file))
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_syncing",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			status := args.Get(1).(*json.RawMessage)

			*status = json.RawMessage("false")
		},
	).Once()

	adminPeersSkipped := true
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"admin_peers",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			adminPeersSkipped = false
		},
	).Maybe()

	block, timestamp, syncStatus, peers, err := c.Status(ctx)
	assert.True(t, adminPeersSkipped)
	assert.Equal(t, &RosettaTypes.BlockIdentifier{
		Hash:  "0x9a65e7f5b277136a4600097687abbbd76ca07662aa297be5279605aa20b73a6a",
		Index: 86613352,
	}, block)
	assert.Equal(t, int64(1648183517000), timestamp)
	assert.Nil(t, syncStatus)
	assert.Equal(t, []*RosettaTypes.Peer{}, peers)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestStatus_Syncing(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"latest",
		false,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			header := args.Get(1).(**types.Header)
			file, err := os.ReadFile("testdata/basic_header.json")
			assert.NoError(t, err)

			*header = new(types.Header)

			assert.NoError(t, (*header).UnmarshalJSON(file))
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_syncing",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			progress := args.Get(1).(*json.RawMessage)
			file, err := os.ReadFile("testdata/syncing_info.json")
			assert.NoError(t, err)

			*progress = json.RawMessage(file)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"admin_peers",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			info := args.Get(1).(*[]*p2p.PeerInfo)

			file, err := os.ReadFile("testdata/peers.json")
			assert.NoError(t, err)

			assert.NoError(t, json.Unmarshal(file, info))
		},
	).Once()

	block, timestamp, syncStatus, peers, err := c.Status(ctx)
	assert.Equal(t, &RosettaTypes.BlockIdentifier{
		Hash:  "0x9a65e7f5b277136a4600097687abbbd76ca07662aa297be5279605aa20b73a6a",
		Index: 86613352,
	}, block)
	assert.Equal(t, int64(1648183517000), timestamp)
	assert.Equal(t, &RosettaTypes.SyncStatus{
		CurrentIndex: RosettaTypes.Int64(25),
		TargetIndex:  RosettaTypes.Int64(8916760),
	}, syncStatus)
	assert.Equal(t, []*RosettaTypes.Peer{
		{
			PeerID: "5cb3a87e20d10d4721bd99a73a5683e7e83b8a518cf6cfdd4098e2b5b5b4f58f928e58ca89e87ec93e10601977733cab0514b9766bb26fce3af9e329c57053c3", // nolint:lll
			Metadata: map[string]interface{}{
				"caps": []string{"istanbul/64"},
				"name": "Klaytn/v1.8.2+556f2311f3/linux-amd64/go1.13.1",
				"protocols": map[string]interface{}{
					"istanbul": map[string]interface{}{
						"blockscore": float64(376),
						"head":       "0xd0291b3fdac3e4c3e6b3ece33fb43e79cfa7af74daafa37565d13e6e0b623c8e",
						"version":    float64(64),
					},
				},
			},
		},
		{
			PeerID: "9fa2e2c701f2492ac4950824903b402727ea3c19f8b86b5c399d546c5a7d9dc7b1cfdbcadf498acd18ffde617950947512fff70bd67e45e9482f3d4273e70230", // nolint:lll
			Metadata: map[string]interface{}{
				"caps": []string{"istanbul/64"},
				"name": "Klaytn/v1.8.2+556f2311f3/linux-amd64/go1.13.1",
				"protocols": map[string]interface{}{
					"istanbul": map[string]interface{}{
						"blockscore": float64(27),
						"head":       "0x2a74c861c70533fab2323d06f30d731baef985a1c569143e3a3b847ac0f3b638",
						"version":    float64(64),
					},
				},
			},
		},
	}, peers)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestStatus_Syncing_SkipAdminCalls(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
		skipAdminCalls: true,
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"latest",
		false,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			header := args.Get(1).(**types.Header)
			file, err := os.ReadFile("testdata/basic_header.json")
			assert.NoError(t, err)

			*header = new(types.Header)

			assert.NoError(t, (*header).UnmarshalJSON(file))
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_syncing",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			progress := args.Get(1).(*json.RawMessage)
			file, err := os.ReadFile("testdata/syncing_info.json")
			assert.NoError(t, err)

			*progress = json.RawMessage(file)
		},
	).Once()

	adminPeersSkipped := true
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"admin_peers",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			adminPeersSkipped = false
		},
	).Maybe()

	block, timestamp, syncStatus, peers, err := c.Status(ctx)
	assert.True(t, adminPeersSkipped)
	assert.Equal(t, &RosettaTypes.BlockIdentifier{
		Hash:  "0x9a65e7f5b277136a4600097687abbbd76ca07662aa297be5279605aa20b73a6a",
		Index: 86613352,
	}, block)
	assert.Equal(t, int64(1648183517000), timestamp)
	assert.Equal(t, &RosettaTypes.SyncStatus{
		CurrentIndex: RosettaTypes.Int64(25),
		TargetIndex:  RosettaTypes.Int64(8916760),
	}, syncStatus)
	assert.Equal(t, []*RosettaTypes.Peer{}, peers)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestBalance(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	addressToQuery := "0xe63960c1c07a3041195ea1bd505f971b9f01e4e2" // nolint
	ctx := context.Background()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"latest",
		false,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*types.Header)

			file, err := os.ReadFile("testdata/basic_header.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getAccount",
		addressToQuery,
		"latest",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*account.AccountSerializer)

			file, err := os.ReadFile(
				"testdata/account_balance_0xe63960c1c07a3041195ea1bd505f971b9f01e4e2.json",
			)
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	resp, err := c.Balance(
		ctx,
		&RosettaTypes.AccountIdentifier{
			Address: addressToQuery,
		},
		nil,
	)
	assert.Equal(t, &RosettaTypes.AccountBalanceResponse{
		BlockIdentifier: &RosettaTypes.BlockIdentifier{
			Hash:  "0x9a65e7f5b277136a4600097687abbbd76ca07662aa297be5279605aa20b73a6a",
			Index: 86613352,
		},
		Balances: []*RosettaTypes.Amount{
			{
				Value:    "618252172001399",
				Currency: Currency,
			},
		},
		Metadata: map[string]interface{}{
			"nonce": int64(0),
		},
	}, resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestBalanceWithNotFoundAccount(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	addressToQuery := "0xe63960c1c07a3041195ea1bd505f971b9f01e4e2" // nolint
	ctx := context.Background()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"latest",
		false,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*types.Header)

			file, err := os.ReadFile("testdata/basic_header.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getAccount",
		addressToQuery,
		"latest",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			acct := args.Get(1).(*account.AccountSerializer)
			*acct = account.AccountSerializer{}
		},
	).Once()

	resp, err := c.Balance(
		ctx,
		&RosettaTypes.AccountIdentifier{
			Address: addressToQuery,
		},
		nil,
	)
	assert.Equal(t, &RosettaTypes.AccountBalanceResponse{
		BlockIdentifier: &RosettaTypes.BlockIdentifier{
			Hash:  "0x9a65e7f5b277136a4600097687abbbd76ca07662aa297be5279605aa20b73a6a",
			Index: 86613352,
		},
		Balances: []*RosettaTypes.Amount{
			{
				Value:    "0",
				Currency: Currency,
			},
		},
		Metadata: map[string]interface{}{
			"nonce": int64(0),
		},
	}, resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestBalance_Historical_Hash(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	addressToQuery := "0xe63960c1c07a3041195ea1bd505f971b9f01e4e2" // nolint
	ctx := context.Background()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByHash",
		"0x9a65e7f5b277136a4600097687abbbd76ca07662aa297be5279605aa20b73a6a",
		false,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*types.Header)

			file, err := os.ReadFile("testdata/basic_header.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getAccount",
		addressToQuery,
		"0x9a65e7f5b277136a4600097687abbbd76ca07662aa297be5279605aa20b73a6a",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*account.AccountSerializer)

			file, err := os.ReadFile(
				"testdata/account_balance_0xe63960c1c07a3041195ea1bd505f971b9f01e4e2.json",
			)
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	resp, err := c.Balance(
		ctx,
		&RosettaTypes.AccountIdentifier{
			Address: addressToQuery,
		},
		&RosettaTypes.PartialBlockIdentifier{
			Hash: RosettaTypes.String(
				"0x9a65e7f5b277136a4600097687abbbd76ca07662aa297be5279605aa20b73a6a",
			),
			Index: RosettaTypes.Int64(86613352),
		},
	)
	assert.Equal(t, &RosettaTypes.AccountBalanceResponse{
		BlockIdentifier: &RosettaTypes.BlockIdentifier{
			Hash:  "0x9a65e7f5b277136a4600097687abbbd76ca07662aa297be5279605aa20b73a6a",
			Index: 86613352,
		},
		Balances: []*RosettaTypes.Amount{
			{
				Value:    "618252172001399",
				Currency: Currency,
			},
		},
		Metadata: map[string]interface{}{
			"nonce": int64(0),
		},
	}, resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestBalance_Historical_Index(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	addressToQuery := "0xe63960c1c07a3041195ea1bd505f971b9f01e4e2" // nolint
	ctx := context.Background()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x5299d68",
		false,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*types.Header)

			file, err := os.ReadFile("testdata/basic_header.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getAccount",
		addressToQuery,
		"0x5299d68",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*account.AccountSerializer)

			file, err := os.ReadFile(
				"testdata/account_balance_0xe63960c1c07a3041195ea1bd505f971b9f01e4e2.json",
			)
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	resp, err := c.Balance(
		ctx,
		&RosettaTypes.AccountIdentifier{
			Address: addressToQuery,
		},
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(86613352),
		},
	)
	assert.Equal(t, &RosettaTypes.AccountBalanceResponse{
		BlockIdentifier: &RosettaTypes.BlockIdentifier{
			Hash:  "0x9a65e7f5b277136a4600097687abbbd76ca07662aa297be5279605aa20b73a6a",
			Index: 86613352,
		},
		Balances: []*RosettaTypes.Amount{
			{
				Value:    "618252172001399",
				Currency: Currency,
			},
		},
		Metadata: map[string]interface{}{
			"nonce": int64(0),
		},
	}, resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestBalance_InvalidAddress(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	addressToQuery := "0x4cfc400fed52f9681b42454c2db4b18ab98f8de"
	ctx := context.Background()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getAccount",
		addressToQuery,
		"latest",
	).Return(
		errors.New("invalid argument 0: json: cannot unmarshal hex string of odd length into Go value of type common.Address"), // nolint:lll
	)

	resp, err := c.Balance(
		ctx,
		&RosettaTypes.AccountIdentifier{
			Address: "0x4cfc400fed52f9681b42454c2db4b18ab98f8de",
		},
		nil,
	)
	assert.Nil(t, resp)
	assert.Error(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestBalance_InvalidHash(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	addressToQuery := "0xe63960c1c07a3041195ea1bd505f971b9f01e4e2" // nolint
	ctx := context.Background()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getAccount",
		addressToQuery,
		"0x7d2a2713026a0e66f131878de2bb2df2fff6c24562c1df61ec0265e5fedf2626",
	).Return(
		errors.New("header for hash not found"),
	)

	resp, err := c.Balance(
		ctx,
		&RosettaTypes.AccountIdentifier{
			Address: addressToQuery,
		},
		&RosettaTypes.PartialBlockIdentifier{
			Hash: RosettaTypes.String(
				"0x7d2a2713026a0e66f131878de2bb2df2fff6c24562c1df61ec0265e5fedf2626",
			),
		},
	)
	assert.Nil(t, resp)
	assert.Error(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestCall_GetBlockByNumber(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x2af0",
		false,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/block_10992.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_10992.json")
	assert.NoError(t, err)
	var correct map[string]interface{}
	assert.NoError(t, json.Unmarshal(correctRaw, &correct))

	resp, err := c.Call(
		ctx,
		&RosettaTypes.CallRequest{
			Method: "klay_getBlockByNumber",
			Parameters: map[string]interface{}{
				"index":                    RosettaTypes.Int64(10992),
				"show_transaction_details": false,
			},
		},
	)
	assert.Equal(t, &RosettaTypes.CallResponse{
		Result:     correct,
		Idempotent: false,
	}, resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestCall_GetBlockByNumber_InvalidArgs(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	resp, err := c.Call(
		ctx,
		&RosettaTypes.CallRequest{
			Method: "klay_getBlockByNumber",
			Parameters: map[string]interface{}{
				"index":                    "a string",
				"show_transaction_details": false,
			},
		},
	)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, ErrCallParametersInvalid))

	mockJSONRPC.AssertExpectations(t)
}

func TestCall_GetTransactionReceipt(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getTransactionReceipt",
		"0x3ee9ad0bd4e344e5492fdb5e3446534d6a3f815278979fe7c75449cc2ab6eee8",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile(
				"testdata/call_0x3ee9ad0bd4e344e5492fdb5e3446534d6a3f815278979fe7c75449cc2ab6eee8.json",
			)
			assert.NoError(t, err)

			err = json.Unmarshal(file, &r)
			assert.NoError(t, err)
		},
	).Once()
	resp, err := c.Call(
		ctx,
		&RosettaTypes.CallRequest{
			Method: "klay_getTransactionReceipt",
			Parameters: map[string]interface{}{
				"tx_hash": "0x3ee9ad0bd4e344e5492fdb5e3446534d6a3f815278979fe7c75449cc2ab6eee8",
			},
		},
	)
	assert.Equal(t, &RosettaTypes.CallResponse{
		Result: map[string]interface{}{
			"contractAddress": nil,
			"gasUsed":         "0x5208",
			"logs":            []interface{}{},
			"logsBloom":       "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", // nolint:lll
			"status":          "0x1",
			"transactionHash": "0x3ee9ad0bd4e344e5492fdb5e3446534d6a3f815278979fe7c75449cc2ab6eee8",
		},
		Idempotent: false,
	}, resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestCall_GetTransactionReceipt_InvalidArgs(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	resp, err := c.Call(
		ctx,
		&RosettaTypes.CallRequest{
			Method: "klay_getTransactionReceipt",
		},
	)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, ErrCallParametersInvalid))

	mockJSONRPC.AssertExpectations(t)
}

func TestCall_Call(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_call",
		map[string]string{
			"to":   "0xB5E5D0F8C0cbA267CD3D7035d6AdC8eBA7Df7Cdd",
			"data": "0x70a08231000000000000000000000000b5e5d0f8c0cba267cd3d7035d6adc8eba7df7cdd",
		},
		toBlockNumArg(big.NewInt(11408349)),
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*string)

			var expected map[string]interface{}
			file, err := os.ReadFile("testdata/call_balance_11408349.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, &expected)
			assert.NoError(t, err)

			*r = expected["data"].(string)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/call_balance_11408349.json")
	assert.NoError(t, err)
	var correct map[string]interface{}
	assert.NoError(t, json.Unmarshal(correctRaw, &correct))

	resp, err := c.Call(
		ctx,
		&RosettaTypes.CallRequest{
			Method: "klay_call",
			Parameters: map[string]interface{}{
				"index": 11408349,
				"to":    "0xB5E5D0F8C0cbA267CD3D7035d6AdC8eBA7Df7Cdd",
				"data":  "0x70a08231000000000000000000000000b5e5d0f8c0cba267cd3d7035d6adc8eba7df7cdd",
			},
		},
	)
	assert.Equal(t, &RosettaTypes.CallResponse{
		Result:     correct,
		Idempotent: false,
	}, resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestCall_Call_InvalidArgs(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	resp, err := c.Call(
		ctx,
		&RosettaTypes.CallRequest{
			Method: "klay_call",
			Parameters: map[string]interface{}{
				"index": 11408349,
				"Hash":  "0x73fc065bc04f16c98247f8ec1e990f581ec58723bcd8059de85f93ab18706448",
				"to":    "not valid  ",
				"data":  "0x70a08231000000000000000000000000b5e5d0f8c0cba267cd3d7035d6adc8eba7df7cdd",
			},
		},
	)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, ErrCallParametersInvalid))

	mockJSONRPC.AssertExpectations(t)
}

func TestCall_EstimateGas(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_estimateGas",
		map[string]string{
			"from": "0xE550f300E477C60CE7e7172d12e5a27e9379D2e3",
			"to":   "0xaD6D458402F60fD3Bd25163575031ACDce07538D",
			"data": "0xa9059cbb000000000000000000000000ae7e48ee0f758cd706b76cf7e2175d982800879a" +
				"00000000000000000000000000000000000000000000000000521c5f98b8ea00",
		},
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*string)

			var expected map[string]interface{}
			file, err := os.ReadFile(
				"testdata/estimate_gas_0xaD6D458402F60fD3Bd25163575031ACDce07538D.json",
			)
			assert.NoError(t, err)

			err = json.Unmarshal(file, &expected)
			assert.NoError(t, err)

			*r = expected["data"].(string)
		},
	).Once()

	correctRaw, err := os.ReadFile(
		"testdata/estimate_gas_0xaD6D458402F60fD3Bd25163575031ACDce07538D.json",
	)
	assert.NoError(t, err)
	var correct map[string]interface{}
	assert.NoError(t, json.Unmarshal(correctRaw, &correct))

	resp, err := c.Call(
		ctx,
		&RosettaTypes.CallRequest{
			Method: "klay_estimateGas",
			Parameters: map[string]interface{}{
				"from": "0xE550f300E477C60CE7e7172d12e5a27e9379D2e3",
				"to":   "0xaD6D458402F60fD3Bd25163575031ACDce07538D",
				"data": "0xa9059cbb000000000000000000000000ae7e48ee0f758cd706b76cf7e2175d982800879a" +
					"00000000000000000000000000000000000000000000000000521c5f98b8ea00",
			},
		},
	)
	assert.Equal(t, &RosettaTypes.CallResponse{
		Result:     correct,
		Idempotent: false,
	}, resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestCall_EstimateGas_InvalidArgs(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	resp, err := c.Call(
		ctx,
		&RosettaTypes.CallRequest{
			Method: "klay_estimateGas",
			Parameters: map[string]interface{}{
				"From": "0xE550f300E477C60CE7e7172d12e5a27e9379D2e3",
				"to":   "0xaD6D458402F60fD3Bd25163575031ACDce07538D",
			},
		},
	)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, ErrCallParametersInvalid))

	mockJSONRPC.AssertExpectations(t)
}

func TestCall_InvalidMethod(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	resp, err := c.Call(
		ctx,
		&RosettaTypes.CallRequest{
			Method: "blah",
		},
	)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, ErrCallMethodInvalid))

	mockJSONRPC.AssertExpectations(t)
}

func testTraceConfig() *tracers.TraceConfig {
	fct := "fastCallTracer"
	return &tracers.TraceConfig{
		Timeout: &tracerTimeout,
		Tracer:  &fct,
	}
}

func TestBlock_Current(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"latest",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_10992.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0xb83667486a9d55238120b71b0f2e0e60e314cf25bb32020a813c6281dd43211e"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0xb83667486a9d55238120b71b0f2e0e60e314cf25bb32020a813c6281dd43211e.json",
			) // nolint
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x2af0",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x2af0",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x2af0",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_10992.json")
	assert.NoError(t, err)
	var correct *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correct))

	resp, err := c.Block(
		ctx,
		nil,
	)
	assert.Equal(t, correct.Block, resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestBlock_Hash(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByHash",
		"0xb83667486a9d55238120b71b0f2e0e60e314cf25bb32020a813c6281dd43211e",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_10992.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0xb83667486a9d55238120b71b0f2e0e60e314cf25bb32020a813c6281dd43211e"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0xb83667486a9d55238120b71b0f2e0e60e314cf25bb32020a813c6281dd43211e.json",
			) // nolint
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x2af0",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x2af0",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x2af0",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_10992.json")
	assert.NoError(t, err)
	var correct *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correct))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Hash: RosettaTypes.String(
				"0xb83667486a9d55238120b71b0f2e0e60e314cf25bb32020a813c6281dd43211e",
			),
		},
	)
	assert.Equal(t, correct.Block, resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestBlock_Index(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x2af0",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_10992.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0xb83667486a9d55238120b71b0f2e0e60e314cf25bb32020a813c6281dd43211e"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0xb83667486a9d55238120b71b0f2e0e60e314cf25bb32020a813c6281dd43211e.json",
			) // nolint
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x2af0",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x2af0",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x2af0",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_10992.json")
	assert.NoError(t, err)
	var correct *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correct))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(10992),
		},
	)
	assert.Equal(t, correct.Block, resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestBlock_FirstBlock(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x0",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_0.json")
			assert.NoError(t, err)

			*r = file
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_0.json")
	assert.NoError(t, err)
	var correct *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correct))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(0),
		},
	)
	assert.Equal(t, correct.Block, resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func jsonifyTransaction(b *RosettaTypes.Transaction) (*RosettaTypes.Transaction, error) {
	bytes, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}

	var tx RosettaTypes.Transaction
	if err := json.Unmarshal(bytes, &tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

func jsonifyBlock(b *RosettaTypes.Block) (*RosettaTypes.Block, error) {
	bytes, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}

	var bo RosettaTypes.Block
	if err := json.Unmarshal(bytes, &bo); err != nil {
		return nil, err
	}

	return &bo, nil
}

func TestTransaction_Hash(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}
	txHash := "0x99a7de4441bcac267741a269aecf4a498fcd6b33bbe442955eac1d17b74ab547"
	blockHash := "0x03cb20a342485a5bed95291b8e03fab32f5309ca69d342ef8884213b90bd454f"

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getTransactionByHash",
		txHash,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/transaction_0x99a7de4441bcac267741a269aecf4a498fcd6b33bbe442955eac1d17b74ab547.json",
			) // nolint
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByHash",
		blockHash,
		false,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(**types.Header)

			file, err := os.ReadFile(
				"testdata/block_0x03cb20a342485a5bed95291b8e03fab32f5309ca69d342ef8884213b90bd454f.json",
			) // nolint
			assert.NoError(t, err)

			*r = new(types.Header)

			assert.NoError(t, (*r).UnmarshalJSON(file))
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getTransactionReceipt",
		common.HexToHash(txHash),
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(**types.Receipt)

			file, err := os.ReadFile(
				"testdata/tx_receipt_0x99a7de4441bcac267741a269aecf4a498fcd6b33bbe442955eac1d17b74ab547.json",
			) // nolint
			assert.NoError(t, err)

			*r = new(types.Receipt)

			assert.NoError(t, (*r).UnmarshalJSON(file))
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceTransaction",
		common.HexToHash(txHash),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/transaction_trace_0x99a7de4441bcac267741a269aecf4a498fcd6b33bbe442955eac1d17b74ab547.json",
			) // nolint
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getHeaderByNumber",
		"0x5369ac3",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile(
				"testdata/block_0x03cb20a342485a5bed95291b8e03fab32f5309ca69d342ef8884213b90bd454f.json",
			) // nolint:lll
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile(
		"testdata/transaction_response_0x99a7de4441bcac267741a269aecf4a498fcd6b33bbe442955eac1d17b74ab547.json",
	) // nolint
	assert.NoError(t, err)
	var correct *RosettaTypes.BlockTransactionResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correct))

	resp, err := c.Transaction(
		ctx,
		&RosettaTypes.BlockIdentifier{
			Hash: blockHash,
		},
		&RosettaTypes.TransactionIdentifier{
			Hash: txHash,
		},
	)
	assert.NoError(t, err)

	jsonResp, err := jsonifyTransaction(resp)
	assert.NoError(t, err)
	assert.Equal(t, correct.Transaction, jsonResp)

	mockJSONRPC.AssertExpectations(t)
}

// Block with transaction
func TestBlock_2500994(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x262982",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_2500994.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockReceipts",
		common.HexToHash("0x2c1792f411ec54ef09d0c93d71133cac48a88da91c7fc0f620a451995f9169f8"),
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*[]*types.Receipt)

			file, err := os.ReadFile(
				"testdata/block_receipts_0x2c1792f411ec54ef09d0c93d71133cac48a88da91c7fc0f620a451995f9169f8.json",
			) // nolint
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0x2c1792f411ec54ef09d0c93d71133cac48a88da91c7fc0f620a451995f9169f8"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0x2c1792f411ec54ef09d0c93d71133cac48a88da91c7fc0f620a451995f9169f8.json",
			) // nolint
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getHeaderByNumber",
		"0x262982",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/block_2500994.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x262982",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x262982",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x262982",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_2500994.json")
	assert.NoError(t, err)
	var correctResp *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correctResp))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(2500994),
		},
	)
	assert.NoError(t, err)

	// Ensure types match
	jsonResp, err := jsonifyBlock(resp)
	assert.NoError(t, err)
	assert.Equal(t, correctResp.Block, jsonResp)

	mockJSONRPC.AssertExpectations(t)
}

// Block with partial success transaction
func TestBlock_87561170(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x53813d2",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_87561170.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0x5f7c9debf1b2f7fc0beefb2478acfb331a271e5f8c42a971040d0ea19e9acfa3"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0x5f7c9debf1b2f7fc0beefb2478acfb331a271e5f8c42a971040d0ea19e9acfa3.json",
			) // nolint
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockReceipts",
		common.HexToHash("0x5f7c9debf1b2f7fc0beefb2478acfb331a271e5f8c42a971040d0ea19e9acfa3"),
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*[]*types.Receipt)

			file, err := os.ReadFile(
				"testdata/block_receipts_0x5f7c9debf1b2f7fc0beefb2478acfb331a271e5f8c42a971040d0ea19e9acfa3.json",
			) // nolint
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getHeaderByNumber",
		"0x53813d2",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/block_87561170.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x53813d2",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x53813d2",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x53813d2",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_87561170.json")
	assert.NoError(t, err)
	var correctResp *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correctResp))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(87561170),
		},
	)
	assert.NoError(t, err)

	// Ensure types match
	jsonResp, err := jsonifyBlock(resp)
	assert.NoError(t, err)
	assert.Equal(t, correctResp.Block, jsonResp)

	mockJSONRPC.AssertExpectations(t)
}

// Block with transfer to destroyed contract
func TestBlock_363415(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x58b97",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_363415.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0x61c3e87c8a2b7f83cba5a03d027fbabeefbc7cc4653a50c2277d7047a1dfbe9b"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0x61c3e87c8a2b7f83cba5a03d027fbabeefbc7cc4653a50c2277d7047a1dfbe9b.json",
			) // nolint
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockReceipts",
		common.HexToHash("0x61c3e87c8a2b7f83cba5a03d027fbabeefbc7cc4653a50c2277d7047a1dfbe9b"),
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*[]*types.Receipt)

			file, err := os.ReadFile(
				"testdata/block_receipts_0x61c3e87c8a2b7f83cba5a03d027fbabeefbc7cc4653a50c2277d7047a1dfbe9b.json",
			) // nolint
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getHeaderByNumber",
		"0x58b97",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/block_363415.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	)
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x58b97",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x58b97",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x58b97",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_363415.json")
	assert.NoError(t, err)
	var correctResp *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correctResp))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(363415),
		},
	)
	assert.NoError(t, err)

	// Ensure types match
	jsonResp, err := jsonifyBlock(resp)
	assert.NoError(t, err)
	assert.Equal(t, correctResp.Block, jsonResp)

	mockJSONRPC.AssertExpectations(t)
}

// Block with transfer to precompiled
func TestBlock_363753(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x58ce9",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_363753.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0x8d244c9940f6d54df863fa2abfceb14a2a77f5e61f885dab780e0158435ccc38"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0x8d244c9940f6d54df863fa2abfceb14a2a77f5e61f885dab780e0158435ccc38.json",
			) // nolint
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockReceipts",
		common.HexToHash("0x8d244c9940f6d54df863fa2abfceb14a2a77f5e61f885dab780e0158435ccc38"),
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*[]*types.Receipt)

			file, err := os.ReadFile(
				"testdata/block_receipts_0x8d244c9940f6d54df863fa2abfceb14a2a77f5e61f885dab780e0158435ccc38.json",
			) // nolint
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getHeaderByNumber",
		"0x58ce9",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/block_363753.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	)
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x58ce9",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x58ce9",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x58ce9",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_363753.json")
	assert.NoError(t, err)
	var correctResp *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correctResp))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(363753),
		},
	)
	assert.NoError(t, err)

	// Ensure types match
	jsonResp, err := jsonifyBlock(resp)
	assert.NoError(t, err)
	assert.Equal(t, correctResp.Block, jsonResp)

	mockJSONRPC.AssertExpectations(t)

	// Additional reward distribution check
	var gov map[string]interface{}
	file, err := os.ReadFile("testdata/governance.json")
	assert.NoError(t, err)
	err = json.Unmarshal(file, &gov)
	assert.NoError(t, err)

	minted, ok := gov["reward.mintingamount"].(string)
	assert.True(t, ok)
	blockReward, ok := new(big.Int).SetString(minted, 10)
	assert.True(t, ok)

	var b map[string]interface{}
	file, err = os.ReadFile("testdata/block_363753.json")
	assert.NoError(t, err)
	err = json.Unmarshal(file, &b)
	assert.NoError(t, err)
	txsInBlock, ok := b["transactions"].([]interface{})
	assert.True(t, ok)

	txsFee := big.NewInt(0)
	txs := []string{
		"0x586d0a158f29da3d0e8fa4d24596d1a9f6ded03b5ccdb68f40e9372980488fc8",
		"0x80fb7e6bfa8dae67cf79f21b9e68c5af727ba52f3ab1e5a5be5c8048a9758f56",
	}
	for i, hash := range txs {
		file, err := os.ReadFile(
			"testdata/tx_receipt_" + hash + ".json",
		) // nolint
		assert.NoError(t, err)
		receipt := new(types.Receipt)
		assert.NoError(t, receipt.UnmarshalJSON(file))

		gasUsed := new(big.Int).SetUint64(receipt.GasUsed)
		txInBlock, ok := txsInBlock[i].(map[string]interface{})
		assert.True(t, ok)
		gp, ok := txInBlock["gasPrice"].(string)
		assert.True(t, ok)
		gasPrice, ok := new(big.Int).SetString(strings.Replace(gp, "0x", "", 1), 16)
		assert.True(t, ok)

		txsFee = new(big.Int).Add(txsFee, new(big.Int).Mul(gasUsed, gasPrice))
	}

	totalReward := new(big.Int).Add(blockReward, txsFee)
	expectedCnReward := new(
		big.Int,
	).Div(new(big.Int).Mul(totalReward, big.NewInt(34)), big.NewInt(100))
	expectedKgfReward := new(
		big.Int,
	).Div(new(big.Int).Mul(totalReward, big.NewInt(54)), big.NewInt(100))
	expectedKirReward := new(
		big.Int,
	).Div(new(big.Int).Mul(totalReward, big.NewInt(12)), big.NewInt(100))
	rewardSum := new(
		big.Int,
	).Add(new(big.Int).Add(expectedCnReward, expectedKgfReward), expectedKirReward)
	remainReward := new(big.Int).Sub(totalReward, rewardSum)
	expectedKgfReward = new(big.Int).Add(expectedKgfReward, remainReward)

	cnReward := big.NewInt(0)
	kgfReward := big.NewInt(0)
	kirReward := big.NewInt(0)
	for i, tx := range correctResp.Block.Transactions {
		// "i == 0" means block reward
		if i == 0 {
			r, ok := new(big.Int).SetString(tx.Operations[0].Amount.Value, 10)
			assert.True(t, ok)
			cnReward = new(big.Int).Add(cnReward, r)

			r, ok = new(big.Int).SetString(tx.Operations[1].Amount.Value, 10)
			assert.True(t, ok)
			kgfReward = new(big.Int).Add(kgfReward, r)

			r, ok = new(big.Int).SetString(tx.Operations[2].Amount.Value, 10)
			assert.True(t, ok)
			kirReward = new(big.Int).Add(kirReward, r)
		}
	}

	assert.True(t, cnReward.Cmp(expectedCnReward) == 0)
	assert.True(t, kgfReward.Cmp(expectedKgfReward) == 0)
	assert.True(t, kirReward.Cmp(expectedKirReward) == 0)
}

// Block with complex self-destruct
func TestBlock_468179(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x724d3",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_468179.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0x4fa7e8fc07281539a7aecc0a045ced15916ed28f0605bd3cdab68f22cb5415c2"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0x4fa7e8fc07281539a7aecc0a045ced15916ed28f0605bd3cdab68f22cb5415c2.json",
			) // nolint
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockReceipts",
		common.HexToHash("0x4fa7e8fc07281539a7aecc0a045ced15916ed28f0605bd3cdab68f22cb5415c2"),
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*[]*types.Receipt)

			file, err := os.ReadFile(
				"testdata/block_receipts_0x4fa7e8fc07281539a7aecc0a045ced15916ed28f0605bd3cdab68f22cb5415c2.json",
			) // nolint
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getHeaderByNumber",
		"0x724d3",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/block_468179.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	)
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x724d3",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x724d3",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x724d3",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_468179.json")
	assert.NoError(t, err)
	var correctResp *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correctResp))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(468179),
		},
	)
	assert.NoError(t, err)

	// Ensure types match
	jsonResp, err := jsonifyBlock(resp)
	assert.NoError(t, err)
	assert.Equal(t, correctResp.Block, jsonResp)

	mockJSONRPC.AssertExpectations(t)
}

// Block with complex resurrection
func TestBlock_363366(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x58b66",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_363366.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0xe2cecd530e305403af6184100d1cca6cbfa07ede30d6c01800aea8412691c0c2"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0xe2cecd530e305403af6184100d1cca6cbfa07ede30d6c01800aea8412691c0c2.json",
			) // nolint
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockReceipts",
		common.HexToHash("0xe2cecd530e305403af6184100d1cca6cbfa07ede30d6c01800aea8412691c0c2"),
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*[]*types.Receipt)

			file, err := os.ReadFile(
				"testdata/block_receipts_0xe2cecd530e305403af6184100d1cca6cbfa07ede30d6c01800aea8412691c0c2.json",
			) // nolint
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getHeaderByNumber",
		"0x58b66",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/block_363366.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	)
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x58b66",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x58b66",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x58b66",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_363366.json")
	assert.NoError(t, err)
	var correctResp *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correctResp))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(363366),
		},
	)
	assert.NoError(t, err)

	// Ensure types match
	jsonResp, err := jsonifyBlock(resp)
	assert.NoError(t, err)
	assert.Equal(t, correctResp.Block, jsonResp)

	mockJSONRPC.AssertExpectations(t)

	// Additional reward distribution check
	var gov map[string]interface{}
	file, err := os.ReadFile("testdata/governance.json")
	assert.NoError(t, err)
	err = json.Unmarshal(file, &gov)
	assert.NoError(t, err)

	minted, ok := gov["reward.mintingamount"].(string)
	assert.True(t, ok)
	blockReward, ok := new(big.Int).SetString(minted, 10)
	assert.True(t, ok)

	var b map[string]interface{}
	file, err = os.ReadFile("testdata/block_363366.json")
	assert.NoError(t, err)
	err = json.Unmarshal(file, &b)
	assert.NoError(t, err)
	txsInBlock, ok := b["transactions"].([]interface{})
	assert.True(t, ok)

	txsFee := big.NewInt(0)
	txs := []string{
		"0x3f11ca203c7fd814751725c2c5a3efa00bebbbd5e89f406a28b4a36559393b6f",
		"0x4cc86d845b6ee5c12db00cc75c42e98f8bbf62060bc925942c5ff6a36878549b",
		"0xf8b84ff00db596c9db15de1a44c939cce36c0dfd60ef6171db6951b11d7d015d",
	}
	for i, hash := range txs {
		file, err := os.ReadFile(
			"testdata/tx_receipt_" + hash + ".json",
		) // nolint
		assert.NoError(t, err)
		receipt := new(types.Receipt)
		assert.NoError(t, receipt.UnmarshalJSON(file))

		gasUsed := new(big.Int).SetUint64(receipt.GasUsed)
		txInBlock, ok := txsInBlock[i].(map[string]interface{})
		assert.True(t, ok)
		gp, ok := txInBlock["gasPrice"].(string)
		assert.True(t, ok)
		gasPrice, ok := new(big.Int).SetString(strings.Replace(gp, "0x", "", 1), 16)
		assert.True(t, ok)

		txsFee = new(big.Int).Add(txsFee, new(big.Int).Mul(gasUsed, gasPrice))
	}

	totalReward := new(big.Int).Add(blockReward, txsFee)
	expectedCnReward := new(
		big.Int,
	).Div(new(big.Int).Mul(totalReward, big.NewInt(34)), big.NewInt(100))
	expectedKgfReward := new(
		big.Int,
	).Div(new(big.Int).Mul(totalReward, big.NewInt(54)), big.NewInt(100))
	expectedKirReward := new(
		big.Int,
	).Div(new(big.Int).Mul(totalReward, big.NewInt(12)), big.NewInt(100))
	rewardSum := new(
		big.Int,
	).Add(new(big.Int).Add(expectedCnReward, expectedKgfReward), expectedKirReward)
	remainReward := new(big.Int).Sub(totalReward, rewardSum)
	expectedKgfReward = new(big.Int).Add(expectedKgfReward, remainReward)

	cnReward := big.NewInt(0)
	kgfReward := big.NewInt(0)
	kirReward := big.NewInt(0)
	for i, tx := range correctResp.Block.Transactions {
		// "i == 0" means block reward
		if i == 0 {
			r, ok := new(big.Int).SetString(tx.Operations[0].Amount.Value, 10)
			assert.True(t, ok)
			cnReward = new(big.Int).Add(cnReward, r)

			r, ok = new(big.Int).SetString(tx.Operations[1].Amount.Value, 10)
			assert.True(t, ok)
			kgfReward = new(big.Int).Add(kgfReward, r)

			r, ok = new(big.Int).SetString(tx.Operations[2].Amount.Value, 10)
			assert.True(t, ok)
			kirReward = new(big.Int).Add(kirReward, r)
		}
	}

	assert.True(t, cnReward.Cmp(expectedCnReward) == 0)
	assert.True(t, kgfReward.Cmp(expectedKgfReward) == 0)
	assert.True(t, kirReward.Cmp(expectedKirReward) == 0)
}

// Block with blackholed funds
func TestBlock_468194(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x724e2",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_468194.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0xd7d24fe1cc82182ba7b8af19c102d54b65e5c14c131ccc24462859df9480a54e"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0xd7d24fe1cc82182ba7b8af19c102d54b65e5c14c131ccc24462859df9480a54e.json",
			) // nolint
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockReceipts",
		common.HexToHash("0xd7d24fe1cc82182ba7b8af19c102d54b65e5c14c131ccc24462859df9480a54e"),
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*[]*types.Receipt)

			file, err := os.ReadFile(
				"testdata/block_receipts_0xd7d24fe1cc82182ba7b8af19c102d54b65e5c14c131ccc24462859df9480a54e.json",
			) // nolint
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getHeaderByNumber",
		"0x724e2",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/block_468194.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	)
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x724e2",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x724e2",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x724e2",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_468194.json")
	assert.NoError(t, err)
	var correctResp *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correctResp))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(468194),
		},
	)
	assert.NoError(t, err)

	// Ensure types match
	jsonResp, err := jsonifyBlock(resp)

	assert.NoError(t, err)
	assert.Equal(t, correctResp.Block, jsonResp)

	mockJSONRPC.AssertExpectations(t)
}

// Block with Magma hard fork dynamic gas pricing txs.
func TestBlock_1078(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x436",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_1078.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0x9ad608123d4c896c57a7dbdd7560873eb988a083cd43401f8be157700c268d68"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0x9ad608123d4c896c57a7dbdd7560873eb988a083cd43401f8be157700c268d68.json",
			) // nolint
			assert.NoError(t, err)
			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockReceipts",
		common.HexToHash("0x9ad608123d4c896c57a7dbdd7560873eb988a083cd43401f8be157700c268d68"),
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*[]*types.Receipt)

			file, err := os.ReadFile(
				"testdata/block_receipts_0x9ad608123d4c896c57a7dbdd7560873eb988a083cd43401f8be157700c268d68.json",
			) // nolint
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getHeaderByNumber",
		"0x436",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/block_1078.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	)
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x436",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x436",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x436",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_1078.json")
	assert.NoError(t, err)
	var correctResp *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correctResp))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(1078),
		},
	)
	assert.NoError(t, err)

	// Ensure types match
	jsonResp, err := jsonifyBlock(resp)

	assert.NoError(t, err)
	assert.Equal(t, correctResp.Block, jsonResp)

	mockJSONRPC.AssertExpectations(t)
}

func TestPendingNonceAt(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getTransactionCount",
		common.HexToAddress("0xfFC614eE978630D7fB0C06758DeB580c152154d3"),
		"pending",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*hexutil.Uint64)

			*r = hexutil.Uint64(10)
		},
	).Once()
	resp, err := c.PendingNonceAt(
		ctx,
		common.HexToAddress("0xfFC614eE978630D7fB0C06758DeB580c152154d3"),
	)
	assert.Equal(t, uint64(10), resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestSuggestGasPrice(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_gasPrice",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*hexutil.Big)

			*r = *(*hexutil.Big)(big.NewInt(100000))
		},
	).Once()
	resp, err := c.SuggestGasPrice(
		ctx,
	)
	assert.Equal(t, big.NewInt(100000), resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

func TestSendTransaction(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_sendRawTransaction",
		"0xf86a80843b9aca00825208941ff502f9fe838cd772874cb67d0d96b93fd1d6d78725d4b6199a415d8029a01d110bf9fd468f7d00b3ce530832e99818835f45e9b08c66f8d9722264bb36c7a02711f47ec99f9ac585840daef41b7118b52ec72f02fcb30d874d36b10b668b59", // nolint
	).Return(
		nil,
	).Once()

	rawTx, err := os.ReadFile("testdata/submitted_tx.json")
	assert.NoError(t, err)

	tx := new(types.Transaction)
	assert.NoError(t, tx.UnmarshalJSON(rawTx))

	assert.NoError(t, c.SendTransaction(
		ctx,
		tx,
	))

	mockJSONRPC.AssertExpectations(t)
}

func TestGetMempool(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}
	ctx := context.Background()
	expectedMempool := &RosettaTypes.MempoolResponse{
		TransactionIdentifiers: []*RosettaTypes.TransactionIdentifier{
			{Hash: "0x994024ef9f05d1cb25d01572642c1f550c78d214a52c306bb100d22c025b59d4"},
			{Hash: "0x0859cb844087a280fa031ce2a0879f4f9832f431d6de67f57d0ca32e90dd9e21"},
			{Hash: "0xa6de13e0a4465c9b55726d0826d020ed179fa1bda882a00e90aa467266af1815"},
			{Hash: "0x94e28f01121fd56dead7b89c46c8d6ba32bb79dad5f7e29c1d523224479f10f4"},
			{Hash: "0xf98b8a6589fa27f66545f31d443140c16eaa3da3896e66c4f05a03defa12cda4"},
			{Hash: "0x4c652b9e779277f06e17b6a4c4db6658ac012f7c93f5eeab80e7802cce3ff556"},
			{Hash: "0xa81f636c4b47831efd324dce5c48fe3a3d7a4f0020fae063887e8c2765ff1088"},
			{Hash: "0x3bc19098a49974002ba933ac8f9c753ce5af538ab93fc68586849e8b8b0dce80"},
			{Hash: "0x3401ec802cbe4b02d5be18717b53bb8177bd746f95a54da4674d7c27620facda"},
			{Hash: "0x3d1c58eced8f15f3224e9b2579420d078dbd2adba8e825cee91fd306b0943f89"},
			{Hash: "0xda591f0b15423aedb52f6b0e778b1fbc6757547d69277e2ba1aa7093583d1efb"},
			{Hash: "0xb39652e66c57a6693e44b92b5c471952c26cabf9442a9a76c372f8690658cb4c"},
			{Hash: "0x1e8700bf7215b2da0cfadcf34717f387577254e0871d828e5453a0593ce8060f"},
			{Hash: "0x83811383fb7840a03c25a7cbae7e9af138b17853563eb9e212727be2d0b9667f"},
			{Hash: "0x1e53751e1312cae3324a6b36c67dc95bfec993d7b4939c0de8c0dc761a0afd31"},
			{Hash: "0xc1052f9378db5a779c42ae2de9a0b94c8a6357c815446d6ba55485dcc1b187ef"},
		},
	}

	c := &Client{
		c:              mockJSONRPC,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	mockJSONRPC.On(
		"CallContext", ctx, mock.Anything, "txpool_content",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r, ok := args.Get(1).(*txPoolContentResponse)
			assert.True(t, ok)

			file, err := os.ReadFile("testdata/txpool_content.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	actualMempool, err := c.GetMempool(ctx)
	assert.NoError(t, err)

	assert.Len(t, actualMempool.TransactionIdentifiers, len(expectedMempool.TransactionIdentifiers))

	// Sort both slices to compare later
	sort.Slice(expectedMempool.TransactionIdentifiers, func(i, j int) bool {
		return expectedMempool.TransactionIdentifiers[i].Hash < expectedMempool.TransactionIdentifiers[j].Hash
	})

	sort.Slice(actualMempool.TransactionIdentifiers, func(i, j int) bool {
		return actualMempool.TransactionIdentifiers[i].Hash < actualMempool.TransactionIdentifiers[j].Hash
	})

	assert.True(t, reflect.DeepEqual(actualMempool, expectedMempool))

	mockJSONRPC.AssertExpectations(t)
}

// Block with various Klaytn tx types.
func TestBlock_335049(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x51cc9",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_335049.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0xe9562bcab826324b0241052f8d866d6943a18b4ce8ab7d777daa0cefbec559a6"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0xe9562bcab826324b0241052f8d866d6943a18b4ce8ab7d777daa0cefbec559a6.json",
			) // nolint
			assert.NoError(t, err)
			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockReceipts",
		common.HexToHash("0xe9562bcab826324b0241052f8d866d6943a18b4ce8ab7d777daa0cefbec559a6"),
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*[]*types.Receipt)

			file, err := os.ReadFile(
				"testdata/block_receipts_0xe9562bcab826324b0241052f8d866d6943a18b4ce8ab7d777daa0cefbec559a6.json",
			) // nolint
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getHeaderByNumber",
		"0x51cc9",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/block_335049.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	)
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x51cc9",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x51cc9",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x51cc9",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_335049.json")
	assert.NoError(t, err)
	var correctResp *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correctResp))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(335049),
		},
	)
	assert.NoError(t, err)

	// Ensure types match
	jsonResp, err := jsonifyBlock(resp)
	assert.NoError(t, err)
	assert.Equal(t, correctResp.Block, jsonResp)

	mockJSONRPC.AssertExpectations(t)
}

// Block with complicate reward distribution.
func TestBlock_1665(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x681",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_1665.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0x0f7c5e802a8ebf79611a04b8780bb9c8bc1c00f41c4f829f470b7b6c8fb0d5a6"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0x0f7c5e802a8ebf79611a04b8780bb9c8bc1c00f41c4f829f470b7b6c8fb0d5a6.json",
			) // nolint
			assert.NoError(t, err)
			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockReceipts",
		common.HexToHash("0x0f7c5e802a8ebf79611a04b8780bb9c8bc1c00f41c4f829f470b7b6c8fb0d5a6"),
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*[]*types.Receipt)

			file, err := os.ReadFile(
				"testdata/block_receipts_0x0f7c5e802a8ebf79611a04b8780bb9c8bc1c00f41c4f829f470b7b6c8fb0d5a6.json",
			) // nolint
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getHeaderByNumber",
		"0x681",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/block_1665.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	)
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x681",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x681",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance_1665.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x681",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_1665.json")
	assert.NoError(t, err)
	var correctResp *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correctResp))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(1665),
		},
	)
	assert.NoError(t, err)

	// Ensure types match
	jsonResp, err := jsonifyBlock(resp)

	assert.NoError(t, err)
	assert.Equal(t, correctResp.Block, jsonResp)

	mockJSONRPC.AssertExpectations(t)
}

// Block without KIR and KGF addresses in staking info.
func TestBlock_4219(t *testing.T) {
	mockJSONRPC := &mocks.JSONRPC{}

	tc := testTraceConfig()
	c := &Client{
		c:              mockJSONRPC,
		tc:             tc,
		p:              params.BaobabChainConfig,
		traceSemaphore: semaphore.NewWeighted(100),
	}

	ctx := context.Background()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"klay_getBlockByNumber",
		"0x107b",
		true,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile("testdata/block_4219.json")
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		common.HexToHash("0x8870ecb298ae34ef59f53a240d6dd80f07ccc8fa42065cc872b2a1ed85a1bb93"),
		tc,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := os.ReadFile(
				"testdata/block_trace_0x8870ecb298ae34ef59f53a240d6dd80f07ccc8fa42065cc872b2a1ed85a1bb93.json",
			) // nolint
			assert.NoError(t, err)
			*r = json.RawMessage(file)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getChainConfig",
		"0x107b",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*params.ChainConfig)

			file, err := os.ReadFile("testdata/chainConfig.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)

			assert.NoError(t, err)
		},
	).Once()
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getParams",
		"0x107b",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*map[string]interface{})

			file, err := os.ReadFile("testdata/governance_1665.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"governance_getStakingInfo",
		"0x107b",
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*reward.StakingInfo)

			file, err := os.ReadFile("testdata/stakingInfo_4219.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, r)
			assert.NoError(t, err)
		},
	).Once()

	correctRaw, err := os.ReadFile("testdata/block_response_4219.json")
	assert.NoError(t, err)
	var correctResp *RosettaTypes.BlockResponse
	assert.NoError(t, json.Unmarshal(correctRaw, &correctResp))

	resp, err := c.Block(
		ctx,
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(4219),
		},
	)
	assert.NoError(t, err)

	// Ensure types match
	jsonResp, err := jsonifyBlock(resp)

	assert.NoError(t, err)
	assert.Equal(t, correctResp.Block, jsonResp)

	mockJSONRPC.AssertExpectations(t)
}
