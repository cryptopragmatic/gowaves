package state

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

//go:generate moq -out scripts_storage_moq_test.go . ScriptStorageState:MockScriptStorageState
type ScriptStorageState interface {
	setScript(scriptType blockchainEntity, key []byte, record scriptRecord, blockID proto.BlockID) error
	scriptBytesByKey(key []byte, filter bool) (proto.Script, error)
	newestScriptBytesByKey(key []byte, filter bool) (proto.Script, error)
	scriptAstFromRecordBytes(recordBytes []byte) (*ride.Tree, crypto.PublicKey, error)
	newestScriptAstByKey(key []byte, filter bool) (*ride.Tree, error)
	scriptTreeByKey(key []byte, filter bool) (*ride.Tree, error)
	commitUncertain(blockID proto.BlockID) error
	dropUncertain()
	setAssetScriptUncertain(assetID proto.AssetID, script proto.Script, pk crypto.PublicKey)
	setAssetScript(assetID proto.AssetID, script proto.Script, pk crypto.PublicKey, blockID proto.BlockID) error
	newestIsSmartAsset(assetID proto.AssetID, filter bool) bool
	isSmartAsset(assetID proto.AssetID, filter bool) (bool, error)
	newestScriptByAsset(assetID proto.AssetID, filter bool) (*ride.Tree, error)
	scriptByAsset(assetID proto.AssetID, filter bool) (*ride.Tree, error)
	scriptBytesByAsset(assetID proto.AssetID, filter bool) (proto.Script, error)
	newestScriptBytesByAsset(assetID proto.AssetID, filter bool) (proto.Script, error)
	setAccountScript(addr proto.WavesAddress, script proto.Script, pk crypto.PublicKey, blockID proto.BlockID) error
	newestAccountHasVerifier(addr proto.WavesAddress, filter bool) (bool, error)
	accountHasVerifier(addr proto.WavesAddress, filter bool) (bool, error)
	newestAccountHasScript(addr proto.WavesAddress, filter bool) (bool, error)
	accountHasScript(addr proto.WavesAddress, filter bool) (bool, error)
	newestScriptByAddr(addr proto.WavesAddress, filter bool) (*ride.Tree, error)
	NewestScriptPKByAddr(addr proto.WavesAddress, filter bool) (crypto.PublicKey, error)
	scriptByAddr(addr proto.WavesAddress, filter bool) (*ride.Tree, error)
	scriptBytesByAddr(addr proto.WavesAddress, filter bool) (proto.Script, error)
	clear() error
	prepareHashes() error
	reset()
	AccountScriptsHasher() *stateHasher
	AssetScriptsHasher() *stateHasher
}