package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	MaxMempoolSize = 10_000
	MaxTxPerBlock  = 1_000
	TxExpiry       = 3600
)

type PendingTx struct {
	Hash      string    `json:"hash"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    int64     `json:"amount"`
	Fee       int64     `json:"fee"`
	Nonce     uint64    `json:"nonce"`
	Timestamp time.Time `json:"timestamp"`
	Priority  int64     `json:"priority"`
	Status    string    `json:"status"`
}

type MempoolStats struct {
	Size       int   `json:"size"`
	MaxSize    int   `json:"max_size"`
	MinFee     int64 `json:"min_fee"`
	MaxFee     int64 `json:"max_fee"`
	AvgFee     int64 `json:"avg_fee"`
	MaxTxBlock int   `json:"max_tx_per_block"`
}

type Mempool struct {
	mu      sync.RWMutex
	pending map[string]*PendingTx
	size    int
}

func NewMempool() *Mempool {
	mp := &Mempool{
		pending: make(map[string]*PendingTx),
	}
	go mp.cleanupRoutine()
	return mp
}

func (mp *Mempool) Submit(from, to string, amount, fee int64, nonce uint64) (*PendingTx, error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.size >= MaxMempoolSize {
		return nil, fmt.Errorf("Mempool voll (%d/%d)", mp.size, MaxMempoolSize)
	}
	if fee < 100 {
		return nil, fmt.Errorf("Fee zu niedrig: minimum 100 unvx")
	}

	hashInput := fmt.Sprintf("%s:%s:%d:%d:%d:%d", from, to, amount, fee, nonce, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(hashInput))
	txHash := "0x" + hex.EncodeToString(hash[:])

	tx := &PendingTx{
		Hash:      txHash,
		From:      from,
		To:        to,
		Amount:    amount,
		Fee:       fee,
		Nonce:     nonce,
		Timestamp: time.Now().UTC(),
		Priority:  fee,
		Status:    "pending",
	}

	mp.pending[txHash] = tx
	mp.size++

	fmt.Printf("[Mempool] TX eingereicht | Hash: %s... | Fee: %d | Pool: %d\n",
		txHash[:18], fee, mp.size)

	return tx, nil
}

func (mp *Mempool) SelectForBlock(maxTxs int) []*PendingTx {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if maxTxs > MaxTxPerBlock {
		maxTxs = MaxTxPerBlock
	}

	candidates := make([]*PendingTx, 0, len(mp.pending))
	now := time.Now()

	for _, tx := range mp.pending {
		if now.Sub(tx.Timestamp).Seconds() > TxExpiry {
			continue
		}
		candidates = append(candidates, tx)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority > candidates[j].Priority
	})

	if len(candidates) > maxTxs {
		candidates = candidates[:maxTxs]
	}

	for _, tx := range candidates {
		tx.Status = "confirmed"
		delete(mp.pending, tx.Hash)
		mp.size--
	}

	return candidates
}

func (mp *Mempool) GetPending(limit int) []*PendingTx {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	txs := make([]*PendingTx, 0, len(mp.pending))
	for _, tx := range mp.pending {
		txs = append(txs, tx)
	}

	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Priority > txs[j].Priority
	})

	if limit > 0 && limit < len(txs) {
		return txs[:limit]
	}
	return txs
}

func (mp *Mempool) Size() int {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.size
}

func (mp *Mempool) Stats() MempoolStats {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	stats := MempoolStats{
		Size:       mp.size,
		MaxSize:    MaxMempoolSize,
		MaxTxBlock: MaxTxPerBlock,
		MinFee:     100,
	}

	if mp.size == 0 {
		return stats
	}

	var total, minF, maxF int64
	first := true
	for _, tx := range mp.pending {
		total += tx.Fee
		if first {
			minF = tx.Fee
			maxF = tx.Fee
			first = false
		} else {
			if tx.Fee < minF { minF = tx.Fee }
			if tx.Fee > maxF { maxF = tx.Fee }
		}
	}

	stats.MinFee = minF
	stats.MaxFee = maxF
	stats.AvgFee = total / int64(mp.size)
	return stats
}

func (mp *Mempool) cleanupRoutine() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		mp.mu.Lock()
		now := time.Now()
		expired := 0
		for hash, tx := range mp.pending {
			if now.Sub(tx.Timestamp).Seconds() > TxExpiry {
				delete(mp.pending, hash)
				mp.size--
				expired++
			}
		}
		if expired > 0 {
			fmt.Printf("[Mempool] %d abgelaufene TXs entfernt\n", expired)
		}
		mp.mu.Unlock()
	}
}
