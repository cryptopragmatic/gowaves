package proto

import (
	"bytes"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"io"
	"math/big"

	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

// EthereumGasPrice is a constant GasPrice which equals 10GWei according to the specification
const EthereumGasPrice = 10 * EthereumGWei
const DiffEthWaves = 1e10 // in ethereum numbers are represented in 10^18. In waves it's 10^8
// EthereumTxType is an ethereum transaction type.
type EthereumTxType byte

const (
	EthereumLegacyTxType EthereumTxType = iota
	EthereumAccessListTxType
	EthereumDynamicFeeTxType
)

const (
	EthereumTransferMinFee uint64 = 100000
	EthereumInvokeMinFee   uint64 = 500000
)

func (e EthereumTxType) String() string {
	switch e {
	case EthereumLegacyTxType:
		return "EthereumLegacyTxType"
	case EthereumAccessListTxType:
		return "EthereumAccessListTxType"
	case EthereumDynamicFeeTxType:
		return "EthereumDynamicFeeTxType"
	default:
		return ""
	}
}

var (
	ErrInvalidSig         = errors.New("invalid transaction v, r, s values")
	ErrTxTypeNotSupported = errors.New("transaction type not supported")
)

type fastRLPSignerHasher interface {
	signerHashFastRLP(chainID *big.Int, arena *fastrlp.Arena) *fastrlp.Value
}

type RLPDecoder interface {
	DecodeRLP([]byte) error
}

type RLPEncoder interface {
	EncodeRLP(io.Writer) error
}

type fastRLPMarshaler interface {
	marshalToFastRLP(arena *fastrlp.Arena) *fastrlp.Value
}

type EthereumTxData interface {
	ethereumTxType() EthereumTxType
	copy() EthereumTxData // creates a deep copy and initializes all fields

	chainID() *big.Int
	accessList() EthereumAccessList
	data() []byte
	gas() uint64
	gasPrice() *big.Int
	gasTipCap() *big.Int
	gasFeeCap() *big.Int
	value() *big.Int
	nonce() uint64
	to() *EthereumAddress

	rawSignatureValues() (v, r, s *big.Int)
	setSignatureValues(chainID, v, r, s *big.Int)

	fastRLPMarshaler
	fastRLPSignerHasher
}

type EthereumTransactionKind interface {
	DecodedData() *ethabi.DecodedCallData
}

type EthereumTransferWavesTxKind struct {
}

func NewEthereumTransferWavesTxKind() *EthereumTransferWavesTxKind {
	return &EthereumTransferWavesTxKind{}
}

func (tx *EthereumTransferWavesTxKind) DecodedData() *ethabi.DecodedCallData {
	return nil
}

type EthereumTransferAssetsErc20TxKind struct {
	decodedData ethabi.DecodedCallData
	Asset       OptionalAsset
}

func NewEthereumTransferAssetsErc20TxKind(decodedData ethabi.DecodedCallData, asset OptionalAsset) *EthereumTransferAssetsErc20TxKind {
	return &EthereumTransferAssetsErc20TxKind{Asset: asset, decodedData: decodedData}
}

func (tx *EthereumTransferAssetsErc20TxKind) DecodedData() *ethabi.DecodedCallData {
	return &tx.decodedData
}

type EthereumInvokeScriptTxKind struct {
	decodedData ethabi.DecodedCallData
}

func NewEthereumInvokeScriptTxKind(decodedData ethabi.DecodedCallData) *EthereumInvokeScriptTxKind {
	return &EthereumInvokeScriptTxKind{decodedData: decodedData}
}

func (tx *EthereumInvokeScriptTxKind) DecodedData() *ethabi.DecodedCallData {
	return &tx.decodedData
}

type EthereumTransaction struct {
	Inner           EthereumTxData
	innerBinarySize int
	TxKind          EthereumTransactionKind
	ID              *crypto.Digest
	SenderPK        *EthereumPublicKey
}

func (tx *EthereumTransaction) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{
		Type:         EthereumMetamaskTransaction,
		ProofVersion: Proof,
	}
}

func (tx *EthereumTransaction) GetVersion() byte {
	return 0
}

func (tx *EthereumTransaction) GetID(scheme Scheme) ([]byte, error) {
	if tx.ID == nil {
		if err := tx.GenerateID(scheme); err != nil {
			return nil, err
		}
	}
	return tx.ID.Bytes(), nil
}

func (tx *EthereumTransaction) GetSender(scheme Scheme) (Address, error) {
	return tx.From()
}

func (tx *EthereumTransaction) GetFee() uint64 {
	// in scala node this is "gasLimit" field.
	return tx.Gas()
}

func (tx *EthereumTransaction) GetTimestamp() uint64 {
	return tx.Nonce()
}

func (tx *EthereumTransaction) Validate() (Transaction, error) {
	if tx.SenderPK != nil {
		return tx, nil
	}
	signer := MakeEthereumSigner(tx.ChainId())
	senderPK, err := signer.SenderPK(tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate EthereumTransaction")
	}
	tx.SenderPK = senderPK
	return tx, nil
}

func (tx *EthereumTransaction) GenerateID(scheme Scheme) error {
	if tx.ID != nil {
		return nil
	}
	body, err := MarshalTxBody(scheme, tx)
	if err != nil {
		return err
	}

	id := Keccak256EthereumHash(body)
	tx.ID = (*crypto.Digest)(&id)
	return nil
}

func (tx *EthereumTransaction) Sign(_ Scheme, _ crypto.SecretKey) error {
	return errors.New("Sign method for EthereumTransaction isn't implemented")
}

func (tx *EthereumTransaction) MarshalBinary() ([]byte, error) {
	rlpData, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal binary ethereum transaction")
	}
	data := make([]byte, 1+len(rlpData))
	data[0] = byte(tx.GetTypeInfo().Type)
	copy(data[1:], rlpData)
	return data, nil
}

func (tx *EthereumTransaction) UnmarshalBinary(bytes []byte, scheme Scheme) error {
	if l := len(bytes); l < 1 {
		return errors.New("failed to UnmarshalBinary ethereum transaction, received empty data")
	}
	if bytes[0] != byte(tx.GetTypeInfo().Type) {
		return errors.Errorf("incorrect transaction type %d for EthereumTransaction transaction", bytes[0])
	}

	ethereumTxCanonicalBytes := bytes[1:]
	if err := tx.DecodeCanonical(ethereumTxCanonicalBytes); err != nil {
		return errors.Wrap(err, "failed to UnmarshalBinary ethereum transaction from canonical representation")
	}
	if err := tx.GenerateID(scheme); err != nil {
		return errors.Wrap(err, "failed to generate ID for ethereum transaction")
	}
	return nil
}

func (tx *EthereumTransaction) BodyMarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	b.Grow(tx.innerBinarySize)
	if err := tx.EncodeCanonical(&b); err != nil {
		return nil, errors.Wrapf(err,
			"failed to marshal ethereum transaction to canonical representation, ehtTxType %q",
			tx.EthereumTxType().String(),
		)
	}
	return b.Bytes(), nil
}

func (tx *EthereumTransaction) bodyUnmarshalBinary(canonical []byte) error {
	if err := tx.DecodeCanonical(canonical); err != nil {
		return errors.Wrapf(err, "failed to unmarshal ethereum transaction from canonical representation")
	}
	return nil
}

func (tx *EthereumTransaction) BinarySize() int {
	return tx.GetTypeInfo().Type.BinarySize() + tx.innerBinarySize
}

func (tx *EthereumTransaction) MarshalToProtobuf(_ Scheme) ([]byte, error) {
	return nil, errors.New("EthereumTransaction does not support 'MarshalToProtobuf' method.")
}

func (tx *EthereumTransaction) UnmarshalFromProtobuf(_ []byte) error {
	return errors.New("EthereumTransaction does not support 'UnmarshalFromProtobuf' method.")
}

func (tx *EthereumTransaction) MarshalSignedToProtobuf(scheme Scheme) ([]byte, error) {
	return MarshalSignedTxDeterministic(tx, scheme)
}

func (tx *EthereumTransaction) UnmarshalSignedFromProtobuf(bytes []byte) error {
	t, err := SignedTxFromProtobuf(bytes)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal from protobuf ethereum transaction")
	}
	ethTx, ok := t.(*EthereumTransaction)
	if !ok {
		return errors.Errorf(
			"failed to cast unmarshalled result '%T' to '*EthereumTransaction' type",
			t,
		)
	}
	*tx = *ethTx
	return nil
}

func (tx *EthereumTransaction) ToProtobuf(_ Scheme) (*g.Transaction, error) {
	return nil, errors.New("EthereumTransaction does not support 'ToProtobuf' method.")
}

func (tx *EthereumTransaction) ToProtobufSigned(_ Scheme) (*g.SignedTransaction, error) {
	canonical, err := tx.BodyMarshalBinary()
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to marshal binary EthereumTransaction, type %q",
			tx.EthereumTxType().String(),
		)
	}
	signed := g.SignedTransaction{
		Transaction: &g.SignedTransaction_EthereumTransaction{
			EthereumTransaction: canonical,
		},
	}
	return &signed, nil
}

// EthereumTxType returns the transaction type.
func (tx *EthereumTransaction) EthereumTxType() EthereumTxType {
	return tx.Inner.ethereumTxType()
}

// ChainId returns the EIP155 chain ID of the transaction. The return value will always be
// non-nil. For legacy transactions which are not replay-protected, the return value is
// zero.
func (tx *EthereumTransaction) ChainId() *big.Int {
	return tx.Inner.chainID()
}

// Data returns the input data of the transaction.
func (tx *EthereumTransaction) Data() []byte { return tx.Inner.data() }

// AccessList returns the access list of the transaction.
func (tx *EthereumTransaction) AccessList() EthereumAccessList { return tx.Inner.accessList() }

// Gas returns the gas limit of the transaction.
func (tx *EthereumTransaction) Gas() uint64 { return tx.Inner.gas() }

// GasPrice returns the gas price of the transaction.
func (tx *EthereumTransaction) GasPrice() *big.Int { return copyBigInt(tx.Inner.gasPrice()) }

// GasTipCap returns the gasTipCap per gas of the transaction.
func (tx *EthereumTransaction) GasTipCap() *big.Int { return copyBigInt(tx.Inner.gasTipCap()) }

// GasFeeCap returns the fee cap per gas of the transaction.
func (tx *EthereumTransaction) GasFeeCap() *big.Int { return copyBigInt(tx.Inner.gasFeeCap()) }

// Value returns the ether amount of the transaction.
func (tx *EthereumTransaction) Value() *big.Int { return copyBigInt(tx.Inner.value()) }

// Nonce returns the sender account nonce of the transaction.
func (tx *EthereumTransaction) Nonce() uint64 { return tx.Inner.nonce() }

// To returns the recipient address of the transaction.
// For contract-creation transactions, To returns nil.
func (tx *EthereumTransaction) To() *EthereumAddress { return tx.Inner.to().copy() }

func (tx *EthereumTransaction) WavesAddressTo(scheme byte) (*WavesAddress, error) {
	toEthAdr := tx.Inner.to()
	if toEthAdr == nil { // contract-creation transactions, To returns nil
		return nil, errors.New("recipient address is nil, but it has been called")
	}

	to, err := toEthAdr.ToWavesAddress(scheme)
	if err != nil {
		return nil, err
	}
	return &to, nil
}

// From returns the sender address of the transaction.
// Returns error if transaction doesn't pass validation.

func (tx *EthereumTransaction) From() (EthereumAddress, error) {
	if _, err := tx.Validate(); err != nil {
		return EthereumAddress{}, err
	}
	addr := tx.SenderPK.EthereumAddress()
	return addr, nil
}

// FromPK returns the sender public key of the transaction.
// Returns error if transaction doesn't pass validation.
func (tx *EthereumTransaction) FromPK() (*EthereumPublicKey, error) {
	if _, err := tx.Validate(); err != nil {
		return nil, err
	}
	return tx.SenderPK.copy(), nil
}

func (tx *EthereumTransaction) WavesAddressFrom(scheme byte) (WavesAddress, error) {
	ethSender, err := tx.From()
	if err != nil {
		return WavesAddress{}, err
	}
	sender, err := ethSender.ToWavesAddress(scheme)
	if err != nil {
		return WavesAddress{}, err
	}
	return sender, nil
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
func (tx *EthereumTransaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.Inner.rawSignatureValues()
}

// DecodeCanonical decodes the canonical binary encoding of transactions.
// It supports legacy RLP transactions and EIP2718 typed transactions.
func (tx *EthereumTransaction) DecodeCanonical(canonicalData []byte) error {
	// check according to the EIP2718
	if len(canonicalData) > 0 && canonicalData[0] > 0x7f {
		// It's a legacy transaction.
		parser := fastrlp.Parser{}
		value, err := parser.Parse(canonicalData)
		if err != nil {
			return errors.Wrap(err, "failed to parse canonical representation as RLP")
		}
		var inner EthereumLegacyTx
		if err := inner.unmarshalFromFastRLP(value); err != nil {
			return errors.Wrapf(err,
				"failed to unmarshal from RLP ethereum legacy transaction, ethereumTxType %q",
				EthereumLegacyTxType.String(),
			)
		}
		tx.Inner = &inner
	} else {
		// It's an EIP2718 typed transaction envelope.
		inner, err := tx.decodeTypedCanonical(canonicalData)
		if err != nil {
			return errors.Wrap(err,
				"failed to unmarshal from canonical representation ethereum typed transaction",
			)
		}
		tx.Inner = inner
	}
	tx.innerBinarySize = len(canonicalData)
	return nil
}

// EncodeCanonical writes the canonical binary encoding of a transaction to w.
// For legacy transactions, it returns the RLP encoding. For EIP-2718 typed
// transactions, it returns the type and payload.
func (tx *EthereumTransaction) EncodeCanonical(w io.Writer) error {
	var (
		canonical []byte
		arena     fastrlp.Arena
	)
	if tx.EthereumTxType() == EthereumLegacyTxType {
		fastrlpTx := tx.Inner.marshalToFastRLP(&arena)
		canonical = fastrlpTx.MarshalTo(nil)
	} else {
		canonical = tx.encodeTypedCanonical(&arena)
	}
	if _, err := w.Write(canonical); err != nil {
		return errors.Wrapf(err, "failed to write canonical encoded ethereum transaction to %T", w)
	}
	return nil
}

// decodeTypedCanonical decodes a typed transaction from the canonical format.
func (tx *EthereumTransaction) decodeTypedCanonical(canonicalData []byte) (EthereumTxData, error) {
	if len(canonicalData) == 0 {
		return nil, errors.New("empty typed transaction bytes")
	}
	switch txType, rlpData := canonicalData[0], canonicalData[1:]; EthereumTxType(txType) {
	case EthereumAccessListTxType:
		var inner EthereumAccessListTx
		if err := inner.DecodeRLP(rlpData); err != nil {
			return nil, errors.Wrapf(err,
				"failed to unmarshal ethereum tx from RLP, ethereumTxType %q",
				EthereumAccessListTxType.String(),
			)
		}
		return &inner, nil
	case EthereumDynamicFeeTxType:
		var inner EthereumDynamicFeeTx
		if err := inner.DecodeRLP(rlpData); err != nil {
			return nil, errors.Wrapf(err,
				"failed to unmarshal ethereum tx from RLP, ethereumTxType %q",
				EthereumDynamicFeeTxType.String(),
			)
		}
		return &inner, nil
	default:
		return nil, ErrTxTypeNotSupported
	}
}

// encodeTypedCanonical returns the canonical encoding of a typed transaction.
func (tx *EthereumTransaction) encodeTypedCanonical(arena *fastrlp.Arena) []byte {
	typedTxVal := tx.Inner.marshalToFastRLP(arena)
	canonicalMarshaled := make([]byte, 0, 1+typedTxVal.Len())
	canonicalMarshaled = append(canonicalMarshaled, byte(tx.EthereumTxType()))
	canonicalMarshaled = typedTxVal.MarshalTo(canonicalMarshaled)
	return canonicalMarshaled
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28 && v != 1 && v != 0
	}
	// anything not 27 or 28 is considered protected
	return true
}

// Protected says whether the transaction is replay-protected.
func (tx *EthereumTransaction) Protected() bool {
	switch tx := tx.Inner.(type) {
	case *EthereumLegacyTx:
		return tx.V != nil && isProtectedV(tx.V)
	default:
		return true
	}
}