package state

import (
	"encoding/base64"
	"encoding/hex"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/types"
	"math/big"
	"strings"
	"testing"
)

func defaultTxAppender(t *testing.T, storage ScriptStorageState, state types.SmartState, scheme proto.Scheme) txAppender {
	var feautures = &MockFeaturesState{
		newestIsActivatedFunc: func(featureID int16) (bool, error) {
			if featureID == int16(settings.SmartAccounts) {
				return true, nil
			}
			if featureID == int16(settings.Ride4DApps) {
				return true, nil
			}
			return false, nil
		},
	}

	assetID := proto.AssetIDFromDigest(proto.NewOptionalAssetWaves().ID)
	assetsUncertain := make(map[proto.AssetID]assetInfo)
	assetsUncertain[assetID] = assetInfo{}

	store := blockchainEntitiesStorage{features: feautures, scriptsStorage: storage, sponsoredAssets: &sponsoredAssets{features: feautures, settings: &settings.BlockchainSettings{}}, assets: &assets{uncertainAssetInfo: assetsUncertain}}
	blockchainSettings := &settings.BlockchainSettings{FunctionalitySettings: settings.FunctionalitySettings{CheckTempNegativeAfterTime: 1, AllowLeasedBalanceTransferUntilTime: 1, AddressSchemeCharacter: scheme}}
	txHandler, err := newTransactionHandler(genBlockId('1'), &store, blockchainSettings, state)
	assert.NoError(t, err)
	blockchainEntitiesStor := blockchainEntitiesStorage{scriptsStorage: storage}
	txAppender := txAppender{
		txHandler:   txHandler,
		stor:        &store,
		state:       state,
		blockDiffer: &blockDiffer{handler: txHandler, settings: &settings.BlockchainSettings{}},
		ia:          &invokeApplier{sc: &scriptCaller{stor: &store, state: state, settings: blockchainSettings}, blockDiffer: &blockDiffer{stor: &store, handler: txHandler, settings: blockchainSettings}, state: state, txHandler: txHandler, settings: blockchainSettings, stor: &blockchainEntitiesStor, invokeDiffStor: &diffStorageWrapped{invokeDiffsStor: &diffStorage{changes: []balanceChanges{}}}},
	}
	return txAppender
}
func defaultEthereumLegacyTxData(value int64, to *proto.EthereumAddress, data []byte, gas uint64) *proto.EthereumLegacyTx {
	v := big.NewInt(87) // MainNet byte
	v.Mul(v, big.NewInt(2))
	v.Add(v, big.NewInt(35))

	return &proto.EthereumLegacyTx{
		Value:    big.NewInt(value),
		To:       to,
		Data:     data,
		GasPrice: big.NewInt(1),
		Nonce:    1479168000000,
		Gas:      gas,
		V:        v,
	}
}
func TestEthereumTransferWaves(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	txAppender := defaultTxAppender(t, nil, nil, proto.MainNetScheme)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	tx := proto.EthereumTransaction{
		Inner:    defaultEthereumLegacyTxData(1000000000000000, &recipientEth, nil, 100000),
		ID:       nil,
		SenderPK: &senderPK,
	}
	tx.TxKind, err = txAppender.guessEthereumTransactionKind(&tx, nil)
	assert.NoError(t, err)
	applRes, err := txAppender.handleDefaultTransaction(&tx, appendTxParams, false)
	assert.NoError(t, err)
	assert.True(t, applRes.status)
	sender, err := tx.SenderPK.EthereumAddress().ToWavesAddress(proto.MainNetScheme)
	assert.NoError(t, err)
	wavesAsset := proto.NewOptionalAssetWaves()
	senderKey := byteKey(sender.ID(), wavesAsset)
	recipient, err := recipientEth.ToWavesAddress(proto.MainNetScheme)
	assert.NoError(t, err)
	recipientKey := byteKey(recipient.ID(), wavesAsset)
	senderBalance := applRes.changes.diff[string(senderKey)].balance
	recipientBalance := applRes.changes.diff[string(recipientKey)].balance
	assert.Equal(t, senderBalance, int64(-200000))
	assert.Equal(t, recipientBalance, int64(100000))

}
func lessenDecodedDataAmount(t *testing.T, decodedData *ethabi.DecodedCallData) {
	v, ok := decodedData.Inputs[1].Value.(ethabi.BigInt)
	assert.True(t, ok)
	res := new(big.Int).Div(v.V, big.NewInt(int64(proto.DiffEthWaves)))
	decodedData.Inputs[1].Value = ethabi.BigInt{V: res}
}

func TestEthereumTransferAssets(t *testing.T) {
	storage := &MockScriptStorageState{
		NewestScriptPKByAddrFunc: func(address proto.WavesAddress, filter bool) (crypto.PublicKey, error) {
			return crypto.NewPublicKeyFromBase58("pmDSxpnULiroUAerTDFBajffTpqgwVJjtMipQq6DQM5")
		},
		newestIsSmartAssetFunc: func(assetID proto.AssetID, filter bool) (bool, error) {
			return false, nil
		},
	}

	appendTxParams := defaultAppendTxParams()
	state := &AnotherMockSmartState{
		AssetInfoByIDFunc: func(id proto.AssetID, filter bool) (*proto.AssetInfo, error) {
			var r crypto.Digest
			copy(r[:20], id[:])
			return &proto.AssetInfo{ID: r}, nil
		},
	}
	txAppender := defaultTxAppender(t, storage, state, proto.MainNetScheme)

	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)
	//var TxSeveralData []proto.EthereumTxData
	//TxSeveralData = append(TxSeveralData, defaultEthereumLegacyTxData(1000000000000000, &recipientEth), defaultEthereumDynamicFeeTx(1000000000000000, &recipientEth), defaultEthereumAccessListTx(1000000000000000, &recipientEth))
	/*
		from https://etherscan.io/tx/0x363f979b58c82614db71229c2a57ed760e7bc454ee29c2f8fd1df99028667ea5
		transfer(address,uint256)
		1 = 0x9a1989946ae4249AAC19ac7a038d24Aab03c3D8c
		2 = 209470300000000000000000
	*/
	hexdata := "0xa9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c000000000000000000000000000000000000000000002c5b68601cc92ad60000"
	data, err := hex.DecodeString(strings.TrimPrefix(hexdata, "0x"))
	require.NoError(t, err)
	var txData proto.EthereumTxData = defaultEthereumLegacyTxData(1000000000000000, &recipientEth, data, 100000)
	tx := proto.EthereumTransaction{
		Inner:    txData,
		ID:       nil,
		SenderPK: &senderPK,
	}
	db := ethabi.NewErc20MethodsMap()
	assert.NotNil(t, tx.Data())
	decodedData, err := db.ParseCallDataRide(tx.Data())
	assert.NoError(t, err)
	lessenDecodedDataAmount(t, decodedData)
	tx.TxKind, err = txAppender.guessEthereumTransactionKind(&tx, appendTxParams)
	assert.NoError(t, err)
	txKindTransferAssets := tx.TxKind.(*proto.EthereumTransferAssetsErc20TxKind)
	tx.TxKind = proto.NewEthereumTransferAssetsErc20TxKind(*decodedData, txKindTransferAssets.Asset)
	applRes, err := txAppender.handleDefaultTransaction(&tx, appendTxParams, false)
	assert.NoError(t, err)
	assert.True(t, applRes.status)
	sender, err := tx.SenderPK.EthereumAddress().ToWavesAddress(proto.MainNetScheme)
	assert.NoError(t, err)
	wavesAsset := proto.NewOptionalAssetWaves()
	senderWavesKey := byteKey(sender.ID(), wavesAsset)
	senderKey := byteKey(sender.ID(), txKindTransferAssets.Asset)

	//rideValue, err := ride.EthABIDataTypeToRideType(decodedData.Inputs[0].Value)
	//assert.NoError(t, err)
	//rideEthRecipientAddress, ok := rideValue.(ride.rideBytes)
	rideEthRecipientAddress, ok := decodedData.Inputs[0].Value.(ethabi.Bytes)
	assert.True(t, ok)
	ethRecipientAddress := proto.BytesToEthereumAddress(rideEthRecipientAddress)
	recipient, err := ethRecipientAddress.ToWavesAddress(proto.MainNetScheme)
	assert.NoError(t, err)
	recipientKey := byteKey(recipient.ID(), txKindTransferAssets.Asset)
	senderWavesBalance := applRes.changes.diff[string(senderWavesKey)].balance
	senderBalance := applRes.changes.diff[string(senderKey)].balance
	recipientBalance := applRes.changes.diff[string(recipientKey)].balance
	assert.Equal(t, senderWavesBalance, int64(-100000))
	assert.Equal(t, senderBalance, int64(-20947030000000))
	assert.Equal(t, recipientBalance, int64(20947030000000))
}
func defaultDecodedData(name string, arguments []ethabi.DecodedArg, payments []ethabi.Payment) ethabi.DecodedCallData {
	var decodedData ethabi.DecodedCallData
	decodedData.Name = name
	decodedData.Inputs = arguments
	decodedData.Payments = payments
	return decodedData
}
func applyScript(t *testing.T, tx *proto.EthereumTransaction, stor ScriptStorageState, info *fallibleValidationParams) (proto.WavesAddress, *ride.Tree) {
	scriptAddr, err := tx.WavesAddressTo(0)
	require.NoError(t, err)
	tree, err := stor.newestScriptByAddr(*scriptAddr, !info.initialisation)
	require.NoError(t, err)
	return *scriptAddr, tree
}
func TestEthereumInvoke(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	newestScriptByAddrFunc := func(addr proto.WavesAddress, filter bool) (*ride.Tree, error) {
		/*
			{-# STDLIB_VERSION 4 #-}
			{-# CONTENT_TYPE DAPP #-}
			{-# SCRIPT_TYPE ACCOUNT #-}
			@Callable(i)
			func call(number: Int) = {
			  [
			    IntegerEntry("int", number)
			  ]
			}
		*/
		src, err := base64.StdEncoding.DecodeString("AAIEAAAAAAAAAAcIAhIDCgEBAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAEAAAAGbnVtYmVyCQAETAAAAAIJAQAAAAxJbnRlZ2VyRW50cnkAAAACAgAAAANpbnQFAAAABm51bWJlcgUAAAADbmlsAAAAAE5VO+E=")
		require.NoError(t, err)
		tree, err := ride.Parse(src)
		require.NoError(t, err)
		assert.NotNil(t, tree)
		return tree, nil
	}
	storage := &MockScriptStorageState{
		newestScriptByAddrFunc: newestScriptByAddrFunc,
		scriptByAddrFunc:       newestScriptByAddrFunc,
		NewestScriptPKByAddrFunc: func(address proto.WavesAddress, filter bool) (crypto.PublicKey, error) {
			return crypto.NewPublicKeyFromBase58("pmDSxpnULiroUAerTDFBajffTpqgwVJjtMipQq6DQM5")
		},
		newestIsSmartAssetFunc: func(assetID proto.AssetID, filter bool) (bool, error) {
			return false, nil
		},
	}
	state := &AnotherMockSmartState{
		AssetInfoByIDFunc: func(id proto.AssetID, filter bool) (*proto.AssetInfo, error) {
			var r crypto.Digest
			copy(r[:20], id[:])
			return &proto.AssetInfo{ID: r}, nil
		},
		AddingBlockHeightFunc: func() (uint64, error) {
			return 1000, nil
		},
		EstimatorVersionFunc: func() (int, error) {
			return 3, nil
		},
	}
	txAppender := defaultTxAppender(t, storage, state, proto.MainNetScheme)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	sender, err := senderPK.EthereumAddress().ToWavesAddress(0)
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)
	var txData proto.EthereumTxData = defaultEthereumLegacyTxData(1000000000000000, &recipientEth, nil, 500000)
	tx := proto.EthereumTransaction{
		Inner:    txData,
		ID:       &crypto.Digest{},
		SenderPK: &senderPK,
	}
	decodedData := defaultDecodedData("call", []ethabi.DecodedArg{{Value: ethabi.Int(10)}}, []ethabi.Payment{{Amount: 5, AssetID: proto.NewOptionalAssetWaves().ID}})
	tx.TxKind, err = txAppender.guessEthereumTransactionKind(&tx, appendTxParams)
	tx.TxKind = proto.NewEthereumInvokeScriptTxKind(decodedData)

	assert.NoError(t, err)
	fallibleInfo := &fallibleValidationParams{appendTxParams: appendTxParams, senderScripted: false, senderAddress: sender}
	scriptAddress, tree := applyScript(t, &tx, storage, fallibleInfo)
	fallibleInfo.rideV5Activated = true
	ok, actions, err := txAppender.ia.sc.invokeFunction(tree, &tx, fallibleInfo, scriptAddress, *tx.ID)
	assert.NoError(t, err)
	assert.True(t, ok)

	_, err = txAppender.ia.txHandler.checkTx(&tx, fallibleInfo.checkerInfo)
	assert.NoError(t, err)
	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "int", Value: 10}},
	}
	assert.Equal(t, expectedDataEntryWrites[0], actions[0])

	// fee test
	var txDataForFeeCheck proto.EthereumTxData = defaultEthereumLegacyTxData(1000000000000000, &recipientEth, nil, 499999)
	tx.Inner = txDataForFeeCheck
	_, err = txAppender.ia.txHandler.checkTx(&tx, fallibleInfo.checkerInfo)
	require.Error(t, err)
}

func TestTransferZeroAmount(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	txAppender := defaultTxAppender(t, nil, nil, proto.MainNetScheme)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	tx := proto.EthereumTransaction{
		Inner:    defaultEthereumLegacyTxData(0, &recipientEth, nil, 100000),
		ID:       nil,
		SenderPK: &senderPK,
	}
	tx.TxKind, err = txAppender.guessEthereumTransactionKind(&tx, nil)
	assert.NoError(t, err)
	_, err = txAppender.handleDefaultTransaction(&tx, appendTxParams, false)
	require.Error(t, err)
}

func TestTransferMainNetTestnet(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	txAppender := defaultTxAppender(t, nil, nil, proto.TestNetScheme)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	tx := proto.EthereumTransaction{
		Inner:    defaultEthereumLegacyTxData(100, &recipientEth, nil, 100000),
		ID:       nil,
		SenderPK: &senderPK,
	}
	tx.TxKind, err = txAppender.guessEthereumTransactionKind(&tx, nil)
	assert.NoError(t, err)
	_, err = txAppender.handleDefaultTransaction(&tx, appendTxParams, false)
	require.Error(t, err)
}

func TestTransferCheckFee(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	txAppender := defaultTxAppender(t, nil, nil, proto.TestNetScheme)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("a783d1CBABe28d25E64aDf84477C4687c1411f94") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)

	tx := proto.EthereumTransaction{
		Inner:    defaultEthereumLegacyTxData(100, &recipientEth, nil, 100),
		ID:       nil,
		SenderPK: &senderPK,
	}
	tx.TxKind, err = txAppender.guessEthereumTransactionKind(&tx, nil)
	assert.NoError(t, err)
	_, err = txAppender.handleDefaultTransaction(&tx, appendTxParams, false)
	require.Error(t, err)
}

func TestEthereumInvokeWithoutPaymentsAndArguments(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	newestScriptByAddrFunc := func(addr proto.WavesAddress, filter bool) (*ride.Tree, error) {
		/*
			{-# STDLIB_VERSION 4 #-}
			{-# CONTENT_TYPE DAPP #-}
			{-# SCRIPT_TYPE ACCOUNT #-}
			@Callable(i)
			func call() = {
			  [
			    IntegerEntry("int", 1)
			  ]
			}
		*/
		src, err := base64.StdEncoding.DecodeString("AAIEAAAAAAAAAAQIAhIAAAAAAAAAAAEAAAABaQEAAAAEY2FsbAAAAAAJAARMAAAAAgkBAAAADEludGVnZXJFbnRyeQAAAAICAAAAA2ludAAAAAAAAAAAAQUAAAADbmlsAAAAAOhqG0I=")
		require.NoError(t, err)
		tree, err := ride.Parse(src)
		require.NoError(t, err)
		assert.NotNil(t, tree)
		return tree, nil
	}
	storage := &MockScriptStorageState{
		newestScriptByAddrFunc: newestScriptByAddrFunc,
		scriptByAddrFunc:       newestScriptByAddrFunc,
		NewestScriptPKByAddrFunc: func(address proto.WavesAddress, filter bool) (crypto.PublicKey, error) {
			return crypto.NewPublicKeyFromBase58("pmDSxpnULiroUAerTDFBajffTpqgwVJjtMipQq6DQM5")
		},
		newestIsSmartAssetFunc: func(assetID proto.AssetID, filter bool) (bool, error) {
			return false, nil
		},
	}
	state := &AnotherMockSmartState{
		AssetInfoByIDFunc: func(id proto.AssetID, filter bool) (*proto.AssetInfo, error) {
			var r crypto.Digest
			copy(r[:20], id[:])
			return &proto.AssetInfo{ID: r}, nil
		},
		AddingBlockHeightFunc: func() (uint64, error) {
			return 1000, nil
		},
		EstimatorVersionFunc: func() (int, error) {
			return 3, nil
		},
	}
	txAppender := defaultTxAppender(t, storage, state, proto.MainNetScheme)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	sender, err := senderPK.EthereumAddress().ToWavesAddress(0)
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)
	var txData proto.EthereumTxData = defaultEthereumLegacyTxData(1000000000000000, &recipientEth, nil, 500000)
	tx := proto.EthereumTransaction{
		Inner:    txData,
		ID:       &crypto.Digest{},
		SenderPK: &senderPK,
	}
	decodedData := defaultDecodedData("call", nil, nil)
	tx.TxKind, err = txAppender.guessEthereumTransactionKind(&tx, appendTxParams)
	tx.TxKind = proto.NewEthereumInvokeScriptTxKind(decodedData)

	assert.NoError(t, err)
	fallibleInfo := &fallibleValidationParams{appendTxParams: appendTxParams, senderScripted: false, senderAddress: sender}
	scriptAddress, tree := applyScript(t, &tx, storage, fallibleInfo)
	fallibleInfo.rideV5Activated = true
	ok, actions, err := txAppender.ia.sc.invokeFunction(tree, &tx, fallibleInfo, scriptAddress, *tx.ID)
	assert.NoError(t, err)
	assert.True(t, ok)

	_, err = txAppender.ia.txHandler.checkTx(&tx, fallibleInfo.checkerInfo)
	assert.NoError(t, err)
	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "int", Value: 1}},
	}
	assert.Equal(t, expectedDataEntryWrites[0], actions[0])

}

func TestEthereumInvokeAllArguments(t *testing.T) {
	appendTxParams := defaultAppendTxParams()
	newestScriptByAddrFunc := func(addr proto.WavesAddress, filter bool) (*ride.Tree, error) {
		/*
			{-# STDLIB_VERSION 4 #-}
			{-# CONTENT_TYPE DAPP #-}
			{-# SCRIPT_TYPE ACCOUNT #-}
			@Callable(i)
			func call(num: Int, flag: Boolean, vec:ByteVector, lis: Int|String, list: List[Int]) = {
			  [
				IntegerEntry("int", num)
			  ]
			}
		*/
		src, err := base64.StdEncoding.DecodeString("AAIEAAAAAAAAAAsIAhIHCgUBBAIJEQAAAAAAAAABAAAAAWkBAAAABGNhbGwAAAAFAAAAA251bQAAAARmbGFnAAAAA3ZlYwAAAANsaXMAAAAEbGlzdAkABEwAAAACCQEAAAAMSW50ZWdlckVudHJ5AAAAAgIAAAADaW50BQAAAANudW0FAAAAA25pbAAAAAC7za7+")
		require.NoError(t, err)
		tree, err := ride.Parse(src)
		require.NoError(t, err)
		assert.NotNil(t, tree)
		return tree, nil
	}
	storage := &MockScriptStorageState{
		newestScriptByAddrFunc: newestScriptByAddrFunc,
		scriptByAddrFunc:       newestScriptByAddrFunc,
		NewestScriptPKByAddrFunc: func(address proto.WavesAddress, filter bool) (crypto.PublicKey, error) {
			return crypto.NewPublicKeyFromBase58("pmDSxpnULiroUAerTDFBajffTpqgwVJjtMipQq6DQM5")
		},
		newestIsSmartAssetFunc: func(assetID proto.AssetID, filter bool) (bool, error) {
			return false, nil
		},
	}
	state := &AnotherMockSmartState{
		AssetInfoByIDFunc: func(id proto.AssetID, filter bool) (*proto.AssetInfo, error) {
			var r crypto.Digest
			copy(r[:20], id[:])
			return &proto.AssetInfo{ID: r}, nil
		},
		AddingBlockHeightFunc: func() (uint64, error) {
			return 1000, nil
		},
		EstimatorVersionFunc: func() (int, error) {
			return 3, nil
		},
	}
	txAppender := defaultTxAppender(t, storage, state, proto.MainNetScheme)
	senderPK, err := proto.NewEthereumPublicKeyFromHexString("c4f926702fee2456ac5f3d91c9b7aa578ff191d0792fa80b6e65200f2485d9810a89c1bb5830e6618119fb3f2036db47fac027f7883108cbc7b2953539b9cb53")
	assert.NoError(t, err)
	sender, err := senderPK.EthereumAddress().ToWavesAddress(0)
	assert.NoError(t, err)
	recipientBytes, err := base58.Decode("3PFpqr7wTCBu68sSqU7vVv9pttYRjQjGFbv") // 0x241Cf7eaf669E0d2FDe4Ba3a534c20B433F4c43d
	assert.NoError(t, err)
	recipientEth := proto.BytesToEthereumAddress(recipientBytes)
	var txData proto.EthereumTxData = defaultEthereumLegacyTxData(1000000000000000, &recipientEth, nil, 500000)
	tx := proto.EthereumTransaction{
		Inner:    txData,
		ID:       &crypto.Digest{},
		SenderPK: &senderPK,
	}

	decodedData := defaultDecodedData("call", []ethabi.DecodedArg{
		{Value: ethabi.Int(1)},
		{Value: ethabi.Bool(true)},
		{Value: ethabi.Bytes([]byte{2})},
		{Value: ethabi.Bool(true)}, // will leave it here
		{Value: ethabi.List{ethabi.Int(4)}},
	}, nil)

	tx.TxKind, err = txAppender.guessEthereumTransactionKind(&tx, appendTxParams)
	tx.TxKind = proto.NewEthereumInvokeScriptTxKind(decodedData)

	assert.NoError(t, err)
	fallibleInfo := &fallibleValidationParams{appendTxParams: appendTxParams, senderScripted: false, senderAddress: sender}
	scriptAddress, tree := applyScript(t, &tx, storage, fallibleInfo)
	fallibleInfo.rideV5Activated = true
	ok, actions, err := txAppender.ia.sc.invokeFunction(tree, &tx, fallibleInfo, scriptAddress, *tx.ID)
	assert.NoError(t, err)
	assert.True(t, ok)

	_, err = txAppender.ia.txHandler.checkTx(&tx, fallibleInfo.checkerInfo)
	assert.NoError(t, err)
	expectedDataEntryWrites := []*proto.DataEntryScriptAction{
		{Entry: &proto.IntegerDataEntry{Key: "int", Value: 1}},
	}
	assert.Equal(t, expectedDataEntryWrites[0], actions[0])
}