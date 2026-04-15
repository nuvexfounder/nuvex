package keeper

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  Nuvex Governance System
//
//  NVX Holder können Proposals erstellen und
//  darüber abstimmen. Voting Power = NVX Balance.
//
//  Proposal Types:
//  - Parameter Change (z.B. APY ändern)
//  - Network Upgrade
//  - Community Fund
//  - Text Proposal
//
//  Voting Period: 7 Tage
//  Quorum: 33% der circulating supply
//  Pass Threshold: 50%+ Yes votes
// ─────────────────────────────────────────────

const (
	VotingPeriod   = 7 * 24 * time.Hour
	Quorum         = 0.33
	PassThreshold  = 0.50
	MinDeposit     = 1000_000_000 // 1000 NVX
)

type ProposalStatus string

const (
	StatusVoting  ProposalStatus = "voting"
	StatusPassed  ProposalStatus = "passed"
	StatusRejected ProposalStatus = "rejected"
	StatusExpired ProposalStatus = "expired"
)

type ProposalType string

const (
	TypeParameterChange ProposalType = "parameter_change"
	TypeNetworkUpgrade  ProposalType = "network_upgrade"
	TypeCommunityFund   ProposalType = "community_fund"
	TypeText            ProposalType = "text"
)

type VoteOption string

const (
	VoteYes     VoteOption = "yes"
	VoteNo      VoteOption = "no"
	VoteAbstain VoteOption = "abstain"
)

// Vote repräsentiert eine einzelne Stimme
type Vote struct {
	Voter     string     `json:"voter"`
	Option    VoteOption `json:"option"`
	Power     uint64     `json:"power"`
	Timestamp time.Time  `json:"timestamp"`
}

// Proposal repräsentiert einen Governance Antrag
type Proposal struct {
	ID          uint64         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Type        ProposalType   `json:"type"`
	Proposer    string         `json:"proposer"`
	Deposit     uint64         `json:"deposit"`
	Status      ProposalStatus `json:"status"`
	StartTime   time.Time      `json:"start_time"`
	EndTime     time.Time      `json:"end_time"`
	Votes       []Vote         `json:"votes"`
	YesPower    uint64         `json:"yes_power"`
	NoPower     uint64         `json:"no_power"`
	AbstainPower uint64        `json:"abstain_power"`
	TotalPower  uint64         `json:"total_power"`
	Passed      bool           `json:"passed"`
}

// GovernanceStats gibt globale Statistiken
type GovernanceStats struct {
	TotalProposals  int    `json:"total_proposals"`
	ActiveProposals int    `json:"active_proposals"`
	PassedProposals int    `json:"passed_proposals"`
	TotalVotes      int    `json:"total_votes"`
	MinDeposit      uint64 `json:"min_deposit"`
	VotingPeriodDays int   `json:"voting_period_days"`
	Quorum          float64 `json:"quorum"`
	PassThreshold   float64 `json:"pass_threshold"`
}

// GovernanceKeeper verwaltet alle Proposals
type GovernanceKeeper struct {
	mu          sync.RWMutex
	proposals   map[uint64]*Proposal
	nextID      uint64
	state       *StateKeeper
}

// NewGovernanceKeeper erstellt einen neuen Governance Keeper
func NewGovernanceKeeper(state *StateKeeper) *GovernanceKeeper {
	gk := &GovernanceKeeper{
		proposals: make(map[uint64]*Proposal),
		nextID:    1,
		state:     state,
	}

	fmt.Println("[Governance] ✅ Nuvex Governance System gestartet")
	fmt.Printf("[Governance] ✅ Voting Period: 7 Tage\n")
	fmt.Printf("[Governance] ✅ Quorum: %.0f%%\n", Quorum*100)
	fmt.Printf("[Governance] ✅ Pass Threshold: %.0f%%\n", PassThreshold*100)
	fmt.Printf("[Governance] ✅ Min Deposit: %d NVX\n", MinDeposit/1_000_000)

	go gk.processLoop()

	return gk
}

// ─────────────────────────────────────────────
//  Proposal erstellen
// ─────────────────────────────────────────────

func (gk *GovernanceKeeper) CreateProposal(
	proposer string,
	title string,
	description string,
	proposalType ProposalType,
	deposit uint64,
) (*Proposal, error) {
	gk.mu.Lock()
	defer gk.mu.Unlock()

	if deposit < MinDeposit {
		return nil, fmt.Errorf("Minimum Deposit: %d NVX", MinDeposit/1_000_000)
	}

	balance := uint64(gk.state.GetBalance(proposer))
	if balance < deposit {
		return nil, fmt.Errorf("Nicht genug NVX: hat %d, braucht %d",
			balance/1_000_000, deposit/1_000_000)
	}

	gk.state.mu.Lock()
	if acc, exists := gk.state.accounts[proposer]; exists {
		acc.Balance -= int64(deposit)
	}
	gk.state.mu.Unlock()

	proposal := &Proposal{
		ID:          gk.nextID,
		Title:       title,
		Description: description,
		Type:        proposalType,
		Proposer:    proposer,
		Deposit:     deposit,
		Status:      StatusVoting,
		StartTime:   time.Now().UTC(),
		EndTime:     time.Now().UTC().Add(VotingPeriod),
		Votes:       []Vote{},
	}

	gk.proposals[gk.nextID] = proposal
	gk.nextID++

	fmt.Printf("[Governance] ✅ Proposal #%d erstellt: %s | Proposer: %s\n",
		proposal.ID, title, proposer[:10]+"...")

	return proposal, nil
}

// ─────────────────────────────────────────────
//  Abstimmen
// ─────────────────────────────────────────────

func (gk *GovernanceKeeper) Vote(
	proposalID uint64,
	voter string,
	option VoteOption,
) (*Vote, error) {
	gk.mu.Lock()
	defer gk.mu.Unlock()

	proposal, exists := gk.proposals[proposalID]
	if !exists {
		return nil, fmt.Errorf("Proposal nicht gefunden: %d", proposalID)
	}

	if proposal.Status != StatusVoting {
		return nil, fmt.Errorf("Proposal ist nicht mehr offen: %s", proposal.Status)
	}

	if time.Now().After(proposal.EndTime) {
		return nil, fmt.Errorf("Voting Period abgelaufen")
	}

	// Voting Power = NVX Balance
	power := uint64(gk.state.GetBalance(voter))
	if power == 0 {
		return nil, fmt.Errorf("Keine Voting Power — kein NVX Balance")
	}

	// Existierende Stimme entfernen falls vorhanden
	for i, v := range proposal.Votes {
		if v.Voter == voter {
			switch v.Option {
			case VoteYes:
				proposal.YesPower -= v.Power
			case VoteNo:
				proposal.NoPower -= v.Power
			case VoteAbstain:
				proposal.AbstainPower -= v.Power
			}
			proposal.TotalPower -= v.Power
			proposal.Votes = append(proposal.Votes[:i], proposal.Votes[i+1:]...)
			break
		}
	}

	vote := Vote{
		Voter:     voter,
		Option:    option,
		Power:     power,
		Timestamp: time.Now().UTC(),
	}

	proposal.Votes = append(proposal.Votes, vote)
	proposal.TotalPower += power

	switch option {
	case VoteYes:
		proposal.YesPower += power
	case VoteNo:
		proposal.NoPower += power
	case VoteAbstain:
		proposal.AbstainPower += power
	}

	fmt.Printf("[Governance] Vote: Proposal #%d | %s | Power: %d | Voter: %s\n",
		proposalID, option, power/1_000_000, voter[:10]+"...")

	return &vote, nil
}

// ─────────────────────────────────────────────
//  Proposal verarbeiten
// ─────────────────────────────────────────────

func (gk *GovernanceKeeper) processLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		gk.mu.Lock()
		for _, p := range gk.proposals {
			if p.Status == StatusVoting && time.Now().After(p.EndTime) {
				gk.finalizeProposal(p)
			}
		}
		gk.mu.Unlock()
	}
}

func (gk *GovernanceKeeper) finalizeProposal(p *Proposal) {
	totalSupply := uint64(500_000_000_000_000)
	quorumReached := float64(p.TotalPower)/float64(totalSupply) >= Quorum

	if !quorumReached {
		p.Status = StatusExpired
		p.Passed = false
		fmt.Printf("[Governance] Proposal #%d EXPIRED — Quorum nicht erreicht\n", p.ID)
		return
	}

	yesRatio := float64(p.YesPower) / float64(p.TotalPower)
	if yesRatio > PassThreshold {
		p.Status = StatusPassed
		p.Passed = true
		gk.state.mu.Lock()
		if acc, exists := gk.state.accounts[p.Proposer]; exists {
			acc.Balance += int64(p.Deposit)
		}
		gk.state.mu.Unlock()
		fmt.Printf("[Governance] Proposal #%d PASSED — %.1f%% Yes\n", p.ID, yesRatio*100)
	} else {
		p.Status = StatusRejected
		p.Passed = false
		fmt.Printf("[Governance] Proposal #%d REJECTED — %.1f%% Yes\n", p.ID, yesRatio*100)
	}
}

// ─────────────────────────────────────────────
//  Abfragen
// ─────────────────────────────────────────────

func (gk *GovernanceKeeper) GetProposal(id uint64) (*Proposal, error) {
	gk.mu.RLock()
	defer gk.mu.RUnlock()
	p, exists := gk.proposals[id]
	if !exists {
		return nil, fmt.Errorf("Proposal nicht gefunden")
	}
	return p, nil
}

func (gk *GovernanceKeeper) GetAllProposals() []*Proposal {
	gk.mu.RLock()
	defer gk.mu.RUnlock()
	proposals := make([]*Proposal, 0, len(gk.proposals))
	for _, p := range gk.proposals {
		proposals = append(proposals, p)
	}
	return proposals
}

func (gk *GovernanceKeeper) GetStats() GovernanceStats {
	gk.mu.RLock()
	defer gk.mu.RUnlock()
	stats := GovernanceStats{
		MinDeposit:       MinDeposit,
		VotingPeriodDays: 7,
		Quorum:           Quorum,
		PassThreshold:    PassThreshold,
	}
	for _, p := range gk.proposals {
		stats.TotalProposals++
		stats.TotalVotes += len(p.Votes)
		if p.Status == StatusVoting {
			stats.ActiveProposals++
		}
		if p.Status == StatusPassed {
			stats.PassedProposals++
		}
	}
	return stats
}

func (p *Proposal) ToJSON() string {
	data, _ := json.MarshalIndent(p, "", "  ")
	return string(data)
}
