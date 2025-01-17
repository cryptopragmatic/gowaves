package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Consensus struct {
	options Options
}

// NewConsensus creates new consensus api section.
func NewConsensus(options Options) *Consensus {
	return &Consensus{
		options: options,
	}
}

type ConsensusGeneratingBalance struct {
	Address proto.WavesAddress `json:"address"`
	Balance uint64             `json:"balance"`
}

// GeneratingBalance returns account's generating balance (the same as balance atm).
func (a Consensus) GeneratingBalance(ctx context.Context, address proto.WavesAddress) (*ConsensusGeneratingBalance, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/consensus/generatingbalance/%s", address.String()))
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(),
		nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(ConsensusGeneratingBalance)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// GenerationSignatureByBlock returns generation signature of a block with specified ID.
func (a *Consensus) GenerationSignatureByBlock(ctx context.Context, blockID string) (string, *Response, error) {
	if a.options.ApiKey == "" {
		return "", nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/consensus/generationsignature/%s", blockID))
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := make(map[string]string)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return "", response, err
	}

	return out["generationSignature"], response, nil
}

// BaseTargetByBlock gets base target of a block with specified ID.
func (a *Consensus) BaseTargetByBlock(ctx context.Context, blockID string) (uint64, *Response, error) {
	if a.options.ApiKey == "" {
		return 0, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, fmt.Sprintf("/consensus/basetarget/%s", blockID))
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := make(map[string]uint64)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return 0, response, err
	}

	return out["baseTarget"], response, nil
}

type ConsensusBaseTarget struct {
	BaseTarget uint64 `json:"baseTarget"`
	Score      string `json:"score"`
}

// BaseTarget gets base target of the last block.
func (a *Consensus) BaseTarget(ctx context.Context) (*ConsensusBaseTarget, *Response, error) {
	if a.options.ApiKey == "" {
		return nil, nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/consensus/basetarget")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := new(ConsensusBaseTarget)
	response, err := doHttp(ctx, a.options, req, out)
	if err != nil {
		return nil, response, err
	}

	return out, response, nil
}

// Algo shows which consensus algo is in use.
func (a *Consensus) Algo(ctx context.Context) (string, *Response, error) {
	if a.options.ApiKey == "" {
		return "", nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/consensus/algo")
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := make(map[string]string)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return "", response, err
	}

	return out["consensusAlgo"], response, nil
}

// GenerationSignature gets generation signature of the last block.
func (a *Consensus) GenerationSignature(ctx context.Context) (string, *Response, error) {
	if a.options.ApiKey == "" {
		return "", nil, NoApiKeyError
	}

	url, err := joinUrl(a.options.BaseUrl, "/consensus/generationsignature")
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("X-API-Key", a.options.ApiKey)

	out := make(map[string]string)
	response, err := doHttp(ctx, a.options, req, &out)
	if err != nil {
		return "", response, err
	}

	return out["generationSignature"], response, nil
}
