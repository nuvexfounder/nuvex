package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Account struct {
	Address   string `json:"address"`
	Balance   int64  `json:"balance"`
	Nonce     uint64 `json:"nonce"`
	PublicKey string `json:"public_key,omitempty"`
}

type Transaction struct {
	Hash      string `json:"hash"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    int64  `json:"amount"`
	Fee       int64  `json:"fee"`
	Nonce     uint64 `json:"nonce"`
	Height    int64  `json:"height"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
	Signature string `json:"signature,omitempty"`
}

type StateKeeper struct {
	mu       sync.RWMutex
	accounts map[string]*Account
	txs      []Transaction
	dbPath   string
}

func NewStateKeeper(dbPath string) *StateKeeper {
	sk := &StateKeeper{
		accounts: make(map[string]*Account),
		txs:      make([]Transaction, 0),
		dbPath:   dbPath,
	}
	sk.load()
	return sk
}

func (sk *StateKeeper) InitGenesis(accounts []GenesisEntry) {
	sk.mu.Lock()
	defer sk.mu.Unlock()
	for _, acc := range accounts {
		if _, exists := sk.accounts[acc.Address]; !exists {
			sk.accounts[acc.Address] = &Account{
				Address:   acc.Address,
				Balance:   acc.Amount,
				Nonce:     0,
				PublicKey: acc.PublicKey,
			}
			fmt.Printf("[State] Genesis: %s = %d unvx\n", acc.Address, acc.Amount)
		}
	}
	sk.save()
}

type GenesisEntry struct {
	Address   string
	Amount    int64
	PublicKey string
}

func (sk *StateKeeper) GetBalance(address string) int64 {
	sk.mu.RLock()
	defer sk.mu.RUnlock()
	if acc, exists := sk.accounts[address]; exists {
		return acc.Balance
	}
	return 0
}

// SignedTransfer: Transaktion NUR mit gültiger Signatur
func (sk *StateKeeper) SignedTransfer(
	from, to string,
	amount, fee int64,
	height int64,
	publicKeyHex string,
	signatureHex string,
) (*Transaction, error) {
	sk.mu.Lock()
	defer sk.mu.Unlock()

	sender, exists := sk.accounts[from]
	if !exists {
		return nil, fmt.Errorf("Adresse nicht gefunden: %s", from)
	}

	// ── SIGNATUR PRÜFEN ──────────────────────────────
	payload := TxPayload{
		From:   from,
		To:     to,
		Amount: amount,
		Fee:    fee,
		Nonce:  sender.Nonce + 1,
	}

	if err := VerifySignature(publicKeyHex, payload, signatureHex); err != nil {
		return nil, fmt.Errorf("SIGNATUR UNGÜLTIG: %w", err)
	}

	// ── ADRESSE PRÜFEN ───────────────────────────────
	derivedAddr, err := AddressFromPublicKey(publicKeyHex)
	if err != nil {
		return nil, fmt.Errorf("Public Key Fehler: %w", err)
	}

	if derivedAddr != from {
		return nil, fmt.Errorf("Public Key passt nicht zur Adresse")
	}

	// ── BALANCE PRÜFEN ───────────────────────────────
	total := amount + fee
	if sender.Balance < total {
		return nil, fmt.Errorf("Nicht genug NVX: hat %d, braucht %d",
			sender.Balance, total)
	}

	// ── TRANSFER AUSFÜHREN ───────────────────────────
	if _, exists := sk.accounts[to]; !exists {
		sk.accounts[to] = &Account{Address: to, Balance: 0}
	}

	sender.Balance -= total
	sk.accounts[to].Balance += amount
	sender.Nonce++

	// TX Hash
	hashInput := fmt.Sprintf("%s%s%d%d%d%s",
		from, to, amount, fee, sender.Nonce, signatureHex)
	hash := sha256.Sum256([]byte(hashInput))
	txHash := "0x" + hex.EncodeToString(hash[:])

	tx := Transaction{
		Hash:      txHash,
		From:      from,
		To:        to,
		Amount:    amount,
		Fee:       fee,
		Nonce:     sender.Nonce,
		Height:    height,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Status:    "confirmed",
		Signature: signatureHex[:16] + "...",
	}

	sk.txs = append(sk.txs, tx)
	sk.save()

	fmt.Printf("[TX ✅] %s → %s | %d unvx | Signatur: VALID\n",
		from[:16]+"...", to[:16]+"...", amount)

	return &tx, nil
}

// Unsignierte TX — nur für interne Genesis Transfers
func (sk *StateKeeper) Transfer(from, to string, amount, fee, height int64) (*Transaction, error) {
	sk.mu.Lock()
	defer sk.mu.Unlock()

	sender, exists := sk.accounts[from]
	if !exists {
		return nil, fmt.Errorf("Adresse nicht gefunden: %s", from)
	}

	total := amount + fee
	if sender.Balance < total {
		return nil, fmt.Errorf("Nicht genug NVX: hat %d, braucht %d", sender.Balance, total)
	}

	if _, exists := sk.accounts[to]; !exists {
		sk.accounts[to] = &Account{Address: to, Balance: 0}
	}

	sender.Balance -= total
	sk.accounts[to].Balance += amount
	sender.Nonce++

	hashInput := fmt.Sprintf("%s%s%d%d%d", from, to, amount, fee, sender.Nonce)
	hash := sha256.Sum256([]byte(hashInput))
	txHash := "0x" + hex.EncodeToString(hash[:])

	tx := Transaction{
		Hash:      txHash,
		From:      from,
		To:        to,
		Amount:    amount,
		Fee:       fee,
		Nonce:     sender.Nonce,
		Height:    height,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Status:    "confirmed",
	}

	sk.txs = append(sk.txs, tx)
	sk.save()

	fmt.Printf("[TX] %s → %s | %d unvx\n", from[:16]+"...", to[:16]+"...", amount)
	return &tx, nil
}

func (sk *StateKeeper) GetTransactions(limit int) []Transaction {
	sk.mu.RLock()
	defer sk.mu.RUnlock()
	if limit > len(sk.txs) {
		limit = len(sk.txs)
	}
	result := make([]Transaction, limit)
	total := len(sk.txs)
	for i := 0; i < limit; i++ {
		result[i] = sk.txs[total-1-i]
	}
	return result
}

func (sk *StateKeeper) GetAllAccounts() []*Account {
	sk.mu.RLock()
	defer sk.mu.RUnlock()
	accounts := make([]*Account, 0)
	for _, acc := range sk.accounts {
		accounts = append(accounts, acc)
	}
	return accounts
}

func (sk *StateKeeper) save() {
	type State struct {
		Accounts map[string]*Account `json:"accounts"`
		Txs      []Transaction       `json:"transactions"`
	}
	data, _ := json.MarshalIndent(State{
		Accounts: sk.accounts,
		Txs:      sk.txs,
	}, "", "  ")
	os.WriteFile(sk.dbPath, data, 0600)
}

func (sk *StateKeeper) load() {
	type State struct {
		Accounts map[string]*Account `json:"accounts"`
		Txs      []Transaction       `json:"transactions"`
	}
	data, err := os.ReadFile(sk.dbPath)
	if err != nil {
		return
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}
	sk.accounts = state.Accounts
	sk.txs = state.Txs
	fmt.Printf("[State] Geladen: %d Konten, %d Transaktionen\n",
		len(sk.accounts), len(sk.txs))
}
