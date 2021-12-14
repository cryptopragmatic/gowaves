package proto

import (
	"fmt"
	"io"
	"math/big"

	"github.com/pkg/errors"
	"github.com/umbracle/fastrlp"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"go.uber.org/atomic"
)

// EthereumGasPrice is a constant GasPrice which equals 10GWei according to the specification
const EthereumGasPrice = 10 * ethereumGWei
const DiffEthWaves = waveletToWeiMultiplier // in ethereum numbers are represented in 10^18. In waves it's 10^8

// EthereumTxType is an ethereum transaction type.
type EthereumTxType byte

const (
	EthereumLegacyTxType EthereumTxType = iota
	EthereumAccessListTxType
	EthereumDynamicFeeTxType
)

const (
	EthereumTransferWavesKind = iota + 1
	EthereumTransferAssetsKind
	EthereumInvokeKind
)

const (
	EthereumTransferMinFee      uint64 = 100_000
	EthereumScriptedAssetMinFee uint64 = 400_000
	EthereumInvokeMinFee        uint64 = 500_000
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
	String() string
	DecodedData() *ethabi.DecodedCallData
	Sender() WavesAddress
}

type EthereumTransferWavesTxKind struct {
	From WavesAddress
}

func NewEthereumTransferWavesTxKind() *EthereumTransferWavesTxKind {
	return &EthereumTransferWavesTxKind{}
}

func (tx *EthereumTransferWavesTxKind) DecodedData() *ethabi.DecodedCallData {
	return nil
}

func (tx *EthereumTransferWavesTxKind) String() string {
	return "EthereumTransferWavesTxKind"
}

func (tx *EthereumTransferWavesTxKind) Sender() WavesAddress {
	return tx.From
}

type EthereumTransferAssetsErc20TxKind struct {
	decodedData ethabi.DecodedCallData
	Arguments   ethabi.ERC20TransferArguments
	Asset       *OptionalAsset
	From        WavesAddress
}

func NewEthereumTransferAssetsErc20TxKind(decodedData ethabi.DecodedCallData, asset *OptionalAsset, arguments ethabi.ERC20TransferArguments) *EthereumTransferAssetsErc20TxKind {
	return &EthereumTransferAssetsErc20TxKind{Asset: asset, decodedData: decodedData, Arguments: arguments}
}

func (tx *EthereumTransferAssetsErc20TxKind) DecodedData() *ethabi.DecodedCallData {
	return &tx.decodedData
}

func (tx *EthereumTransferAssetsErc20TxKind) String() string {
	return "EthereumTransferAssetsErc20TxKind"
}

func (tx *EthereumTransferAssetsErc20TxKind) Sender() WavesAddress {
	return tx.From
}

type EthereumInvokeScriptTxKind struct {
	DecodedCallData *ethabi.DecodedCallData
	From            WavesAddress
}

func NewEthereumInvokeScriptTxKind(decodedData *ethabi.DecodedCallData) *EthereumInvokeScriptTxKind {
	return &EthereumInvokeScriptTxKind{DecodedCallData: decodedData}
}

func (tx *EthereumInvokeScriptTxKind) DecodedData() *ethabi.DecodedCallData {
	return tx.DecodedCallData
}

func (tx *EthereumInvokeScriptTxKind) String() string {
	return "EthereumInvokeScriptTxKind"
}

func (tx *EthereumInvokeScriptTxKind) Sender() WavesAddress {
	return tx.From
}

type EthereumTransaction struct {
	inner           EthereumTxData
	value           int64
	chainID         int64
	gasPrice        uint64
	innerBinarySize int
	senderPK        atomic.Value // *EthereumPublicKey
	TxKind          EthereumTransactionKind
	ID              *crypto.Digest
}

// NewEthereumTransaction is a utility function which should be used ONLY in tests
func NewEthereumTransaction(inner EthereumTxData, txKind EthereumTransactionKind, id *crypto.Digest, senderPK *EthereumPublicKey, innerBinarySize int) EthereumTransaction {
	tx := EthereumTransaction{
		inner:           inner,
		innerBinarySize: innerBinarySize,
		TxKind:          txKind,
		ID:              id,
	}
	res := new(big.Int).Div(tx.inner.value(), big.NewInt(int64(DiffEthWaves)))
	tx.value = res.Int64()

	tx.threadSafeSetSenderPK(senderPK)
	return tx
}

func (tx *EthereumTransaction) GetTypeInfo() TransactionTypeInfo {
	return TransactionTypeInfo{
		Type:         EthereumMetamaskTransaction,
		ProofVersion: Proof,
	}
}

func (tx *EthereumTransaction) GetVersion() byte {
	// EthereumTransaction version always should be zero
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

func (tx *EthereumTransaction) GetSender(_ Scheme) (Address, error) {
	return tx.From()
}

func (tx *EthereumTransaction) GetFee() uint64 {
	// in scala node this is "gasLimit" field.
	return tx.Gas()
}

func (tx *EthereumTransaction) GetTimestamp() uint64 {
	return tx.Nonce()
}

func (tx *EthereumTransaction) threadSafeGetSenderPK() *EthereumPublicKey {
	senderPK := tx.senderPK.Load()
	if senderPK != nil {
		return senderPK.(*EthereumPublicKey)
	}
	return nil
}

func (tx *EthereumTransaction) threadSafeSetSenderPK(senderPK *EthereumPublicKey) {
	tx.senderPK.Store(senderPK)
}

// Verify performs ONLY transaction signature verification and calculates EthereumPublicKey of transaction
// For basic transaction checks use Validate method
func (tx *EthereumTransaction) Verify() (*EthereumPublicKey, error) {
	if senderPK := tx.threadSafeGetSenderPK(); senderPK != nil {
		return senderPK, nil
	}
	signer := MakeEthereumSigner(tx.inner.chainID())
	senderPK, err := signer.SenderPK(tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to verify EthereumTransaction")
	}
	tx.threadSafeSetSenderPK(senderPK)
	return senderPK, nil
}

// Validate performs basic checks for EthereumTransaction according to the specification
// This method doesn't include signature verification. Use Verify method for signature verification
func (tx *EthereumTransaction) Validate(scheme Scheme) (Transaction, error) {
	// same chainID
	if tx.ChainId() != int64(scheme) {
		// TODO: introduce new error type for scheme validation
		txChainID := tx.ChainId()
		return nil, errs.NewTxValidationError(fmt.Sprintf(
			"Address belongs to another network: expected: %d(%c), actual: %d(%c)",
			scheme, scheme,
			txChainID, txChainID,
		))
	}
	// accept only EthereumLegacyTxType (this check doesn't exist in scala)
	if tx.EthereumTxType() != EthereumLegacyTxType {
		return nil, errs.NewTxValidationError("the ethereum transaction's type is not legacy tx")
	}
	// max size of EthereumTransaction is 1Mb (this check doesn't exist in scala)
	if tx.innerBinarySize > 1024*1024 {
		return nil, errs.NewTxValidationError("too big size of transaction")
	}
	// insufficient fee
	if tx.Gas() <= 0 {
		return nil, errs.NewFeeValidation("insufficient fee")
	}
	// too many waves (this check doesn't exist in scala)
	// TODO I'm not sure this is should be checked for all eth tx kinds. Only for transfer waves kind
	wavelets, err := EthereumWeiToWavelet(tx.Value())
	if err != nil {
		return nil, errs.NewFeeValidation(err.Error())
	}
	// non positive amount
	if wavelets < 0 {
		return nil, errs.NewNonPositiveAmount(wavelets, "waves")
	}
	// a cancel transaction: value == 0 && data == 0x
	if tx.Value() == 0 && len(tx.Data()) == 0 {
		return nil, errs.NewTxValidationError("Transaction cancellation is not supported")
	}
	// either data or value field is set
	if tx.Value() != 0 && len(tx.Data()) != 0 {
		return nil, errs.NewTxValidationError("Transaction should have either data or value")
	}
	// gasPrice == 10GWei
	if tx.GasPrice() != EthereumGasPrice {
		return nil, errs.NewTxValidationError("Gas price must be 10 Gwei")
	}
	// deny a contract creation transaction (this check doesn't exist in scala)
	if tx.To() == nil {
		return nil, errs.NewTxValidationError("Contract creation transaction is not supported")
	}
	// positive timestamp (this check doesn't exist in scala)
	if tx.Nonce() <= 0 {
		return nil, errs.NewTxValidationError("invalid timestamp")
	}
	return tx, nil
}

func (tx *EthereumTransaction) GenerateID(_ Scheme) error {
	if tx.ID != nil {
		return nil
	}
	body, err := tx.EncodeCanonical()
	if err != nil {
		return err
	}

	id := Keccak256EthereumHash(body)
	tx.ID = (*crypto.Digest)(&id)
	return nil
}

func (tx *EthereumTransaction) MerkleBytes(_ Scheme) ([]byte, error) {
	return tx.EncodeCanonical()
}

func (tx *EthereumTransaction) Sign(_ Scheme, _ crypto.SecretKey) error {
	return errors.New("Sign method for EthereumTransaction isn't implemented")
}

func (tx *EthereumTransaction) MarshalBinary() ([]byte, error) {
	return nil, errors.New("EthereumTransaction does not support 'MarshalBinary' method.")
}

func (tx *EthereumTransaction) UnmarshalBinary(_ []byte, _ Scheme) error {
	return errors.New("EthereumTransaction does not support 'UnmarshalBinary' method.")
}

func (tx *EthereumTransaction) BodyMarshalBinary() ([]byte, error) {
	return nil, errors.New("EthereumTransaction does not support 'BodyMarshalBinary' method.")
}

func (tx *EthereumTransaction) BinarySize() int {
	return 0
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
	canonical, err := tx.EncodeCanonical()
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
	return tx.inner.ethereumTxType()
}

// ChainId returns the EIP155 chain ID of the transaction. The return value will always be
// non-nil. For legacy transactions which are not replay-protected, the return value is
// zero.
func (tx *EthereumTransaction) ChainId() int64 {
	return tx.chainID
}

// Data returns the input data of the transaction.
func (tx *EthereumTransaction) Data() []byte { return tx.inner.data() }

// AccessList returns the access list of the transaction.
func (tx *EthereumTransaction) AccessList() EthereumAccessList { return tx.inner.accessList() }

// Gas returns the gas limit of the transaction.
func (tx *EthereumTransaction) Gas() uint64 { return tx.inner.gas() }

// GasPrice returns the gas price of the transaction.
func (tx *EthereumTransaction) GasPrice() uint64 { return tx.gasPrice }

// Value returns the ether amount of the transaction.
func (tx *EthereumTransaction) Value() int64 { return tx.value }

// Nonce returns the sender account nonce of the transaction.
func (tx *EthereumTransaction) Nonce() uint64 { return tx.inner.nonce() }

// To returns the recipient address of the transaction.
// For contract-creation transactions, To returns nil.
func (tx *EthereumTransaction) To() *EthereumAddress { return tx.inner.to().copy() }

func (tx *EthereumTransaction) WavesAddressTo(scheme byte) (*WavesAddress, error) {
	toEthAdr := tx.inner.to()
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
	senderPK, err := tx.Verify()
	if err != nil {
		return EthereumAddress{}, err
	}
	return senderPK.EthereumAddress(), nil
}

// FromPK returns the sender public key of the transaction.
// Returns error if transaction doesn't pass validation.
func (tx *EthereumTransaction) FromPK() (*EthereumPublicKey, error) {
	senderPK, err := tx.Verify()
	if err != nil {
		return nil, err
	}
	return senderPK.copy(), nil
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
	return tx.inner.rawSignatureValues()
}

func validateEthereumTx(tx *EthereumTransaction) error {
	switch tx.TxKind.(type) {
	case *EthereumTransferWavesTxKind:
		res := new(big.Int).Div(tx.inner.value(), big.NewInt(int64(DiffEthWaves)))
		if ok := res.IsInt64(); !ok {
			return errors.Errorf("failed to convert amount from ethreum transaction (big int) to int64. value is %d", tx.Value())
		}
		tx.value = res.Int64()
		return nil
	case *EthereumTransferAssetsErc20TxKind:
		return nil
	case *EthereumInvokeScriptTxKind:
		return nil
	default:
		return errors.New("failed to check ethereum transaction, wrong kind of tx")
	}
}

func GuessEthereumTransactionKind(data []byte) (int64, error) {
	if len(data) == 0 {
		return EthereumTransferWavesKind, nil
	}

	selectorBytes := data
	if len(data) < ethabi.SelectorSize {
		return 0, errors.Errorf("length of data from ethereum transaction is less than %d", ethabi.SelectorSize)
	}
	selector, err := ethabi.NewSelectorFromBytes(selectorBytes[:ethabi.SelectorSize])
	if err != nil {
		return 0, errors.Wrap(err, "failed to guess ethereum transaction kind")
	}

	if ethabi.IsERC20TransferSelector(selector) {
		return EthereumTransferAssetsKind, nil
	}

	return EthereumInvokeKind, nil
}
func GetEthereumTransactionKind(ethTx EthereumTransaction) (EthereumTransactionKind, error) {
	txKind, err := GuessEthereumTransactionKind(ethTx.Data())
	if err != nil {
		return nil, errors.Wrap(err, "failed to guess ethereum tx kind")
	}

	switch txKind {
	case EthereumTransferWavesKind:
		return NewEthereumTransferWavesTxKind(), nil
	case EthereumTransferAssetsKind:
		db := ethabi.NewErc20MethodsMap()
		decodedData, err := db.ParseCallData(ethTx.Data())
		if err != nil {
			return nil, errors.Errorf("failed to parse ethereum data")
		}
		if len(decodedData.Inputs) != ethabi.NumberOfERC20TransferArguments {
			return nil, errors.Errorf("the number of arguments of erc20 function is %d, but expected it to be %d", len(decodedData.Inputs), ethabi.NumberOfERC20TransferArguments)
		}

		erc20Arguments, err := ethabi.GetERC20TransferArguments(decodedData)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get erc20 arguments from decoded data")
		}
		return NewEthereumTransferAssetsErc20TxKind(*decodedData, nil, erc20Arguments), nil
	case EthereumInvokeKind:
		return NewEthereumInvokeScriptTxKind(nil), nil
	default:
		return nil, errors.New("unexpected ethereum tx kind")
	}
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
		tx.inner = &inner
	} else {
		// It's an EIP2718 typed transaction envelope.
		inner, err := tx.decodeTypedCanonical(canonicalData)
		if err != nil {
			return errors.Wrap(err,
				"failed to unmarshal from canonical representation ethereum typed transaction",
			)
		}
		tx.inner = inner
	}
	tx.innerBinarySize = len(canonicalData)

	if ok := tx.inner.chainID().IsInt64(); !ok {
		return errors.Errorf("failed to recognize chainID of ethereum transaction: over int64")
	}
	tx.chainID = tx.inner.chainID().Int64()
	if ok := tx.inner.gasPrice().IsUint64(); !ok {
		return errors.Errorf("failed to recognize GasPrice of ethereum transaction: over uint64")
	}
	tx.gasPrice = tx.inner.gasPrice().Uint64()
	var err error
	tx.TxKind, err = GetEthereumTransactionKind(*tx)
	if err != nil {
		return errors.Errorf("failed to guess ethereum transaction kind, %v", err)
	}
	err = validateEthereumTx(tx)
	if err != nil {
		return errors.Errorf("validation of ethereum transaction after initialization failed , %v", err)
	}
	return nil
}

// EncodeCanonical returns the canonical binary encoding of a transaction.
// For legacy transactions, it returns the RLP encoding. For EIP-2718 typed
// transactions, it returns the type and payload.
func (tx *EthereumTransaction) EncodeCanonical() ([]byte, error) {
	var (
		canonical []byte
		arena     fastrlp.Arena
	)
	if tx.EthereumTxType() == EthereumLegacyTxType {
		fastrlpTx := tx.inner.marshalToFastRLP(&arena)
		canonical = fastrlpTx.MarshalTo(nil)
	} else {
		canonical = tx.encodeTypedCanonical(&arena)
	}
	return canonical, nil
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
	typedTxVal := tx.inner.marshalToFastRLP(arena)
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
	switch tx := tx.inner.(type) {
	case *EthereumLegacyTx:
		return tx.V != nil && isProtectedV(tx.V)
	default:
		return true
	}
}
