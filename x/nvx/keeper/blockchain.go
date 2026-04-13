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

// ─────────────────────────────────────────────
//  Block — die grundlegende Einheit der Chain
// ─────────────────────────────────────────────

type Block struct {
	// Header
	Height        int64  `json:"height"`
	Hash          string `json:"hash"`
	PrevHash      string `json:"prev_hash"`
	Timestamp     string `json:"timestamp"`
	Validator     string `json:"validator"`

	// Inhalt
	Transactions  []string `json:"transactions"`
	TxCount       int      `json:"tx_count"`
	TotalFees     int64    `json:"total_fees"`
	BurnedAmount  int64    `json:"burned_amount"`
	MiningReward  int64    `json:"mining_reward"`

	// Konsens
	StateRoot     string `json:"state_root"`
	Signature     string `json:"signature"`
}

// ─────────────────────────────────────────────
//  Blockchain — die vollständige Chain
// ─────────────────────────────────────────────

type Blockchain struct {
	mu       sync.RWMutex
	blocks   []*Block
	dbPath   string
	indexPath string
}

func NewBlockchain(dbPath string) *Blockchain {
	bc := &Blockchain{
		blocks:    make([]*Block, 0),
		dbPath:    dbPath,
		indexPath: dbPath + ".index",
	}
	bc.load()

	// Genesis Block erstellen falls leer
	if len(bc.blocks) == 0 {
		bc.createGenesisBlock()
	}

	return bc
}

// ─────────────────────────────────────────────
//  Genesis Block
// ─────────────────────────────────────────────

func (bc *Blockchain) createGenesisBlock() {
	genesis := &Block{
		Height:       0,
		PrevHash:     "0000000000000000000000000000000000000000000000000000000000000000",
		Timestamp:    time.Date(2025, 10, 1, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Validator:    "nuvex19d85718c8da8f4213e1a2a41fe894ba928b9c9",
		Transactions: []string{},
		TxCount:      0,
		TotalFees:    0,
		BurnedAmount: 0,
		MiningReward: 50_000_000,
		StateRoot:    "GENESIS_STATE_ROOT",
	}

	genesis.Hash = bc.calculateHash(genesis)

	bc.blocks = append(bc.blocks, genesis)
	bc.save()

	fmt.Println("[Blockchain] ✅ Genesis Block erstellt")
	fmt.Printf("[Blockchain] Hash: %s\n", genesis.Hash[:32]+"...")
}

// ─────────────────────────────────────────────
//  Neuen Block hinzufügen
// ─────────────────────────────────────────────

func (bc *Blockchain) AddBlock(
	validator string,
	txHashes []string,
	totalFees int64,
	burnedAmount int64,
	miningReward int64,
) *Block {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	prevBlock := bc.blocks[len(bc.blocks)-1]

	block := &Block{
		Height:       prevBlock.Height + 1,
		PrevHash:     prevBlock.Hash,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Validator:    validator,
		Transactions: txHashes,
		TxCount:      len(txHashes),
		TotalFees:    totalFees,
		BurnedAmount: burnedAmount,
		MiningReward: miningReward,
		StateRoot:    bc.calculateStateRoot(txHashes),
	}

	block.Hash = bc.calculateHash(block)
	bc.blocks = append(bc.blocks, block)

	// Sofort auf Disk speichern
	bc.appendBlock(block)

	fmt.Printf("[Blockchain] Block #%d | Hash: %s... | Txs: %d | Reward: %.4f NVX\n",
		block.Height,
		block.Hash[:16],
		block.TxCount,
		float64(block.MiningReward)/1_000_000,
	)

	return block
}

// ─────────────────────────────────────────────
//  Hash Berechnung
// ─────────────────────────────────────────────

func (bc *Blockchain) calculateHash(b *Block) string {
	data := fmt.Sprintf("%d%s%s%s%d%d%d%s",
		b.Height,
		b.PrevHash,
		b.Timestamp,
		b.Validator,
		b.TxCount,
		b.TotalFees,
		b.MiningReward,
		b.StateRoot,
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (bc *Blockchain) calculateStateRoot(txHashes []string) string {
	h := sha256.New()
	for _, tx := range txHashes {
		h.Write([]byte(tx))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ─────────────────────────────────────────────
//  Chain Validierung
// ─────────────────────────────────────────────

func (bc *Blockchain) ValidateChain() error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	for i := 1; i < len(bc.blocks); i++ {
		current := bc.blocks[i]
		previous := bc.blocks[i-1]

		// Hash prüfen
		expectedHash := bc.calculateHash(current)
		if current.Hash != expectedHash {
			return fmt.Errorf("❌ Block #%d: Hash ungültig", current.Height)
		}

		// Verkettung prüfen
		if current.PrevHash != previous.Hash {
			return fmt.Errorf("❌ Block #%d: Verkettung gebrochen", current.Height)
		}

		// Höhe prüfen
		if current.Height != previous.Height+1 {
			return fmt.Errorf("❌ Block #%d: Höhe falsch", current.Height)
		}
	}

	fmt.Printf("[Blockchain] ✅ Chain validiert — %d Blöcke korrekt\n", len(bc.blocks))
	return nil
}

// ─────────────────────────────────────────────
//  Abfragen
// ─────────────────────────────────────────────

func (bc *Blockchain) GetLatestBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if len(bc.blocks) == 0 {
		return nil
	}
	return bc.blocks[len(bc.blocks)-1]
}

func (bc *Blockchain) GetBlock(height int64) *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	for _, b := range bc.blocks {
		if b.Height == height {
			return b
		}
	}
	return nil
}

func (bc *Blockchain) GetRecentBlocks(limit int) []*Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	total := len(bc.blocks)
	if limit > total {
		limit = total
	}
	result := make([]*Block, limit)
	for i := 0; i < limit; i++ {
		result[i] = bc.blocks[total-1-i]
	}
	return result
}

func (bc *Blockchain) Height() int64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if len(bc.blocks) == 0 {
		return 0
	}
	return bc.blocks[len(bc.blocks)-1].Height
}

func (bc *Blockchain) TotalBlocks() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.blocks)
}

// ─────────────────────────────────────────────
//  Persistenz — Speichern & Laden
// ─────────────────────────────────────────────

// Einzelnen Block anhängen (effizient — kein komplettes Rewrite)
func (bc *Blockchain) appendBlock(block *Block) {
	data, err := json.Marshal(block)
	if err != nil {
		return
	}

	f, err := os.OpenFile(bc.dbPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	f.Write(data)
	f.WriteString("\n")
}

// Komplette Chain speichern
func (bc *Blockchain) save() {
	f, err := os.Create(bc.dbPath)
	if err != nil {
		return
	}
	defer f.Close()

	for _, block := range bc.blocks {
		data, _ := json.Marshal(block)
		f.Write(data)
		f.WriteString("\n")
	}
}

// Chain von Disk laden
func (bc *Blockchain) load() {
	data, err := os.ReadFile(bc.dbPath)
	if err != nil {
		return
	}

	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var block Block
		if err := json.Unmarshal(line, &block); err != nil {
			continue
		}
		bc.blocks = append(bc.blocks, &block)
	}

	if len(bc.blocks) > 0 {
		fmt.Printf("[Blockchain] ✅ %d Blöcke geladen (Height: %d)\n",
			len(bc.blocks),
			bc.blocks[len(bc.blocks)-1].Height,
		)
	}
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
