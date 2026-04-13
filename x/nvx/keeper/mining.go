package keeper

import (
	"fmt"
	"time"
)

// ─────────────────────────────────────────────
//  Nuvex Mining & Halving System
//
//  Wie Bitcoin — aber grün:
//  - Keine energiefressenden ASICs
//  - Staking = Mining (Proof of Stake)
//  - Belohnung halbiert sich alle 5 Jahre
//  - Macht NVX mit der Zeit knapper
// ─────────────────────────────────────────────

const (
	// Initiale Block-Belohnung: 50 NVX
	InitialBlockReward = int64(50_000_000) // 50 NVX in micro-NVX

	// Halving alle 5 Jahre
	// Bei 400ms pro Block = ~78,840,000 Blöcke pro 5 Jahre
	BlocksPerHalving = int64(78_840_000)

	// Genesis Jahr
	GenesisYear = 2025

	// Maximale Halvings bevor Belohnung = 0
	MaxHalvings = 10
)

// HalvingState verfolgt den aktuellen Halving-Status
type HalvingState struct {
	// Aktuelle Epoche (0 = erste Epoche, 1 = nach erstem Halving, etc.)
	CurrentEpoch int

	// Aktuelle Block-Belohnung in micro-NVX
	CurrentReward int64

	// Nächster Block bei dem Halving stattfindet
	NextHalvingBlock int64

	// Gesamte geminte NVX bisher
	TotalMined int64

	// Maximum der mintbaren NVX (aus Staking Rewards Pool)
	MaxMineable int64
}

// MiningKeeper verwaltet das Mining und Halving System
type MiningKeeper struct {
	state      HalvingState
	burnKeeper *BurnKeeper
}

// NewMiningKeeper erstellt einen neuen Mining Keeper
func NewMiningKeeper(burnKeeper *BurnKeeper) *MiningKeeper {
	return &MiningKeeper{
		burnKeeper: burnKeeper,
		state: HalvingState{
			CurrentEpoch:     0,
			CurrentReward:    InitialBlockReward,
			NextHalvingBlock: BlocksPerHalving,
			TotalMined:       0,
			MaxMineable:      100_000_000_000_000, // 100M NVX Staking Pool
		},
	}
}

// ─────────────────────────────────────────────
//  Block Belohnung berechnen
// ─────────────────────────────────────────────

// GetBlockReward gibt die aktuelle Block-Belohnung zurück
func (mk *MiningKeeper) GetBlockReward(height int64) int64 {
	// Halving prüfen
	mk.checkHalving(height)

	// Kein Mining mehr wenn Pool leer
	if mk.state.TotalMined >= mk.state.MaxMineable {
		return 0
	}

	// Letzte Belohnung anpassen falls Pool fast leer
	remaining := mk.state.MaxMineable - mk.state.TotalMined
	if remaining < mk.state.CurrentReward {
		return remaining
	}

	return mk.state.CurrentReward
}

// ProcessBlockReward verarbeitet die Belohnung für einen neuen Block
func (mk *MiningKeeper) ProcessBlockReward(
	height int64,
	validatorAddress string,
) (int64, error) {
	reward := mk.GetBlockReward(height)

	if reward == 0 {
		return 0, nil // Mining abgeschlossen
	}

	mk.state.TotalMined += reward

	fmt.Printf("[Mining] Block %d: +%d unvx → %s | Epoch: %d | Next Halving: Block %d\n",
		height,
		reward,
		validatorAddress[:16]+"...",
		mk.state.CurrentEpoch,
		mk.state.NextHalvingBlock,
	)

	return reward, nil
}

// ─────────────────────────────────────────────
//  Halving Logic
// ─────────────────────────────────────────────

// checkHalving prüft ob ein Halving stattfinden soll
func (mk *MiningKeeper) checkHalving(height int64) {
	if height < mk.state.NextHalvingBlock {
		return // Noch kein Halving
	}

	if mk.state.CurrentEpoch >= MaxHalvings {
		return // Maximale Halvings erreicht
	}

	// HALVING EVENT
	oldReward := mk.state.CurrentReward
	mk.state.CurrentReward = mk.state.CurrentReward / 2
	mk.state.CurrentEpoch++
	mk.state.NextHalvingBlock += BlocksPerHalving

	halvingYear := GenesisYear + (mk.state.CurrentEpoch * 5)

	fmt.Println("\n🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥")
	fmt.Printf("  NUVEX HALVING #%d — Jahr %d\n", mk.state.CurrentEpoch, halvingYear)
	fmt.Printf("  Alte Belohnung:  %d unvx (%.2f NVX)\n", oldReward, float64(oldReward)/1_000_000)
	fmt.Printf("  Neue Belohnung:  %d unvx (%.2f NVX)\n", mk.state.CurrentReward, float64(mk.state.CurrentReward)/1_000_000)
	fmt.Printf("  Nächstes Halving: Block %d (~Jahr %d)\n", mk.state.NextHalvingBlock, halvingYear+5)
	fmt.Println("🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥\n")
}

// ─────────────────────────────────────────────
//  Mining Statistiken
// ─────────────────────────────────────────────

type MiningStats struct {
	CurrentEpoch      int       `json:"current_epoch"`
	CurrentReward     int64     `json:"current_reward_unvx"`
	CurrentRewardNVX  float64   `json:"current_reward_nvx"`
	NextHalvingBlock  int64     `json:"next_halving_block"`
	NextHalvingYear   int       `json:"next_halving_year"`
	TotalMined        int64     `json:"total_mined_unvx"`
	TotalMinedNVX     float64   `json:"total_mined_nvx"`
	MaxMineable       int64     `json:"max_mineable_unvx"`
	PercentMined      float64   `json:"percent_mined"`
	EstimatedLastCoin string    `json:"estimated_last_coin"`
}

func (mk *MiningKeeper) GetStats(currentHeight int64) MiningStats {
	mk.checkHalving(currentHeight)

	percentMined := float64(mk.state.TotalMined) / float64(mk.state.MaxMineable) * 100
	nextHalvingYear := GenesisYear + ((mk.state.CurrentEpoch + 1) * 5)

	return MiningStats{
		CurrentEpoch:     mk.state.CurrentEpoch,
		CurrentReward:    mk.state.CurrentReward,
		CurrentRewardNVX: float64(mk.state.CurrentReward) / 1_000_000,
		NextHalvingBlock: mk.state.NextHalvingBlock,
		NextHalvingYear:  nextHalvingYear,
		TotalMined:       mk.state.TotalMined,
		TotalMinedNVX:    float64(mk.state.TotalMined) / 1_000_000,
		MaxMineable:      mk.state.MaxMineable,
		PercentMined:     percentMined,
		EstimatedLastCoin: fmt.Sprintf("~Jahr %d", GenesisYear+(MaxHalvings*5)),
	}
}

// ─────────────────────────────────────────────
//  Halving Schedule (für Whitepaper & Website)
// ─────────────────────────────────────────────

type HalvingEvent struct {
	Epoch       int
	Year        int
	Block       int64
	Reward      float64
	Date        string
}

func GetHalvingSchedule() []HalvingEvent {
	events := make([]HalvingEvent, MaxHalvings+1)
	reward := float64(InitialBlockReward) / 1_000_000

	for i := 0; i <= MaxHalvings; i++ {
		year := GenesisYear + (i * 5)
		events[i] = HalvingEvent{
			Epoch:  i,
			Year:   year,
			Block:  int64(i) * BlocksPerHalving,
			Reward: reward,
			Date:   fmt.Sprintf("~%s", time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format("January 2006")),
		}
		reward = reward / 2
	}
	return events
}

func PrintHalvingSchedule() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║           NUVEX HALVING SCHEDULE                         ║")
	fmt.Println("╠══════════════════════════════════════════════════════════╣")
	fmt.Printf("║  %-8s %-10s %-20s %-15s║\n", "Epoch", "Jahr", "Block", "Belohnung/Block")
	fmt.Println("╠══════════════════════════════════════════════════════════╣")

	for _, e := range GetHalvingSchedule() {
		fmt.Printf("║  %-8d %-10d %-20d %-10.4f NVX  ║\n",
			e.Epoch, e.Year, e.Block, e.Reward)
	}
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
}
