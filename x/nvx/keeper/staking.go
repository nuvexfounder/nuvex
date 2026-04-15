package keeper

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  Nuvex Staking System
//
//  NVX Holder können ihre Coins staken und
//  Rewards verdienen.
//
//  APY: 10% pro Jahr (linear)
//  Minimum Stake: 100 NVX
//  Lock Period: 0 (flexible staking)
//  Compound: täglich
// ─────────────────────────────────────────────

const (
	// Basis APY: 10% pro Jahr
	BaseAPY = 0.10

	// Bonus für grüne Validators: +0.5%
	GreenBonus = 0.005

	// Minimum Stake Betrag
	MinStake = 100_000_000 // 100 NVX in unvx

	// Rewards werden täglich berechnet
	RewardInterval = 24 * time.Hour
)

// StakePosition repräsentiert eine Staking Position
type StakePosition struct {
	// ID ist der eindeutige Identifier
	ID string `json:"id"`

	// Owner ist die Nuvex Adresse
	Owner string `json:"owner"`

	// Amount ist die gestakte Menge in unvx
	Amount uint64 `json:"amount"`

	// StartTime ist wann der Stake begann
	StartTime time.Time `json:"start_time"`

	// LastReward ist wann zuletzt Rewards berechnet wurden
	LastReward time.Time `json:"last_reward"`

	// TotalRewards sind alle bisherigen Rewards
	TotalRewards uint64 `json:"total_rewards"`

	// IsGreen gibt an ob der Staker grüne Energie nutzt
	IsGreen bool `json:"is_green"`

	// IsActive zeigt ob die Position aktiv ist
	IsActive bool `json:"is_active"`
}

// StakingStats gibt globale Staking Statistiken
type StakingStats struct {
	TotalStaked     uint64  `json:"total_staked"`
	TotalStakers    int     `json:"total_stakers"`
	TotalRewardsPaid uint64 `json:"total_rewards_paid"`
	BaseAPY         float64 `json:"base_apy"`
	GreenAPY        float64 `json:"green_apy"`
	MinStake        uint64  `json:"min_stake"`
}

// StakingKeeper verwaltet alle Staking Positionen
type StakingKeeper struct {
	mu        sync.RWMutex
	positions map[string]*StakePosition
	state     *StateKeeper
}

// NewStakingKeeper erstellt einen neuen Staking Keeper
func NewStakingKeeper(state *StateKeeper) *StakingKeeper {
	sk := &StakingKeeper{
		positions: make(map[string]*StakePosition),
		state:     state,
	}

	fmt.Println("[Staking] ✅ Nuvex Staking System gestartet")
	fmt.Printf("[Staking] ✅ Base APY: %.0f%%\n", BaseAPY*100)
	fmt.Printf("[Staking] ✅ Green Bonus: +%.1f%%\n", GreenBonus*100)
	fmt.Printf("[Staking] ✅ Min Stake: %d NVX\n", MinStake/1_000_000)

	// Starte automatische Reward Berechnung
	go sk.rewardLoop()

	return sk
}

// ─────────────────────────────────────────────
//  Staken
// ─────────────────────────────────────────────

// Stake erstellt eine neue Staking Position
func (sk *StakingKeeper) Stake(
	owner string,
	amount uint64,
	isGreen bool,
) (*StakePosition, error) {
	sk.mu.Lock()
	defer sk.mu.Unlock()

	// Minimum prüfen
	if amount < MinStake {
		return nil, fmt.Errorf("Minimum Stake: %d NVX — du hast %d NVX",
			MinStake/1_000_000, amount/1_000_000)
	}

	// Balance prüfen
	balance := uint64(sk.state.GetBalance(owner))
	if balance < amount {
		return nil, fmt.Errorf("Nicht genug NVX: hat %d, braucht %d",
			balance/1_000_000, amount/1_000_000)
	}

	// Balance abziehen
	sk.state.mu.Lock()
	sk.state.accounts[owner].Balance -= int64(amount)
	sk.state.mu.Unlock()

	// Position ID
	id := fmt.Sprintf("%s_%d", owner[:10], time.Now().UnixNano())

	position := &StakePosition{
		ID:           id,
		Owner:        owner,
		Amount:       amount,
		StartTime:    time.Now().UTC(),
		LastReward:   time.Now().UTC(),
		TotalRewards: 0,
		IsGreen:      isGreen,
		IsActive:     true,
	}

	sk.positions[id] = position

	apy := BaseAPY
	if isGreen {
		apy += GreenBonus
	}

	fmt.Printf("[Staking] ✅ Stake: %s | %d NVX | APY: %.1f%% | Green: %v\n",
		owner[:10]+"...", amount/1_000_000, apy*100, isGreen)

	return position, nil
}

// ─────────────────────────────────────────────
//  Unstaken
// ─────────────────────────────────────────────

// Unstake gibt die gestakten Coins zurück
func (sk *StakingKeeper) Unstake(
	owner string,
	positionID string,
) (*StakePosition, error) {
	sk.mu.Lock()
	defer sk.mu.Unlock()

	position, exists := sk.positions[positionID]
	if !exists {
		return nil, fmt.Errorf("Position nicht gefunden: %s", positionID)
	}

	if position.Owner != owner {
		return nil, fmt.Errorf("Nicht deine Position")
	}

	if !position.IsActive {
		return nil, fmt.Errorf("Position bereits inaktiv")
	}

	// Letzte Rewards berechnen
	sk.calculateRewards(position)

	// Stake zurückgeben
	totalReturn := position.Amount + position.TotalRewards
	sk.state.mu.Lock()
	sk.state.accounts[owner].Balance += int64(totalReturn)
	sk.state.mu.Unlock()

	position.IsActive = false

	fmt.Printf("[Staking] Unstake: %s | %d NVX + %d Rewards zurück\n",
		owner[:10]+"...", position.Amount/1_000_000,
		position.TotalRewards/1_000_000)

	return position, nil
}

// ─────────────────────────────────────────────
//  Reward Berechnung
// ─────────────────────────────────────────────

// calculateRewards berechnet die Rewards für eine Position
func (sk *StakingKeeper) calculateRewards(pos *StakePosition) uint64 {
	now := time.Now().UTC()
	elapsed := now.Sub(pos.LastReward)

	// APY berechnen
	apy := BaseAPY
	if pos.IsGreen {
		apy += GreenBonus
	}

	// Tägliche Rate
	dailyRate := apy / 365.0

	// Elapsed in Tagen
	days := elapsed.Hours() / 24.0

	// Rewards = Amount * dailyRate * days
	rewards := float64(pos.Amount) * dailyRate * days
	rewardsUint := uint64(math.Floor(rewards))

	if rewardsUint > 0 {
		pos.TotalRewards += rewardsUint
		pos.LastReward = now

		// Rewards dem State hinzufügen (compound)
		fmt.Printf("[Staking] Reward: %s | +%d unvx\n",
			pos.Owner[:10]+"...", rewardsUint)
	}

	return rewardsUint
}

// rewardLoop berechnet automatisch Rewards alle 1 Stunde
func (sk *StakingKeeper) rewardLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		sk.mu.Lock()
		total := uint64(0)
		for _, pos := range sk.positions {
			if pos.IsActive {
				total += sk.calculateRewards(pos)
			}
		}
		if total > 0 {
			fmt.Printf("[Staking] Hourly rewards distributed: %d unvx total\n", total)
		}
		sk.mu.Unlock()
	}
}

// ─────────────────────────────────────────────
//  Abfragen
// ─────────────────────────────────────────────

// GetPosition gibt eine Staking Position zurück
func (sk *StakingKeeper) GetPosition(id string) (*StakePosition, error) {
	sk.mu.RLock()
	defer sk.mu.RUnlock()
	pos, exists := sk.positions[id]
	if !exists {
		return nil, fmt.Errorf("Position nicht gefunden")
	}
	return pos, nil
}

// GetPositionsByOwner gibt alle Positionen eines Owners zurück
func (sk *StakingKeeper) GetPositionsByOwner(owner string) []*StakePosition {
	sk.mu.RLock()
	defer sk.mu.RUnlock()
	positions := make([]*StakePosition, 0)
	for _, pos := range sk.positions {
		if pos.Owner == owner {
			positions = append(positions, pos)
		}
	}
	return positions
}

// GetStats gibt globale Staking Statistiken zurück
func (sk *StakingKeeper) GetStats() StakingStats {
	sk.mu.RLock()
	defer sk.mu.RUnlock()

	stats := StakingStats{
		BaseAPY:  BaseAPY,
		GreenAPY: BaseAPY + GreenBonus,
		MinStake: MinStake,
	}

	for _, pos := range sk.positions {
		if pos.IsActive {
			stats.TotalStaked += pos.Amount
			stats.TotalStakers++
		}
		stats.TotalRewardsPaid += pos.TotalRewards
	}

	return stats
}

// PendingRewards berechnet ausstehende Rewards
func (sk *StakingKeeper) PendingRewards(positionID string) uint64 {
	sk.mu.RLock()
	defer sk.mu.RUnlock()

	pos, exists := sk.positions[positionID]
	if !exists || !pos.IsActive {
		return 0
	}

	apy := BaseAPY
	if pos.IsGreen {
		apy += GreenBonus
	}

	elapsed := time.Since(pos.LastReward)
	days := elapsed.Hours() / 24.0
	dailyRate := apy / 365.0
	rewards := float64(pos.Amount) * dailyRate * days

	return uint64(math.Floor(rewards))
}

// ToJSON serialisiert eine Position
func (p *StakePosition) ToJSON() string {
	data, _ := json.MarshalIndent(p, "", "  ")
	return string(data)
}
