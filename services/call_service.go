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
// Modified and improved for the Kaia development.

package services

import (
	"context"
	"errors"

	"github.com/kaiachain/rosetta-kaia/kaia"

	"github.com/kaiachain/rosetta-kaia/configuration"
	"github.com/klaytn/rosetta-sdk-go-klaytn/types"
)

// CallAPIService implements the server.CallAPIServicer interface.
type CallAPIService struct {
	config *configuration.Configuration
	client Client
}

// NewCallAPIService creates a new instance of a CallAPIService.
func NewCallAPIService(cfg *configuration.Configuration, client Client) *CallAPIService {
	return &CallAPIService{
		config: cfg,
		client: client,
	}
}

// Call implements the /call endpoint.
func (s *CallAPIService) Call(
	ctx context.Context,
	request *types.CallRequest,
) (*types.CallResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, ErrUnavailableOffline
	}

	response, err := s.client.Call(ctx, request)
	if errors.Is(err, kaia.ErrCallParametersInvalid) {
		return nil, wrapErr(ErrCallParametersInvalid, err)
	}
	if errors.Is(err, kaia.ErrCallOutputMarshal) {
		return nil, wrapErr(ErrCallOutputMarshal, err)
	}
	if errors.Is(err, kaia.ErrCallMethodInvalid) {
		return nil, wrapErr(ErrCallMethodInvalid, err)
	}
	if err != nil {
		return nil, wrapErr(ErrClient, err)
	}

	return response, nil
}
