package keeper

import (
	"fmt"
	"sync"
)

type BurnKeeper struct {
	mu          sync.RWMutex
	totalBurned int64
	circulating int64
}

func NewBurnKeeper() *BurnKeeper {
	return &BurnKeeper{circulating: 500_000_000_000_000}
}

func (bk *BurnKeeper) Burn(amount int64) {
	bk.mu.Lock()
	defer bk.mu.Unlock()
	bk.totalBurned += amount
	bk.circulating -= amount
	fmt.Printf("[Nuvex] Burned %d unvx | Total burned: %d\n", amount, bk.totalBurned)
}

func (bk *BurnKeeper) BurnFromFee(fee int64, txHash string, height int64) {
	burnAmount := fee * 10 / 10_000
	if burnAmount > 0 {
		bk.Burn(burnAmount)
	}
}

func (bk *BurnKeeper) Stats() (burned int64, circulating int64) {
	bk.mu.RLock()
	defer bk.mu.RUnlock()
	return bk.totalBurned, bk.circulating
}
