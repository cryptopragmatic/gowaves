package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Debug struct {
	options Options
}

func NewDebug(options Options) *Debug {
	return &Debug{
		options: options,
	}
}

type DebugInfo struct {
	StateHeight              uint64 `json:"stateHeight"`
	ExtensionLoaderState     string `json:"extensionLoaderState"`
	HistoryReplierCacheSizes struct {
		Blocks      uint64 `json:"blocks"`
		MicroBlocks uint64 `json:"microBlocks"`
	} `json:"historyReplierCacheSizes"`
	MicroBlockSynchronizerCacheSizes struct {
		MicroBlockOwners     uint64 `json:"microBlockOwners"`
		NextInvs             uint64 `json:"nextInvs"`
		Awaiting             uint64 `json:"awaiting"`
		SuccessfullyReceived uint64 `json:"successfullyReceived"`
	} `json:"microBlockSynchronizerCacheSizes"`
	ScoreObserverStats struct {
		LocalScore         LocalScore `json:"localScore"`
		CurrentBestChannel string     `json:"currentBestChannel"`
		ScoresCacheSize    uint64     `json:"scoresCacheSize"`
	} `json:"scoreObserverStats"`
	MinerState string `json:"minerState"`
}

type LocalScore string

func (a *LocalScore) UnmarshalJSON(data []byte) error {
	*a = LocalScore(data)
	return nil
}

// Info returns all info you need to debug.
func (a *Debug) Info(ctx context.Context) (*DebugInfo, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/debug/info")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(DebugInfo)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// Blocks gets sizes and full hashes for last blocks.
func (a *Debug) Blocks(ctx context.Context, howMany uint64) ([]map[uint64]string, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/debug/blocks/%d", howMany))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	var out []map[uint64]string
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type DebugMinerInfo struct {
	Address       proto.WavesAddress `json:"address"`
	MiningBalance uint64             `json:"miningBalance"`
	Timestamp     uint64             `json:"timestamp"`
}

// MinerInfo gets all miner info you need to debug.
func (a *Debug) MinerInfo(ctx context.Context) ([]*DebugMinerInfo, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/debug/minerInfo")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set(ApiKeyHeader, a.options.ApiKey)

	var out []*DebugMinerInfo
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type DebugHistoryInfo struct {
	LastBlockIds  []proto.BlockID `json:"lastBlockIds"`
	MicroBlockIds []proto.BlockID `json:"microBlockIds"`
}

// HistoryInfo gets all history info you need to debug.
func (a *Debug) HistoryInfo(ctx context.Context) (*DebugHistoryInfo, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/debug/historyInfo")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set(ApiKeyHeader, a.options.ApiKey)

	out := new(DebugHistoryInfo)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// ConfigInfo currently running node config.
func (a *Debug) ConfigInfo(ctx context.Context, full bool) ([]byte, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/debug/configInfo?full=%t", full))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set(ApiKeyHeader, a.options.ApiKey)

	buf := new(bytes.Buffer)
	response, err := doHttp(ctx, a.options, req, buf)
	if err != nil {
		return nil, response, err
	}

	return buf.Bytes(), response, nil
}

func (a *Debug) StateHash(ctx context.Context, height uint64) (*proto.StateHash, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/debug/stateHash/%d", height))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	out := &proto.StateHash{}
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type BalancesHistoryRow struct {
	Height  uint64 `json:"height"`
	Balance uint64 `json:"balance"`
}

func (a *Debug) BalancesHistory(ctx context.Context, address proto.WavesAddress) ([]*BalancesHistoryRow, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/debug/balances/history/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var out []*BalancesHistoryRow
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

type StateChangesResponse struct {
	Id           string `json:"id"`
	Height       uint64 `json:"height"`
	StateChanges struct {
		Data      proto.DataEntries `json:"data"`
		Transfers []*struct {
			Address proto.WavesAddress `json:"address"`
			Asset   crypto.Digest      `json:"asset"`
			Amount  uint64             `json:"amount"`
		} `json:"transfers"`
	} `json:"stateChanges"`
}

func (a *Debug) StateChanges(ctx context.Context, id crypto.Digest) (*StateChangesResponse, *Response, error) {
	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/debug/stateChanges/info/%s", id.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var out StateChangesResponse
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return nil, response, err
	}

	return &out, response, nil
}

func (a *Debug) PrintMsg(ctx context.Context, msg string) (*Response, error) {
	type printMsgRequestBody struct {
		Message string `json:"message"`
	}

	url, err := joinUrl(a.options.BaseUrl, "/debug/print")
	if err != nil {
		return nil, err
	}
	bts, err := json.Marshal(printMsgRequestBody{Message: msg})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(bts))
	if err != nil {
		return nil, err
	}
	req.Header.Add(ApiKeyHeader, a.options.ApiKey)
	req.Header.Add("Accept", "*/*")

	return doHttp(ctx, a.options, req, nil)
}
