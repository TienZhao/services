package btc

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"

	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"github.com/skycoin/skycoin/src/cipher"
)

const (
	defaultAddr = "23.92.24.9"
	defaultUser = "YnWD3EmQAOw11IOrUJwWxAThAyobwLC"
	defaultPass = `f*Z"[1215o{qKW{Buj/wheO8@h.}j*u`
	defaultCert = `-----BEGIN CERTIFICATE-----
MIICbTCCAc+gAwIBAgIRAKnAvGj6JobKblRUcmxOqxowCgYIKoZIzj0EAwQwNjEg
MB4GA1UEChMXYnRjZCBhdXRvZ2VuZXJhdGVkIGNlcnQxEjAQBgNVBAMTCWxvY2Fs
aG9zdDAeFw0xNzExMDYwNTMzNDRaFw0yNzExMDUwNTMzNDRaMDYxIDAeBgNVBAoT
F2J0Y2QgYXV0b2dlbmVyYXRlZCBjZXJ0MRIwEAYDVQQDEwlsb2NhbGhvc3QwgZsw
EAYHKoZIzj0CAQYFK4EEACMDgYYABAEYn5Xj5QfV6vK6jjeLnG63H5U8yrga5wYJ
bqBhuSR+540zqVjviZQXDi9OVTcYffDk+VrP2KmD8Q8FW2yFAjo2ewA63DHQibtJ
Jb2bSCSJnMa7MqWeYle61oIwt9wIiq+9gjVIagnlEAOVm86TBeuiCgUu5t3k1CrI
R4XFVPAgDQXnzqN7MHkwDgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB/wQFMAMBAf8w
VgYDVR0RBE8wTYIJbG9jYWxob3N0hwR/AAABhxAAAAAAAAAAAAAAAAAAAAABhwQX
XBgJhxAmADwBAAAAAPA8kf/+zLGFhxD+gAAAAAAAAPA8kf/+zLGFMAoGCCqGSM49
BAMEA4GLADCBhwJCATk6kLPOcQh5V5r6SVcmcPUhOKRu54Ip/wrtagAFN5WDqm/T
rVUFT9wbSwqLaJfVBhCe14PWx3jR7+EXJJLv8R3sAkEK79/zPd3sHJc0pIM7SDQX
FZAzYmyXme/Ki0138hSmFvby/r7NeNmcJUZRj1+fWXMgfPv7/kZ0ScpsRqY34AP2
ig==
-----END CERTIFICATE-----`
	defaultBlockExplorer         = "https://api.blockcypher.com"
	walletBalanceDefaultEndpoint = "/v1/btc/main/addrs/"
	txStatusDefaultEndpoint      = "/v1/btc/main/txs/"
)

// ServiceBtc encapsulates operations with bitcoin
type ServiceBtc struct {
	nodeAddress string
	client      *rpcclient.Client
	// Circuit breaker related fields
	isOpen        uint32
	openTimeout   time.Duration
	retryCount    uint
	blockExplorer string
}

type TxStatus struct {
	Amount        float64 `json:"amount"`
	Confirmations int64   `json:"confirmations"`
	Fee           float64 `json:"fee"`

	BlockHash  string `json:"blockhash"`
	BlockIndex int64  `json:"block_index"`

	Hash      string `json:"hash"`
	Confirmed int64  `json:"confirmed"`
	Received  int64  `json:"received"`
}

type explorerTxStatus struct {
	Total         float64 `json:"total"`
	Fees          float64 `json:"fees"`
	Confirmations int64   `json:"confirmations"`

	BlockHash  string `json:"block_hash"`
	BlockIndex int64  `json:"block_index"`

	Hash      string    `json:"hash"`
	Confirmed time.Time `json:"confirmed"`
	Received  time.Time `json:"received"`
}

type explorerAddressResponse struct {
	Address            string  `json:"address"`
	TotalReceived      int     `json:"total_received"`
	TotalSent          int     `json:"total_sent"`
	Balance            int64   `json:"balance"`
	UnconfirmedBalance float64 `json:"unconfirmed_balance"`
	FinalBalance       float64 `json:"final_balance"`
	NTx                int     `json:"n_tx"`
}

// NewBTCService returns ServiceBtc instance
func NewBTCService(btcAddr, btcUser, btcPass string, disableTLS bool, cert []byte, blockExplorer string) (*ServiceBtc, error) {
	if len(btcAddr) == 0 {
		btcAddr = defaultAddr
	}

	if len(btcUser) == 0 {
		btcUser = defaultUser
	}

	if len(btcPass) == 0 {
		btcPass = defaultPass
	}

	if !disableTLS && len(cert) == 0 {
		cert = []byte(defaultCert)
	}

	if len(blockExplorer) == 0 {
		blockExplorer = defaultBlockExplorer
	}

	client, err := rpcclient.New(&rpcclient.ConnConfig{
		HTTPPostMode: true,
		DisableTLS:   disableTLS,
		Host:         btcAddr,
		User:         btcUser,
		Pass:         btcPass,
		Certificates: cert,
	}, nil)

	if err != nil {
		//TODO: handle that stuff more meaningful way
		return nil, errors.New(fmt.Sprintf("error creating new btc client: %v", err))
	}

	return &ServiceBtc{
		nodeAddress:   btcAddr,
		client:        client,
		retryCount:    3,
		openTimeout:   time.Second * 10,
		isOpen:        0,
		blockExplorer: blockExplorer,
	}, nil
}

// GenerateAddr generates an address for bitcoin
func (s ServiceBtc) GenerateAddr(publicKey cipher.PubKey) (string, error) {
	address := cipher.BitcoinAddressFromPubkey(publicKey)

	return address, nil
}

// GenerateKeyPair generates keypair for bitcoin
func (s ServiceBtc) GenerateKeyPair() (cipher.PubKey, cipher.SecKey) {
	seed := make([]byte, 256)
	rand.Read(seed)

	pub, sec := cipher.GenerateDeterministicKeyPair(seed)

	return pub, sec
}

// CheckBalance checks a balance for given bitcoin wallet
func (s *ServiceBtc) CheckBalance(address string) (float64, error) {
	// If breaker is open - get info from block explorer
	if s.isOpen == 1 {
		balance, err := s.getBalanceFromExplorer(address)

		if err != nil {
			return 0, err
		}

		return balance, nil
	}

	var i uint = 0

	balance, err := s.getBalanceFromNode(address)
	if err != nil {
		log.Printf("Get balance from node returned error %s", err.Error())
	}

	for i < s.retryCount && err != nil {
		if err != nil {
			log.Printf("Get balance from node returned error %s", err.Error())
		}

		balance, err = s.getBalanceFromNode(address)

		if err != nil {
			time.Sleep(time.Millisecond * time.Duration(1<<i) * 100)
		}
		i++
	}

	if err != nil {
		s.isOpen = 1

		go func() {
			time.Sleep(s.openTimeout)
			// This assignment is atomic since on 64-bit platforms this operation is atomic
			s.isOpen = 0
		}()

		balance, err := s.getBalanceFromExplorer(address)

		if err != nil {
			return 0.0, err
		}

		return balance, nil
	}

	return balance, nil
}

func (s *ServiceBtc) CheckTxStatus(txId string) (*TxStatus, error) {
	// If breaker is open - get info from block explorer
	if s.isOpen == 1 {
		status, err := s.getTxStatusFromExplorer(txId)

		if err != nil {
			return nil, err
		}

		return status, nil
	}

	var i uint = 0

	status, err := s.getTxStatusFromNode(txId)
	if err != nil {
		log.Printf("Get status from node returned error %s", err.Error())
	}

	for i < s.retryCount && err != nil {
		if err != nil {
			log.Printf("Get status from node returned error %s", err.Error())
		}

		status, err = s.getTxStatusFromNode(txId)

		if err != nil {
			time.Sleep(time.Millisecond * time.Duration(1<<i) * 100)
		}
		i++
	}

	if err != nil {
		s.isOpen = 1

		go func() {
			time.Sleep(s.openTimeout)
			// This assignment is atomic since on 64-bit platforms this operation is atomic
			s.isOpen = 0
		}()

		status, err := s.getTxStatusFromExplorer(txId)

		if err != nil {
			return status, err
		}

		return status, nil
	}

	return status, nil
}

func (s *ServiceBtc) getTxStatusFromNode(txId string) (*TxStatus, error) {
	hash, err := chainhash.NewHash([]byte(txId))

	if err != nil {
		return nil, err
	}

	rawTx, err := s.client.GetTransaction(hash)

	if err != nil {
		return nil, err
	}

	txStatus := &TxStatus{
		Amount:        rawTx.Amount,
		Confirmations: rawTx.Confirmations,
		Fee:           rawTx.Fee,

		BlockHash:  rawTx.BlockHash,
		BlockIndex: rawTx.BlockIndex,

		Hash:      rawTx.TxID,
		Confirmed: rawTx.Time,
		Received:  rawTx.TimeReceived,
	}

	if err != nil {
		return nil, err
	}

	return txStatus, nil
}

func (s *ServiceBtc) getTxStatusFromExplorer(txId string) (*TxStatus, error) {
	url := s.blockExplorer + txStatusDefaultEndpoint + txId
	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	explorerResp := &explorerTxStatus{}
	err = json.Unmarshal(data, explorerResp)

	if err != nil {
		return nil, err
	}

	txStatus := &TxStatus{
		// NOTE(stgleb): amount goes in satoshis
		Amount:        explorerResp.Total,
		Confirmations: explorerResp.Confirmations,
		Fee:           explorerResp.Fees,

		BlockHash:  explorerResp.BlockHash,
		BlockIndex: explorerResp.BlockIndex,

		Hash:      explorerResp.Hash,
		Confirmed: explorerResp.Confirmed.Unix(),
		Received:  explorerResp.Received.Unix(),
	}

	return txStatus, nil
}

func (s *ServiceBtc) getBalanceFromNode(address string) (float64, error) {
	// First get an address in proper form
	a, err := btcutil.DecodeAddress(address, &chaincfg.MainNetParams)

	if err != nil {
		return 0.0, err
	}

	log.Printf("Get account of address %s", address)
	account, err := s.client.GetAccount(a)

	if err != nil {
		return 0.0, err
	}

	log.Printf("Send request for getting balance of address %s", address)
	amount, err := s.client.GetBalance(account)

	if err != nil {
		return 0.0, errors.New(fmt.Sprintf("error creating new btc client: %v", err))
	}

	log.Printf("Balance is equal to %f", amount)
	balance := amount.ToUnit(btcutil.AmountSatoshi)

	return balance, nil
}

func (s *ServiceBtc) getBalanceFromExplorer(address string) (float64, error) {
	url := s.blockExplorer + walletBalanceDefaultEndpoint + address
	resp, err := http.Get(url)

	if err != nil {
		return 0, err
	}

	var r explorerAddressResponse

	err = json.NewDecoder(resp.Body).Decode(&r)

	if err != nil {
		return 0, err
	}

	return r.FinalBalance, nil
}

// Api method for monitoring btc service circuit breaker
func (s *ServiceBtc) IsOpen() bool {
	return s.isOpen == 1
}

func (s *ServiceBtc) GetHost() string {
	return s.nodeAddress
}