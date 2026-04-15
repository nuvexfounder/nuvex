package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// ─────────────────────────────────────────────
//  Nuvex LevelDB State Storage
//
//  Ersetzt JSON File mit LevelDB:
//  - 100x schneller bei vielen Accounts
//  - Skaliert auf Millionen von Accounts
//  - Wie Bitcoin und Ethereum intern
//
//  Key Schema:
//  acc:{address}  → Account JSON
//  tx:{hash}      → Transaction JSON
//  txidx:{height}:{hash} → TX Index
// ─────────────────────────────────────────────

type LevelDBStateKeeper struct {
	mu     sync.RWMutex
	db     *leveldb.DB
	dbPath string

	// In-Memory Cache für schnelle Reads
	accountCache map[string]*Account
	cacheSize    int
}

// NewLevelDBStateKeeper erstellt einen neuen LevelDB State Keeper
func NewLevelDBStateKeeper(dbPath string) (*LevelDBStateKeeper, error) {
	db, err := leveldb.OpenFile(dbPath+".leveldb", nil)
	if err != nil {
		return nil, fmt.Errorf("LevelDB Fehler: %w", err)
	}

	sk := &LevelDBStateKeeper{
		db:           db,
		dbPath:       dbPath,
		accountCache: make(map[string]*Account),
		cacheSize:    1000,
	}

	fmt.Println("[State] ✅ LevelDB State Storage gestartet")

	count := sk.countAccounts()
	txCount := sk.countTransactions()
	fmt.Printf("[State] ✅ Geladen: %d Konten, %d Transaktionen\n", count, txCount)

	return sk, nil
}

// ─────────────────────────────────────────────
//  Account Operationen
// ─────────────────────────────────────────────

func (sk *LevelDBStateKeeper) GetAccount(address string) *Account {
	sk.mu.RLock()
	if acc, exists := sk.accountCache[address]; exists {
		sk.mu.RUnlock()
		return acc
	}
	sk.mu.RUnlock()

	sk.mu.Lock()
	defer sk.mu.Unlock()

	key := []byte("acc:" + address)
	data, err := sk.db.Get(key, nil)
	if err != nil {
		return nil
	}

	var acc Account
	if err := json.Unmarshal(data, &acc); err != nil {
		return nil
	}

	sk.accountCache[address] = &acc
	return &acc
}

func (sk *LevelDBStateKeeper) SetAccount(acc *Account) error {
	data, err := json.Marshal(acc)
	if err != nil {
		return err
	}

	key := []byte("acc:" + acc.Address)
	if err := sk.db.Put(key, data, nil); err != nil {
		return err
	}

	sk.accountCache[acc.Address] = acc
	return nil
}

func (sk *LevelDBStateKeeper) GetBalanceLDB(address string) int64 {
	acc := sk.GetAccount(address)
	if acc == nil {
		return 0
	}
	return acc.Balance
}

// ─────────────────────────────────────────────
//  Transaction Operationen
// ─────────────────────────────────────────────

func (sk *LevelDBStateKeeper) SaveTransaction(tx *Transaction) error {
	data, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	// TX nach Hash speichern
	txKey := []byte("tx:" + tx.Hash)
	if err := sk.db.Put(txKey, data, nil); err != nil {
		return err
	}

	// TX Index nach Height
	idxKey := []byte(fmt.Sprintf("txidx:%010d:%s", tx.Height, tx.Hash))
	if err := sk.db.Put(idxKey, []byte(tx.Hash), nil); err != nil {
		return err
	}

	// TX Index nach Address (From)
	fromKey := []byte(fmt.Sprintf("txaddr:%s:%010d:%s", tx.From, tx.Height, tx.Hash))
	if err := sk.db.Put(fromKey, []byte(tx.Hash), nil); err != nil {
		return err
	}

	// TX Index nach Address (To)
	toKey := []byte(fmt.Sprintf("txaddr:%s:%010d:%s", tx.To, tx.Height, tx.Hash))
	if err := sk.db.Put(toKey, []byte(tx.Hash), nil); err != nil {
		return err
	}

	return nil
}

func (sk *LevelDBStateKeeper) GetTransaction(hash string) (*Transaction, error) {
	key := []byte("tx:" + hash)
	data, err := sk.db.Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("TX nicht gefunden: %s", hash)
	}

	var tx Transaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

// GetTransactionsByAddress gibt alle TXs einer Adresse zurück
func (sk *LevelDBStateKeeper) GetTransactionsByAddress(address string, limit int) []Transaction {
	sk.mu.RLock()
	defer sk.mu.RUnlock()

	prefix := []byte("txaddr:" + address + ":")
	iter := sk.db.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()

	txs := make([]Transaction, 0)
	count := 0

	for iter.Last(); iter.Valid() && count < limit; iter.Prev() {
		txHash := string(iter.Value())
		tx, err := sk.GetTransaction(txHash)
		if err == nil {
			txs = append(txs, *tx)
			count++
		}
	}

	return txs
}

// GetRecentTransactions gibt die letzten N Transaktionen zurück
func (sk *LevelDBStateKeeper) GetRecentTransactions(limit int) []Transaction {
	sk.mu.RLock()
	defer sk.mu.RUnlock()

	prefix := []byte("txidx:")
	iter := sk.db.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()

	txs := make([]Transaction, 0)
	count := 0

	for iter.Last(); iter.Valid() && count < limit; iter.Prev() {
		txHash := string(iter.Value())
		tx, err := sk.GetTransaction(txHash)
		if err == nil {
			txs = append(txs, *tx)
			count++
		}
	}

	return txs
}

// ─────────────────────────────────────────────
//  Transfer mit LevelDB
// ─────────────────────────────────────────────

func (sk *LevelDBStateKeeper) TransferLDB(
	from, to string,
	amount, fee, height int64,
) (*Transaction, error) {
	sk.mu.Lock()
	defer sk.mu.Unlock()

	sender := sk.GetAccount(from)
	if sender == nil {
		return nil, fmt.Errorf("Adresse nicht gefunden: %s", from)
	}

	total := amount + fee
	if sender.Balance < total {
		return nil, fmt.Errorf("Nicht genug NVX: hat %d, braucht %d",
			sender.Balance, total)
	}

	receiver := sk.GetAccount(to)
	if receiver == nil {
		receiver = &Account{Address: to, Balance: 0}
	}

	sender.Balance -= total
	receiver.Balance += amount
	sender.Nonce++

	if err := sk.SetAccount(sender); err != nil {
		return nil, err
	}
	if err := sk.SetAccount(receiver); err != nil {
		return nil, err
	}

	hashInput := fmt.Sprintf("%s%s%d%d%d", from, to, amount, fee, sender.Nonce)
	hash := sha256.Sum256([]byte(hashInput))
	txHash := "0x" + hex.EncodeToString(hash[:])

	tx := &Transaction{
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

	if err := sk.SaveTransaction(tx); err != nil {
		return nil, err
	}

	fmt.Printf("[LevelDB TX] %s → %s | %d unvx\n",
		from[:16]+"...", to[:16]+"...", amount)

	return tx, nil
}

// ─────────────────────────────────────────────
//  Statistiken
// ─────────────────────────────────────────────

func (sk *LevelDBStateKeeper) countAccounts() int {
	iter := sk.db.NewIterator(util.BytesPrefix([]byte("acc:")), nil)
	defer iter.Release()
	count := 0
	for iter.Next() {
		count++
	}
	return count
}

func (sk *LevelDBStateKeeper) countTransactions() int {
	iter := sk.db.NewIterator(util.BytesPrefix([]byte("tx:")), nil)
	defer iter.Release()
	count := 0
	for iter.Next() {
		count++
	}
	return count
}

func (sk *LevelDBStateKeeper) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"accounts":     sk.countAccounts(),
		"transactions": sk.countTransactions(),
		"storage":      "LevelDB",
		"cache_size":   len(sk.accountCache),
	}
}

func (sk *LevelDBStateKeeper) Close() {
	sk.db.Close()
}
