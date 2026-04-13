package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

type Validator struct {
	Address       string
	VotingPower   int64
	IsActive      bool
	IsGreen       bool
	CommissionBps int64
	TotalEarned   int64
}

type ConsensusEngine struct {
	validators    []Validator
	quorum        int
	burnKeeper    *BurnKeeper
	miningKeeper  *MiningKeeper
}

func NewConsensusEngine(validators []Validator) *ConsensusEngine {
	burnKeeper := NewBurnKeeper()
	return &ConsensusEngine{
		validators:   validators,
		quorum:       (2*len(validators)/3) + 1,
		burnKeeper:   burnKeeper,
		miningKeeper: NewMiningKeeper(burnKeeper),
	}
}

func (ce *ConsensusEngine) SelectLeader(prevHash []byte, height int64) *Validator {
	if len(ce.validators) == 0 {
		return nil
	}
	h := sha256.New()
	h.Write(prevHash)
	hb := make([]byte, 8)
	binary.BigEndian.PutUint64(hb, uint64(height))
	h.Write(hb)
	seed := h.Sum(nil)

	var total int64
	for _, v := range ce.validators {
		if v.IsActive {
			total += v.VotingPower
		}
	}
	if total == 0 {
		return &ce.validators[0]
	}

	slot := int64(binary.BigEndian.Uint64(seed[:8]) % uint64(total))
	var cum int64
	for i := range ce.validators {
		if !ce.validators[i].IsActive {
			continue
		}
		cum += ce.validators[i].VotingPower
		if slot < cum {
			return &ce.validators[i]
		}
	}
	return &ce.validators[0]
}

func (ce *ConsensusEngine) ProcessBlock(height int64, txCount int, totalFees int64) {
	leader := ce.SelectLeader([]byte("prevhash"), height)
	if leader == nil {
		return
	}

	// Fee burn
	ce.burnKeeper.BurnFromFee(totalFees, fmt.Sprintf("block-%d", height), height)

	// Mining reward
	reward, _ := ce.miningKeeper.ProcessBlockReward(height, leader.Address)

	// Green validator bonus: +0.5%
	if leader.IsGreen && reward > 0 {
		bonus := reward * 5 / 1000
		reward += bonus
	}

	for i := range ce.validators {
		if ce.validators[i].Address == leader.Address {
			ce.validators[i].TotalEarned += reward
		}
	}

	fmt.Printf("[Block %d] Leader: %s | Txs: %d | Reward: %.4f NVX\n",
		height,
		leader.Address[:16]+"...",
		txCount,
		float64(reward)/1_000_000,
	)
}

func (ce *ConsensusEngine) GetMiningStats(height int64) MiningStats {
	return ce.miningKeeper.GetStats(height)
}

func (ce *ConsensusEngine) PrintHalvingSchedule() {
	PrintHalvingSchedule()
}
