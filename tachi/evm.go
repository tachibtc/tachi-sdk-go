package tachi

import (
	"context"
	"net/url"
)

// EVMService groups the read-only EVM endpoints: eth_call-style
// simulation, contract code, receipts, account and storage lookups.
// Addresses are 0x-prefixed 20-byte hex; hashes are 32-byte hex.
type EVMService service

// EVMCallRequest is the body of EVMService.Call. Hex fields take an
// optional 0x prefix. Leaving To empty simulates a deploy of Data, which is
// how a client sizes the gas limit of a TxDeploy.
type EVMCallRequest struct {
	// From is the caller's EVM address; optional.
	From string `json:"from,omitempty"`
	// To is the contract to call; empty simulates a deploy of Data.
	To string `json:"to,omitempty"`
	// Data is the calldata, or the init code when To is empty.
	Data string `json:"data"`
	// Gas caps the simulation; zero (or anything above it) means the
	// per-call limit of 1,000,000.
	Gas uint64 `json:"gas,omitempty"`
	// Nonce, when set, is the sender nonce to simulate at; it fixes a
	// simulated deploy's address. Without it a From simulates at the EVM
	// state nonce, which can lag the sender nonce, so pass the nonce you
	// will sign with: EVMService.Account's Nonce for your pubkey. (Not
	// Address.Nonce's NextNonce, which is one too high for EVM txs.)
	Nonce *uint64 `json:"nonce,omitempty"`
}

// EVMLog is one event emitted by an EVM execution.
type EVMLog struct {
	// LogIndex is the log's position in its tx. Set by Tx.Get and
	// EVM.Receipt; always zero from EVM.Call.
	LogIndex int `json:"log_index"`
	// Address is the emitting contract.
	Address string `json:"address"`
	// Topics are the indexed event topics, 0x hex.
	Topics []string `json:"topics"`
	// Data is the non-indexed event data, 0x hex.
	Data string `json:"data"`
}

// EVMCallResponse is returned by EVMService.Call.
type EVMCallResponse struct {
	// Status is 1 for success, 0 for a revert or exceptional halt.
	Status uint8 `json:"status"`
	// GasUsed is the gas the call would be charged, after refunds.
	GasUsed uint64 `json:"gas_used"`
	// ContractAddress is the address a simulated deploy would create.
	ContractAddress string `json:"contract_address,omitempty"`
	// ReturnData is the call's output, or the revert payload, 0x hex.
	ReturnData string `json:"return_data,omitempty"`
	// Logs are the events the call would emit; empty when Status is 0.
	Logs []EVMLog `json:"logs"`
	// Error describes why execution failed; empty on success.
	Error string `json:"error,omitempty"`
}

// EVMExecution is the outcome of an EVM tx executed in a committed block.
type EVMExecution struct {
	// Status is 1 for success, 0 for a revert or halt that still applied.
	Status uint8 `json:"status"`
	// Error is why execution failed; empty on success.
	Error string `json:"error,omitempty"`
	// GasUsed is the gas charged.
	GasUsed uint64 `json:"gas_used"`
	// FeeSats is GasUsed at GasPrice in sats, rounded up.
	FeeSats int64 `json:"fee_sats"`
	// RefundSats is what the inputs set aside beyond outputs and fee.
	RefundSats int64 `json:"refund_sats"`
	// RefundVTXOID is the refund VTXO (output index len(outputs)); empty
	// when there is no refund.
	RefundVTXOID string `json:"refund_vtxo_id,omitempty"`
	// ReturnData is the call output or revert payload, 0x hex.
	ReturnData string `json:"return_data,omitempty"`
	// Logs are the events emitted.
	Logs []EVMLog `json:"logs"`
}

// TxEVM is the EVM view of a TxDeploy/TxCall on GetTransactionResponse.
// The embedded EVMExecution is nil until the tx has executed in a
// committed block (Executed is true).
type TxEVM struct {
	// From is the sender's EVM address, derived from the tx pubkey.
	From string `json:"from"`
	// To is the called contract; TxCall only.
	To string `json:"to,omitempty"`
	// ContractAddress is the deployed contract; TxDeploy only. From the
	// receipt once executed, derived from sender and nonce while pending.
	ContractAddress string `json:"contract_address,omitempty"`
	// Nonce is the sender nonce the tx was signed with.
	Nonce uint64 `json:"nonce"`
	// GasLimit is the tx's gas limit.
	GasLimit uint64 `json:"gas_limit"`
	// GasPrice is in millisats per gas.
	GasPrice uint64 `json:"gas_price"`
	// MaxFeeSats is GasLimit at GasPrice in sats, rounded up.
	MaxFeeSats int64 `json:"max_fee_sats"`
	// Input is the init code (deploy) or calldata (call), 0x hex.
	Input string `json:"input"`
	// InputSize is the length of Input in bytes.
	InputSize int `json:"input_size"`
	// MethodID is the first 4 bytes of calldata; TxCall with input >= 4 bytes.
	MethodID string `json:"method_id,omitempty"`
	// Executed is true once the tx has a receipt.
	Executed bool `json:"executed"`
	*EVMExecution
}

// EVMReceiptResponse is returned by EVMService.Receipt: the stored receipt
// plus read-time context from the committed tx. If the daemon's CometBFT
// has not indexed the tx, only the stored fields (Status, GasUsed,
// ContractAddress, ReturnData, Logs, Error) are set.
type EVMReceiptResponse struct {
	// TxHash is the tx hash, bare hex.
	TxHash string `json:"tx_hash"`
	// Height is the committing block height.
	Height int64 `json:"height"`
	// TxIndex is the tx's position in its block.
	TxIndex uint32 `json:"tx_index"`
	// BlockTime is the committing block's time, unix seconds.
	BlockTime int64 `json:"block_time"`
	// From is the sender's EVM address.
	From string `json:"from"`
	// To is the called contract; TxCall only.
	To string `json:"to,omitempty"`
	// ContractAddress is the deployed contract; TxDeploy only.
	ContractAddress string `json:"contract_address,omitempty"`
	// GasLimit is the tx's gas limit.
	GasLimit uint64 `json:"gas_limit"`
	// GasPrice is in millisats per gas.
	GasPrice uint64 `json:"gas_price"`
	EVMExecution
}

// EVMCodeResponse is returned by EVMService.Code.
type EVMCodeResponse struct {
	// Address is the queried EVM address.
	Address string `json:"address"`
	// Code is the runtime bytecode, 0x hex; "0x" when there is none.
	Code string `json:"code"`
}

// EVMAccountResponse is returned by EVMService.Account.
type EVMAccountResponse struct {
	// Address is the EVM address.
	Address string `json:"address"`
	// Pubkey echoes the input when it was a pubkey.
	Pubkey string `json:"pubkey,omitempty"`
	// Nonce is the account's next nonce. For a pubkey input it is the
	// sender nonce the next tx must use.
	Nonce uint64 `json:"nonce"`
	// IsContract reports whether code is deployed at Address.
	IsContract bool `json:"is_contract"`
	// CodeSize is the runtime code length in bytes.
	CodeSize int `json:"code_size"`
	// CodeHash is keccak256 of the runtime code; contracts only.
	CodeHash string `json:"code_hash,omitempty"`
}

// EVMStorageResponse is returned by EVMService.Storage.
type EVMStorageResponse struct {
	// Address is the queried EVM address.
	Address string `json:"address"`
	// Slot is the left-padded 32-byte slot, 0x hex.
	Slot string `json:"slot"`
	// Value is the 32-byte slot value, 0x hex; zero if never written.
	Value string `json:"value"`
}

// Call runs calldata against committed state without committing anything,
// like eth_call. Leave req.To empty to simulate a deploy and read its gas.
// The daemon rate-limits this endpoint per IP (429 *ErrorResponse).
func (s *EVMService) Call(ctx context.Context, req *EVMCallRequest) (*EVMCallResponse, *Response, error) {
	var out EVMCallResponse
	resp, err := s.client.post(ctx, "tachi_evmCall", req, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Code returns the runtime bytecode deployed at an EVM address.
func (s *EVMService) Code(ctx context.Context, address string) (*EVMCodeResponse, *Response, error) {
	var out EVMCodeResponse
	resp, err := s.client.get(ctx, "tachi_evmCode", url.Values{"address": {address}}, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Receipt returns the receipt of a committed EVM tx by its 32-byte hash.
// Returns a 400 *ErrorResponse if there is no receipt for hash.
func (s *EVMService) Receipt(ctx context.Context, hash string) (*EVMReceiptResponse, *Response, error) {
	var out EVMReceiptResponse
	resp, err := s.client.get(ctx, "tachi_evmReceipt", url.Values{"hash": {hash}}, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Account returns an EVM account's nonce and code summary. address is a
// 0x EVM address, or a Tachi pubkey hex (x-only, compressed or
// uncompressed), which is mapped to its EVM address.
func (s *EVMService) Account(ctx context.Context, address string) (*EVMAccountResponse, *Response, error) {
	var out EVMAccountResponse
	resp, err := s.client.get(ctx, "tachi_evmAccount", url.Values{"address": {address}}, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Storage returns the 32-byte value at a storage slot of an EVM address.
// slot is hex of up to 32 bytes; the daemon left-pads it.
func (s *EVMService) Storage(ctx context.Context, address, slot string) (*EVMStorageResponse, *Response, error) {
	var out EVMStorageResponse
	resp, err := s.client.get(ctx, "tachi_evmStorage", url.Values{"address": {address}, "slot": {slot}}, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
