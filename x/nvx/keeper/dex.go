package keeper

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  Nuvex DEX — Decentralized Exchange
//
//  Automated Market Maker (AMM) — wie Uniswap v2
//
//  Funktioniert nach der Formel:
//  x * y = k  (Constant Product Formula)
//
//  Beispiel:
//  Pool hat 1000 NVX und 1000 USDT
//  k = 1000 * 1000 = 1,000,000
//  Jemand kauft 100 NVX:
//  Neues x = 900 NVX
//  Neues y = k / 900 = 1111 USDT
//  Preis = 111 USDT für 100 NVX
// ─────────────────────────────────────────────

const (
	// Fee: 0.3% pro Swap (wie Uniswap)
	SwapFeeBps = 30 // 30 basis points = 0.3%

	// Minimum Liquidity — verhindert Division durch 0
	MinimumLiquidity = 1000
)

// LiquidityPool repräsentiert ein Handelspaar
type LiquidityPool struct {
	// ID ist der eindeutige Identifier (z.B. "NVX-USDT")
	ID string `json:"id"`

	// TokenA ist der erste Token (meist NVX)
	TokenA string `json:"token_a"`

	// TokenB ist der zweite Token
	TokenB string `json:"token_b"`

	// ReserveA ist die Menge von TokenA im Pool
	ReserveA float64 `json:"reserve_a"`

	// ReserveB ist die Menge von TokenB im Pool
	ReserveB float64 `json:"reserve_b"`

	// TotalLPTokens ist die Gesamtmenge der LP Tokens
	// LP Tokens = Beweis dass jemand Liquidität bereitgestellt hat
	TotalLPTokens float64 `json:"total_lp_tokens"`

	// LPHolders verfolgt wer wie viele LP Tokens hat
	LPHolders map[string]float64 `json:"lp_holders"`

	// TotalVolume ist das gesamte Handelsvolumen
	TotalVolume float64 `json:"total_volume"`

	// TotalFees ist die Gesamtmenge gesammelter Fees
	TotalFees float64 `json:"total_fees"`

	// CreatedAt ist wann der Pool erstellt wurde
	CreatedAt string `json:"created_at"`

	// Trades ist die Anzahl aller Swaps
	Trades uint64 `json:"trades"`
}

// SwapResult ist das Ergebnis eines Swaps
type SwapResult struct {
	// AmountIn ist wie viel der User einzahlt
	AmountIn float64 `json:"amount_in"`

	// AmountOut ist wie viel der User bekommt
	AmountOut float64 `json:"amount_out"`

	// Fee ist die bezahlte Fee
	Fee float64 `json:"fee"`

	// PriceImpact ist wie stark der Swap den Preis bewegt (in %)
	PriceImpact float64 `json:"price_impact"`

	// NewPrice ist der Preis nach dem Swap
	NewPrice float64 `json:"new_price"`

	// TokenIn ist welcher Token eingezahlt wird
	TokenIn string `json:"token_in"`

	// TokenOut ist welcher Token rauskommt
	TokenOut string `json:"token_out"`

	// PoolID ist welcher Pool verwendet wurde
	PoolID string `json:"pool_id"`

	// Timestamp ist wann der Swap stattfand
	Timestamp string `json:"timestamp"`

	// Success zeigt ob der Swap erfolgreich war
	Success bool `json:"success"`

	// Error enthält Fehlermeldungen
	Error string `json:"error,omitempty"`
}

// AddLiquidityResult ist das Ergebnis des Hinzufügens von Liquidität
type AddLiquidityResult struct {
	PoolID        string  `json:"pool_id"`
	TokenA        string  `json:"token_a"`
	TokenB        string  `json:"token_b"`
	AmountA       float64 `json:"amount_a"`
	AmountB       float64 `json:"amount_b"`
	LPTokens      float64 `json:"lp_tokens_received"`
	Provider      string  `json:"provider"`
	Success       bool    `json:"success"`
	Error         string  `json:"error,omitempty"`
}

// PoolStats gibt Statistiken eines Pools zurück
type PoolStats struct {
	PoolID       string  `json:"pool_id"`
	TokenA       string  `json:"token_a"`
	TokenB       string  `json:"token_b"`
	ReserveA     float64 `json:"reserve_a"`
	ReserveB     float64 `json:"reserve_b"`
	Price        float64 `json:"price_a_in_b"`
	TotalVolume  float64 `json:"total_volume"`
	TotalFees    float64 `json:"total_fees"`
	Trades       uint64  `json:"trades"`
	LPProviders  int     `json:"lp_providers"`
	TVL          float64 `json:"tvl_in_token_b"`
}

// DEXKeeper verwaltet den gesamten DEX
type DEXKeeper struct {
	mu    sync.RWMutex
	pools map[string]*LiquidityPool

	// swapHistory speichert alle Swaps
	swapHistory []SwapResult
}

// NewDEXKeeper erstellt einen neuen DEX
func NewDEXKeeper() *DEXKeeper {
	dk := &DEXKeeper{
		pools:       make(map[string]*LiquidityPool),
		swapHistory: make([]SwapResult, 0),
	}

	fmt.Println("[DEX] ✅ Nuvex DEX gestartet")
	fmt.Println("[DEX] ✅ AMM Model: Constant Product (x*y=k)")
	fmt.Printf("[DEX] ✅ Swap Fee: 0.%d%%\n", SwapFeeBps/10)

	return dk
}

// ─────────────────────────────────────────────
//  Pool erstellen
// ─────────────────────────────────────────────

// CreatePool erstellt einen neuen Liquiditätspool
func (dk *DEXKeeper) CreatePool(
	tokenA string,
	tokenB string,
	initialAmountA float64,
	initialAmountB float64,
	creator string,
) (*LiquidityPool, error) {
	dk.mu.Lock()
	defer dk.mu.Unlock()

	poolID := tokenA + "-" + tokenB

	// Pool bereits vorhanden?
	if _, exists := dk.pools[poolID]; exists {
		return nil, fmt.Errorf("Pool %s existiert bereits", poolID)
	}

	if initialAmountA <= 0 || initialAmountB <= 0 {
		return nil, fmt.Errorf("Initiale Liquidität muss positiv sein")
	}

	// LP Tokens = sqrt(amountA * amountB) — wie Uniswap
	lpTokens := math.Sqrt(initialAmountA*initialAmountB) - MinimumLiquidity
	if lpTokens <= 0 {
		return nil, fmt.Errorf("Zu wenig initiale Liquidität")
	}

	pool := &LiquidityPool{
		ID:            poolID,
		TokenA:        tokenA,
		TokenB:        tokenB,
		ReserveA:      initialAmountA,
		ReserveB:      initialAmountB,
		TotalLPTokens: lpTokens + MinimumLiquidity,
		LPHolders:     map[string]float64{creator: lpTokens},
		TotalVolume:   0,
		TotalFees:     0,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		Trades:        0,
	}

	dk.pools[poolID] = pool

	initialPrice := initialAmountB / initialAmountA

	fmt.Printf("[DEX] ✅ Pool erstellt: %s | %s Reserve: %.2f | %s Reserve: %.2f | Preis: %.4f\n",
		poolID, tokenA, initialAmountA, tokenB, initialAmountB, initialPrice)

	return pool, nil
}

// ─────────────────────────────────────────────
//  Liquidität hinzufügen
// ─────────────────────────────────────────────

// AddLiquidity fügt Liquidität zu einem Pool hinzu
func (dk *DEXKeeper) AddLiquidity(
	poolID string,
	amountA float64,
	amountB float64,
	provider string,
) (*AddLiquidityResult, error) {
	dk.mu.Lock()
	defer dk.mu.Unlock()

	pool, exists := dk.pools[poolID]
	if !exists {
		return nil, fmt.Errorf("Pool nicht gefunden: %s", poolID)
	}

	if amountA <= 0 || amountB <= 0 {
		return nil, fmt.Errorf("Beträge müssen positiv sein")
	}

	// LP Tokens berechnen (proportional zur bestehenden Liquidität)
	lpTokens := math.Min(
		amountA/pool.ReserveA*pool.TotalLPTokens,
		amountB/pool.ReserveB*pool.TotalLPTokens,
	)

	// Reserven updaten
	pool.ReserveA += amountA
	pool.ReserveB += amountB
	pool.TotalLPTokens += lpTokens

	// LP Tokens dem Provider gutschreiben
	pool.LPHolders[provider] += lpTokens

	fmt.Printf("[DEX] Liquidität hinzugefügt: %s | %.2f %s + %.2f %s | LP: %.4f\n",
		poolID, amountA, pool.TokenA, amountB, pool.TokenB, lpTokens)

	return &AddLiquidityResult{
		PoolID:   poolID,
		TokenA:   pool.TokenA,
		TokenB:   pool.TokenB,
		AmountA:  amountA,
		AmountB:  amountB,
		LPTokens: lpTokens,
		Provider: provider,
		Success:  true,
	}, nil
}

// ─────────────────────────────────────────────
//  Liquidität entfernen
// ─────────────────────────────────────────────

// RemoveLiquidity entfernt Liquidität aus einem Pool
func (dk *DEXKeeper) RemoveLiquidity(
	poolID string,
	lpTokens float64,
	provider string,
) (float64, float64, error) {
	dk.mu.Lock()
	defer dk.mu.Unlock()

	pool, exists := dk.pools[poolID]
	if !exists {
		return 0, 0, fmt.Errorf("Pool nicht gefunden: %s", poolID)
	}

	providerLP := pool.LPHolders[provider]
	if providerLP < lpTokens {
		return 0, 0, fmt.Errorf("Nicht genug LP Tokens: hat %.4f, braucht %.4f",
			providerLP, lpTokens)
	}

	// Anteil berechnen
	share := lpTokens / pool.TotalLPTokens
	amountA := share * pool.ReserveA
	amountB := share * pool.ReserveB

	// Reserven updaten
	pool.ReserveA -= amountA
	pool.ReserveB -= amountB
	pool.TotalLPTokens -= lpTokens
	pool.LPHolders[provider] -= lpTokens

	fmt.Printf("[DEX] Liquidität entfernt: %s | %.2f %s + %.2f %s\n",
		poolID, amountA, pool.TokenA, amountB, pool.TokenB)

	return amountA, amountB, nil
}

// ─────────────────────────────────────────────
//  Swap — der Kernmechanismus
// ─────────────────────────────────────────────

// Swap tauscht Token A gegen Token B (oder umgekehrt)
func (dk *DEXKeeper) Swap(
	poolID string,
	tokenIn string,
	amountIn float64,
	minAmountOut float64,
	trader string,
) (*SwapResult, error) {
	dk.mu.Lock()
	defer dk.mu.Unlock()

	pool, exists := dk.pools[poolID]
	if !exists {
		return nil, fmt.Errorf("Pool nicht gefunden: %s", poolID)
	}

	if amountIn <= 0 {
		return nil, fmt.Errorf("AmountIn muss positiv sein")
	}

	// Bestimme welcher Token rein und raus geht
	var reserveIn, reserveOut float64
	var tokenOut string

	if tokenIn == pool.TokenA {
		reserveIn = pool.ReserveA
		reserveOut = pool.ReserveB
		tokenOut = pool.TokenB
	} else if tokenIn == pool.TokenB {
		reserveIn = pool.ReserveB
		reserveOut = pool.ReserveA
		tokenOut = pool.TokenA
	} else {
		return nil, fmt.Errorf("Token %s nicht in Pool %s", tokenIn, poolID)
	}

	// Fee berechnen (0.3%)
	fee := amountIn * float64(SwapFeeBps) / 10_000
	amountInAfterFee := amountIn - fee

	// Constant Product Formula: x * y = k
	// amountOut = reserveOut - k / (reserveIn + amountInAfterFee)
	k := reserveIn * reserveOut
	newReserveIn := reserveIn + amountInAfterFee
	newReserveOut := k / newReserveIn
	amountOut := reserveOut - newReserveOut

	// Slippage Schutz
	if amountOut < minAmountOut {
		return &SwapResult{
			Success: false,
			Error:   fmt.Sprintf("AmountOut %.4f unter Minimum %.4f (Slippage)", amountOut, minAmountOut),
		}, nil
	}

	// Price Impact berechnen
	oldPrice := reserveOut / reserveIn
	newPrice := newReserveOut / newReserveIn
	priceImpact := math.Abs(oldPrice-newPrice) / oldPrice * 100

	// Reserven updaten
	if tokenIn == pool.TokenA {
		pool.ReserveA += amountInAfterFee
		pool.ReserveB = newReserveOut
	} else {
		pool.ReserveB += amountInAfterFee
		pool.ReserveA = newReserveOut
	}

	// Statistiken updaten
	pool.TotalVolume += amountIn
	pool.TotalFees += fee
	pool.Trades++

	result := &SwapResult{
		AmountIn:    amountIn,
		AmountOut:   amountOut,
		Fee:         fee,
		PriceImpact: priceImpact,
		NewPrice:    newPrice,
		TokenIn:     tokenIn,
		TokenOut:    tokenOut,
		PoolID:      poolID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Success:     true,
	}

	dk.swapHistory = append(dk.swapHistory, *result)

	fmt.Printf("[DEX] Swap: %.4f %s → %.4f %s | Fee: %.4f | Impact: %.2f%% | Trader: %s\n",
		amountIn, tokenIn, amountOut, tokenOut, fee, priceImpact, trader[:10]+"...")

	return result, nil
}

// ─────────────────────────────────────────────
//  Preis berechnen (Quote)
// ─────────────────────────────────────────────

// GetQuote berechnet wie viel man für einen Swap bekommt (ohne auszuführen)
func (dk *DEXKeeper) GetQuote(
	poolID string,
	tokenIn string,
	amountIn float64,
) (*SwapResult, error) {
	dk.mu.RLock()
	defer dk.mu.RUnlock()

	pool, exists := dk.pools[poolID]
	if !exists {
		return nil, fmt.Errorf("Pool nicht gefunden: %s", poolID)
	}

	var reserveIn, reserveOut float64
	var tokenOut string

	if tokenIn == pool.TokenA {
		reserveIn = pool.ReserveA
		reserveOut = pool.ReserveB
		tokenOut = pool.TokenB
	} else {
		reserveIn = pool.ReserveB
		reserveOut = pool.ReserveA
		tokenOut = pool.TokenA
	}

	fee := amountIn * float64(SwapFeeBps) / 10_000
	amountInAfterFee := amountIn - fee
	k := reserveIn * reserveOut
	newReserveIn := reserveIn + amountInAfterFee
	newReserveOut := k / newReserveIn
	amountOut := reserveOut - newReserveOut

	oldPrice := reserveOut / reserveIn
	newPrice := newReserveOut / newReserveIn
	priceImpact := math.Abs(oldPrice-newPrice) / oldPrice * 100

	return &SwapResult{
		AmountIn:    amountIn,
		AmountOut:   amountOut,
		Fee:         fee,
		PriceImpact: priceImpact,
		NewPrice:    newPrice,
		TokenIn:     tokenIn,
		TokenOut:    tokenOut,
		PoolID:      poolID,
		Success:     true,
	}, nil
}

// ─────────────────────────────────────────────
//  Statistiken & Abfragen
// ─────────────────────────────────────────────

// GetPoolStats gibt Statistiken eines Pools zurück
func (dk *DEXKeeper) GetPoolStats(poolID string) (*PoolStats, error) {
	dk.mu.RLock()
	defer dk.mu.RUnlock()

	pool, exists := dk.pools[poolID]
	if !exists {
		return nil, fmt.Errorf("Pool nicht gefunden: %s", poolID)
	}

	price := 0.0
	if pool.ReserveA > 0 {
		price = pool.ReserveB / pool.ReserveA
	}

	return &PoolStats{
		PoolID:      pool.ID,
		TokenA:      pool.TokenA,
		TokenB:      pool.TokenB,
		ReserveA:    pool.ReserveA,
		ReserveB:    pool.ReserveB,
		Price:       price,
		TotalVolume: pool.TotalVolume,
		TotalFees:   pool.TotalFees,
		Trades:      pool.Trades,
		LPProviders: len(pool.LPHolders),
		TVL:         pool.ReserveB * 2,
	}, nil
}

// GetAllPools gibt alle Pools zurück
func (dk *DEXKeeper) GetAllPools() []*LiquidityPool {
	dk.mu.RLock()
	defer dk.mu.RUnlock()
	pools := make([]*LiquidityPool, 0, len(dk.pools))
	for _, p := range dk.pools {
		pools = append(pools, p)
	}
	return pools
}

// GetSwapHistory gibt die letzten Swaps zurück
func (dk *DEXKeeper) GetSwapHistory(limit int) []SwapResult {
	dk.mu.RLock()
	defer dk.mu.RUnlock()
	total := len(dk.swapHistory)
	if limit > total { limit = total }
	result := make([]SwapResult, limit)
	for i := 0; i < limit; i++ {
		result[i] = dk.swapHistory[total-1-i]
	}
	return result
}

// DEXStats gibt globale DEX Statistiken zurück
type DEXStats struct {
	TotalPools   int     `json:"total_pools"`
	TotalTrades  uint64  `json:"total_trades"`
	TotalVolume  float64 `json:"total_volume"`
	TotalFees    float64 `json:"total_fees"`
	SwapFeeBps   int     `json:"swap_fee_bps"`
}

func (dk *DEXKeeper) Stats() DEXStats {
	dk.mu.RLock()
	defer dk.mu.RUnlock()
	stats := DEXStats{
		TotalPools: len(dk.pools),
		SwapFeeBps: SwapFeeBps,
	}
	for _, p := range dk.pools {
		stats.TotalTrades += p.Trades
		stats.TotalVolume += p.TotalVolume
		stats.TotalFees += p.TotalFees
	}
	return stats
}

// ToJSON serialisiert einen Pool zu JSON
func (p *LiquidityPool) ToJSON() string {
	data, _ := json.MarshalIndent(p, "", "  ")
	return string(data)
}
