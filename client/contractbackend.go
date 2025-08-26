package client

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/jhttp"
	"perun.network/perun-stellar-backend/event"
	"sync"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"perun.network/go-perun/wallet"

	chTypes "perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wallet/types"
	"perun.network/perun-stellar-backend/wire"
)

const stellarDefaultChainID = 2

// Sender is an interface for sending transactions.
type Sender interface {
	SignSendTx(txnbuild.Transaction) (xdr.TransactionMeta, string, error)
	SetHzClient(*horizonclient.Client)
}

// ContractBackend is a struct that implements the ContractBackend interface.
type ContractBackend struct {
	Invoker
	tr      StellarSigner
	chainID int
	cbMutex sync.Mutex
}

// NewContractBackend creates a new ContractBackend.
func NewContractBackend(trConfig *TransactorConfig) *ContractBackend {
	transactor := NewTransactor(*trConfig)
	return &ContractBackend{
		tr:      *transactor,
		chainID: stellarDefaultChainID,
		cbMutex: sync.Mutex{},
	}
}

// StellarSigner is a struct that implements the Transactor interface for Stellar.
type StellarSigner struct {
	keyPair     *keypair.Full
	participant *types.Participant
	account     *wallet.Account
	hzClient    *horizonclient.Client
	sender      Sender
}

// TransactorConfig is a struct that contains the configuration for the Transactor.
type TransactorConfig struct {
	keyPair     *keypair.Full
	participant *types.Participant
	account     *wallet.Account
	sender      Sender
	horizonURL  string
}

// SetKeyPair sets the keypair of the TransactorConfig.
func (tc *TransactorConfig) SetKeyPair(kp *keypair.Full) {
	tc.keyPair = kp
}

// SetParticipant sets the participant of the TransactorConfig.
func (tc *TransactorConfig) SetParticipant(participant *types.Participant) {
	tc.participant = participant
}

// SetAccount sets the account of the TransactorConfig.
func (tc *TransactorConfig) SetAccount(account wallet.Account) {
	tc.account = &account
}

// SetSender sets the sender of the TransactorConfig.
func (tc *TransactorConfig) SetSender(sender Sender) {
	tc.sender = sender
}

// SetHorizonURL sets the horizon URL of the TransactorConfig.
func (tc *TransactorConfig) SetHorizonURL(url string) {
	tc.horizonURL = url
}

// NewTransactor creates a new Transactor using the transactor configuration.
func NewTransactor(cfg TransactorConfig) *StellarSigner {
	st := &StellarSigner{}

	if cfg.sender != nil {
		st.sender = cfg.sender
	} else {
		st.sender = &TxSender{}
	}

	if cfg.keyPair != nil {
		st.keyPair = cfg.keyPair
		if txSender, ok := st.sender.(*TxSender); ok {
			txSender.kp = st.keyPair
		}
	}
	if cfg.participant != nil {
		st.participant = cfg.participant
	}
	if cfg.account != nil {
		st.account = cfg.account
	}

	if cfg.horizonURL != "" {
		st.hzClient = NewHorizonClient(cfg.horizonURL)
	} else {
		st.hzClient = NewHorizonClient(HorizonURL)
	}

	return st
}

// GetTransactor returns the transactor of the ContractBackend.
func (c *ContractBackend) GetTransactor() *StellarSigner {
	return &c.tr
}

// GetBalance returns the balance of the contract.
func (c *ContractBackend) GetBalance(cID xdr.ScAddress) (string, error) {
	tr := c.GetTransactor()
	add, err := tr.GetAddress()
	if err != nil {
		return "", err
	}
	accountID, err := xdr.AddressToAccountId(add)
	if err != nil {
		return "", err
	}
	scAdd, err := xdr.NewScAddress(xdr.ScAddressTypeScAddressTypeAccount, accountID)
	if err != nil {
		return "", err
	}
	TokenNameArgs, err := BuildGetTokenBalanceArgs(scAdd)
	if err != nil {
		return "", err
	}
	_, bal, err := c.InvokeUnsignedTx("balance", TokenNameArgs, cID)
	if err != nil {
		return "", err
	}
	return bal, nil
}

// GetHorizonAccount returns the horizon account of the StellarSigner.
func (st *StellarSigner) GetHorizonAccount() (horizon.Account, error) {
	hzAddress, err := st.GetAddress()
	if err != nil {
		return horizon.Account{}, err
	}
	accountReq := horizonclient.AccountRequest{AccountID: hzAddress}
	hzAccount, err := st.hzClient.AccountDetail(accountReq)
	if err != nil {
		return hzAccount, err
	}
	return hzAccount, nil
}

// GetAddress returns the address of the StellarSigner.
func (st *StellarSigner) GetAddress() (string, error) {
	if st.keyPair != nil {
		return st.keyPair.Address(), nil
	}
	if st.account != nil {
		return (*st.account).Address().String(), nil
	}
	if st.participant != nil {
		return st.participant.AddressString(), nil
	}

	return "", errors.New("transactor cannot retrieve address")
}

// GetHorizonClient returns the horizon client of the StellarSigner.
func (st *StellarSigner) GetHorizonClient() *horizonclient.Client {
	return st.hzClient
}

// InvokeUnsignedTx invokes an unsigned transaction.
func (c *ContractBackend) InvokeUnsignedTx(fname string, callTxArgs xdr.ScVec, contractAddr xdr.ScAddress) (wire.Channel, string, error) { // xdr.TransactionMeta, error
	c.cbMutex.Lock()
	defer c.cbMutex.Unlock()
	fnameXdr := xdr.ScSymbol(fname)
	hzAcc, err := c.tr.GetHorizonAccount()
	if err != nil {
		return wire.Channel{}, "", err
	}

	hzClient := c.tr.GetHorizonClient()

	c.tr.sender.SetHzClient(hzClient)
	chanInf := fname == "get_channel"

	invokeHostFunctionOp := BuildContractCallOp(hzAcc, fnameXdr, callTxArgs, contractAddr)
	chanInfo, bal, _, _, err := PreflightHostFunctionsResult(hzClient, &hzAcc, *invokeHostFunctionOp, chanInf)
	if err != nil {
		return wire.Channel{}, "", err
	}

	return chanInfo, bal, nil
}

// InvokeSignedTx invokes a signed transaction.
func (c *ContractBackend) InvokeSignedTx(fname string, callTxArgs xdr.ScVec, contractAddr xdr.ScAddress) (xdr.TransactionMeta, string, error) {
	c.cbMutex.Lock()
	defer c.cbMutex.Unlock()
	fnameXdr := xdr.ScSymbol(fname)
	hzAcc, err := c.tr.GetHorizonAccount()
	if err != nil {
		return xdr.TransactionMeta{}, "", errors.Join(errors.New("failed to get horizon account"), err)
	}

	hzClient := c.tr.GetHorizonClient()

	c.tr.sender.SetHzClient(hzClient)
	invokeHostFunctionOp := BuildContractCallOp(hzAcc, fnameXdr, callTxArgs, contractAddr)
	preFlightOp, minFee, err := PreflightHostFunctions(hzClient, &hzAcc, *invokeHostFunctionOp)
	if err != nil {
		return xdr.TransactionMeta{}, "", err
	}
	minFeeCustom := int64(100) //nolint:gomnd
	txParams := GetBaseTransactionParamsWithFee(&hzAcc, minFee+minFeeCustom, &preFlightOp)
	txUnsigned, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return xdr.TransactionMeta{}, "", errors.Join(errors.New("error building Transaction"), err)
	}
	txMeta, hash, err := c.tr.sender.SignSendTx(*txUnsigned)
	if err != nil {
		return xdr.TransactionMeta{}, "", errors.Join(errors.New("sending tx"), err)
	}

	return txMeta, hash, nil
}

type rpcGetTxResp struct {
	Status                string  `json:"status"` // SUCCESS | FAILED | NOT_FOUND | PENDING
	LatestLedger          uint32  `json:"latestLedger"`
	LatestLedgerCloseTime string  `json:"latestLedgerCloseTime"`
	OldestLedger          uint32  `json:"oldestLedger"`
	OldestLedgerCloseTime string  `json:"oldestLedgerCloseTime"`
	Ledger                *uint32 `json:"ledger,omitempty"`

	// Optional: base64 XDR of meta/envelope/result if you still want them
	ResultMetaXdr string `json:"resultMetaXdr,omitempty"`
	EnvelopeXdr   string `json:"envelopeXdr,omitempty"`
	ResultXdr     string `json:"resultXdr,omitempty"`

	// Optional diagnostics (only if RPC enables them)
	DiagnosticEventsXdr []string `json:"diagnosticEventsXdr,omitempty"`

	// Contract + transaction events live here
	Events struct {
		TransactionEventsXdr []string   `json:"transactionEventsXdr"`
		ContractEventsXdr    [][]string `json:"contractEventsXdr"`
	} `json:"events"`
}

func decodeDiagEventsB64(b64s []string) ([]xdr.DiagnosticEvent, error) {
	out := make([]xdr.DiagnosticEvent, 0, len(b64s))
	for _, s := range b64s {
		var ev xdr.DiagnosticEvent
		if err := xdr.SafeUnmarshalBase64(s, &ev); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, nil
}

// base64 -> ContractEvent[]
func decodeContractEventsB64(nested [][]string) ([]xdr.ContractEvent, error) {
	flat := make([]string, 0, 64)
	for _, op := range nested {
		flat = append(flat, op...)
	}

	out := make([]xdr.ContractEvent, 0, len(flat))
	for _, s := range flat {
		var ev xdr.ContractEvent
		if err := xdr.SafeUnmarshalBase64(s, &ev); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, nil
}

func (c *ContractBackend) FetchTxDiagEvents(ctx context.Context, txHashHex string) ([]event.PerunEvent, error) {
	// Build the same RPC client you use for simulate
	hzClient := c.tr.GetHorizonClient()

	c.tr.sender.SetHzClient(hzClient)
	var link string
	if hzClient.HorizonURL == horizonClientURL {
		link = sorobanRPCLink
	} else {
		link = sorobanTestnet
	}
	ch := jhttp.NewChannel(link, nil) // keep rpcURL in your backend
	rpc := jrpc2.NewClient(ch, nil)

	var r rpcGetTxResp
	if err := rpc.CallResult(ctx, "getTransaction", struct {
		Hash string `json:"hash"`
	}{txHashHex}, &r); err != nil {
		return nil, err
	}
	// Prefer diagnostics if present (simpler, single list)
	if len(r.DiagnosticEventsXdr) > 0 {
		diag, err := decodeDiagEventsB64(r.DiagnosticEventsXdr)
		if err != nil {
			return nil, err
		}
		return event.DecodePerunFromDiag(diag)
	}

	// Fallback to contract events
	ce, err := decodeContractEventsB64(r.Events.ContractEventsXdr)
	if err != nil {
		return nil, err
	}
	return event.DecodePerunFromContract(ce)
}

// StringToScAddress converts a string to a xdr.ScAddress.
func StringToScAddress(s string) (xdr.ScAddress, error) {
	hash, err := StringToHash(s)
	if err != nil {
		return xdr.ScAddress{}, err
	}
	return chTypes.MakeContractAddress(xdr.ContractId(hash))
}

// StringToHash converts a hex string to a xdr.Hash.
func StringToHash(s string) (xdr.Hash, error) {
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return xdr.Hash{}, fmt.Errorf("failed to decode hex string: %w", err)
	}
	var hash xdr.Hash
	copy(hash[:], bytes)
	return hash, nil
}
