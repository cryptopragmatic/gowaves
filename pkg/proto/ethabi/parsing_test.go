package ethabi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	"sort"
	"strings"
	"testing"
)

// TODO(nickeskov): check MethodsMap when parsePayments == true

func TestTransferWithRideTypes(t *testing.T) {
	// from https://etherscan.io/tx/0x363f979b58c82614db71229c2a57ed760e7bc454ee29c2f8fd1df99028667ea5

	expectedSignature := "transfer(address,uint256)"
	expectedName := "transfer"
	expectedFirstArg := "0x9a1989946ae4249AAC19ac7a038d24Aab03c3D8c"
	expectedSecondArg := "209470300000000000000000"

	hexdata := "0xa9059cbb0000000000000000000000009a1989946ae4249aac19ac7a038d24aab03c3d8c000000000000000000000000000000000000000000002c5b68601cc92ad60000"
	data, err := hex.DecodeString(strings.TrimPrefix(hexdata, "0x"))
	require.NoError(t, err)

	erc20Db := NewErc20MethodsMap()
	callData, err := erc20Db.ParseCallDataRide(data)
	require.NoError(t, err)

	require.Equal(t, expectedSignature, callData.Signature.String())
	require.Equal(t, expectedName, callData.Name)
	require.Equal(t, expectedFirstArg, proto.BytesToEthereumAddress(callData.Inputs[0].Value.(ride.RideBytes)).String())
	require.Equal(t, expectedSecondArg, callData.Inputs[1].Value.(ride.RideBigInt).String())
}

func TestRandomFunctionABIParsing(t *testing.T) {
	// taken and modified from https://etherscan.io/tx/0x2667bb17f2076cad4966849255898fbcaca68f2eb0d9ba585b310c79c098e970

	const (
		testSignature = Signature("minta(address,uint256,uint256,uint256,uint256)")
		hexData       = "0xe00c88d6000000000000000000000000892555e75350e11f2058d086c72b9c94c9493d7200000000000000000000000000000000000000000000000000000000000000a50000000000000000000000000000000000000000000000056bc75e2d631000000000000000000000000000000000000000000000000000056bc75e2d63100000000000000000000000000000000000000000000000000000000000000000000a"
	)

	var customDB = map[Selector]Method{
		testSignature.Selector(): {
			RawName: "minta",
			Inputs: Arguments{
				{Name: "_token", Type: Type{T: AddressTy}},
				{Name: "_id", Type: Type{T: UintTy, Size: 256}},
				{Name: "_supply", Type: Type{T: UintTy, Size: 256}},
				{Name: "_listPrice", Type: Type{T: UintTy, Size: 256}},
				{Name: "_fee", Type: Type{T: UintTy, Size: 256}},
			},
			Payments: nil,
			Sig:      testSignature,
		},
	}

	data, err := hex.DecodeString(strings.TrimPrefix(hexData, "0x"))
	require.NoError(t, err)
	db := MethodsMap{
		methods:       customDB,
		parsePayments: false,
	}
	callData, err := db.ParseCallDataRide(data)
	require.NoError(t, err)

	require.Equal(t, "minta", callData.Name)
	require.Equal(t,
		"0x892555E75350E11f2058d086C72b9C94C9493d72",
		proto.BytesToEthereumAddress(callData.Inputs[0].Value.(ride.RideBytes)).String(),
	)
	require.Equal(t, "165", callData.Inputs[1].Value.(ride.RideBigInt).String())
	require.Equal(t, "100000000000000000000", callData.Inputs[2].Value.(ride.RideBigInt).String())
	require.Equal(t, "100000000000000000000", callData.Inputs[3].Value.(ride.RideBigInt).String())
	require.Equal(t, "10", callData.Inputs[4].Value.(ride.RideBigInt).String())
}

func TestJsonAbi(t *testing.T) {
	expectedJson := `
[
  {
    "name": "transfer",
    "type": "function",
    "inputs": [
      {
        "name": "_to",
        "type": "bytes"
      },
      {
        "name": "_value",
        "type": "uint256"
      }
    ]
  }
]
`
	var expectedABI []abi
	err := json.Unmarshal([]byte(expectedJson), &expectedABI)
	require.NoError(t, err)

	erc20Meth := make([]Method, 0, len(erc20Methods))
	for _, method := range erc20Methods {
		erc20Meth = append(erc20Meth, method)
	}

	resJsonABI, err := getJsonAbi(erc20Meth)
	require.NoError(t, err)
	fmt.Println(string(resJsonABI))
	var abiRes []abi
	err = json.Unmarshal(resJsonABI, &abiRes)
	require.NoError(t, err)

	sort.Slice(abiRes, func(i, j int) bool { return abiRes[i].Name < abiRes[j].Name })
	sort.Slice(expectedABI, func(i, j int) bool { return expectedABI[i].Name < expectedABI[j].Name })

	require.Equal(t, expectedABI, abiRes)
}

var TestMethodWithAllTypes = []Method{
	{
		RawName: "testFunction",
		Inputs: Arguments{
			{Name: "stringVar", Type: Type{T: StringTy}},
			{Name: "intVar", Type: Type{T: IntTy, Size: 64}},
			{Name: "bytesVar", Type: Type{T: BytesTy}},
			{Name: "boolVar", Type: Type{T: BoolTy}},
			{
				Name: "sliceVar",
				Type: Type{
					T:    SliceTy,
					Elem: &Type{T: IntTy, Size: 64}},
			},
			{
				Name: "tupleSliceVar",
				Type: Type{
					T: TupleTy,
					TupleFields: Arguments{
						{Name: "union_index", Type: Type{T: UintTy, Size: 8}},
						{Name: "stringVar", Type: Type{T: StringTy}},
						{Name: "boolVar", Type: Type{T: BoolTy}},
						{Name: "intVar", Type: Type{T: IntTy, Size: 64}},
						{Name: "bytesVar", Type: Type{T: BytesTy}},
					},
				},
			},
		},
		Payments: &Argument{
			Name: "payments",
			Type: Type{
				T: SliceTy,
				Elem: &Type{
					T: TupleTy,
					TupleFields: Arguments{
						{Name: "number", Type: Type{T: IntTy, Size: 64}},
						{Name: "addr", Type: Type{T: AddressTy}},
					},
				},
			},
		},
	},
}

func TestJsonAbiWithAllTypes(t *testing.T) {
	expectedJson := `
[
  {
    "name": "testFunction",
    "type": "function",
    "inputs": [
      {
        "name": "stringVar",
        "type": "string"
      },
      {
        "name": "intVar",
        "type": "int64"
      },
      {
        "name": "bytesVar",
        "type": "bytes"
      },
      {
        "name": "boolVar",
        "type": "bool"
      },
      {
        "name": "sliceVar",
        "type": "int64[]"
      },
      {
        "name": "tupleSliceVar",
        "type": "tuple",
        "components": [
          {
            "name": "union_index",
            "type": "uint8"
          },
          {
            "name": "stringVar",
            "type": "string"
          },
          {
            "name": "boolVar",
            "type": "bool"
          },
          {
            "name": "intVar",
            "type": "int64"
          },
          {
            "name": "bytesVar",
            "type": "bytes"
          }
        ]
      },
      {
        "name": "payments",
        "type": "tuple[]",
        "components": [
          {
            "name": "number",
            "type": "int64"
          },
          {
            "name": "addr",
            "type": "bytes"
          }
        ]
      }
    ]
  }
]
`
	var expectedABI []abi
	err := json.Unmarshal([]byte(expectedJson), &expectedABI)
	require.NoError(t, err)

	resJsonABI, err := getJsonAbi(TestMethodWithAllTypes)
	require.NoError(t, err)
	fmt.Println(string(resJsonABI))
	var abiRes []abi
	err = json.Unmarshal(resJsonABI, &abiRes)
	require.NoError(t, err)

	sort.Slice(abiRes, func(i, j int) bool { return abiRes[i].Name < abiRes[j].Name })
	sort.Slice(expectedABI, func(i, j int) bool { return expectedABI[i].Name < expectedABI[j].Name })

	require.Equal(t, expectedABI, abiRes)
}

func TestParsingABIUsingRideMeta(t *testing.T) {
	// hexdata created with https://github.com/rust-ethereum/ethabi

	testdata := []struct {
		rideFunctionMeta     meta.Function
		hexdata              string
		expectedResultValues []ride.RideType
	}{
		{
			rideFunctionMeta: meta.Function{
				Name:      "some_test_fn",
				Arguments: []meta.Type{meta.Boolean, meta.String, meta.String},
			},
			hexdata: "0x7afebf3b0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000000861736661736466730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000015657468657265756d2061626920746573742e2e2e2e0000000000000000000000",
			expectedResultValues: []ride.RideType{
				ride.RideBoolean(true), ride.RideString("asfasdfs"), ride.RideString("ethereum abi test...."),
			},
		},
	}
	for _, test := range testdata {
		data, err := hex.DecodeString(strings.TrimPrefix(test.hexdata, "0x"))
		require.NoError(t, err)

		dAppMeta := meta.DApp{
			Version:       1,
			Functions:     []meta.Function{test.rideFunctionMeta},
			Abbreviations: meta.Abbreviations{},
		}
		db, err := newMethodsMapFromRideDAppMeta(dAppMeta, false)
		require.NoError(t, err)

		decodedCallData, err := db.ParseCallDataRide(data)
		require.NoError(t, err)

		values := make([]ride.RideType, 0, len(decodedCallData.Inputs))
		for _, arg := range decodedCallData.Inputs {
			values = append(values, arg.Value)
		}
		require.Equal(t, test.expectedResultValues, values)
	}
}
