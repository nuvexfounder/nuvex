package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type ConsensusPhase int

const (
	PhaseNewRound  ConsensusPhase = iota
	PhasePropose
	PhasePrepare
	PhaseCommit
	PhaseFinalized
)

func (p ConsensusPhase) String() string {
	switch p {
	case PhaseNewRound:  return "NEW_ROUND"
	case PhasePropose:   return "PROPOSE"
	case PhasePrepare:   return "PREPARE"
	case PhaseCommit:    return "COMMIT"
	case PhaseFinalized: return "FINALIZED"
	default:             return "UNKNOWN"
	}
}

type ConsensusVote struct {
	ValidatorAddr string    `json:"validator"`
	BlockHash     string    `json:"block_hash"`
	Height        int64     `json:"height"`
	Round         int       `json:"round"`
	Phase         string    `json:"phase"`
	Timestamp     time.Time `json:"timestamp"`
}

type ConsensusRound struct {
	Height        int64
	Round         int
	Phase         ConsensusPhase
	LeaderAddr    string
	ProposedBlock *Block
	PrepareVotes  map[string]*ConsensusVote
	CommitVotes   map[string]*ConsensusVote
	StartTime     time.Time
	FinalizedAt   *time.Time
}

type BFTStats struct {
	Phase          string `json:"phase"`
	Height         int64  `json:"height"`
	Round          int    `json:"round"`
	Leader         string `json:"leader"`
	PrepareVotes   int    `json:"prepare_votes"`
	CommitVotes    int    `json:"commit_votes"`
	Quorum         int    `json:"quorum"`
	TotalBlocks    int64  `json:"total_blocks_finalized"`
	FailedRounds   int64  `json:"failed_rounds"`
	ValidatorCount int    `json:"validator_count"`
}

type BFTConsensus struct {
	mu           sync.RWMutex
	validators   []Validator
	quorum       int
	currentRound *ConsensusRound
	blockchain   *Blockchain
	mempool      *Mempool
	p2p          *P2PNode
	nodeAddr     string
	totalBlocks  int64
	failedRounds int64
}

func NewBFTConsensus(validators []Validator, blockchain *Blockchain, mempool *Mempool, p2p *P2PNode, nodeAddr string) *BFTConsensus {
	n := len(validators)
	quorum := (2*n/3) + 1
	if quorum < 1 {
		quorum = 1
	}
	fmt.Printf("[BFT] Validatoren: %d | Quorum: %d\n", n, quorum)
	return &BFTConsensus{
		validators: validators,
		quorum:     quorum,
		blockchain: blockchain,
		mempool:    mempool,
		p2p:        p2p,
		nodeAddr:   nodeAddr,
	}
}

func (bft *BFTConsensus) StartRound(height int64, round int) {
	bft.mu.Lock()
	leader := bft.selectLeader(height, round)
	bft.currentRound = &ConsensusRound{
		Height:       height,
		Round:        round,
		Phase:        PhaseNewRound,
		LeaderAddr:   leader.Address,
		PrepareVotes: make(map[string]*ConsensusVote),
		CommitVotes:  make(map[string]*ConsensusVote),
		StartTime:    time.Now(),
	}
	bft.mu.Unlock()
	if leader.Address == bft.nodeAddr {
		bft.proposeBlock()
	}
}

func (bft *BFTConsensus) proposeBlock() {
	bft.mu.Lock()
	if bft.currentRound == nil {
		bft.mu.Unlock()
		return
	}
	pendingTxs := bft.mempool.SelectForBlock(1000)
	txHashes := make([]string, len(pendingTxs))
	totalFees := int64(0)
	for i, tx := range pendingTxs {
		txHashes[i] = tx.Hash
		totalFees += tx.Fee
	}
	latest := bft.blockchain.GetLatestBlock()
	prevHash := "GENESIS"
	if latest != nil {
		prevHash = latest.Hash
	}
	block := &Block{
		Height:       bft.currentRound.Height,
		PrevHash:     prevHash,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Validator:    bft.nodeAddr,
		Transactions: txHashes,
		TxCount:      len(txHashes),
		TotalFees:    totalFees,
		BurnedAmount: totalFees * 10 / 10_000,
		MiningReward: 50_000_000,
		StateRoot:    bft.calcStateRoot(txHashes),
	}
	block.Hash = bft.hashBlock(block)
	bft.currentRound.ProposedBlock = block
	bft.currentRound.Phase = PhasePropose
	bft.mu.Unlock()
	fmt.Printf("[BFT] Block #%d vorgeschlagen\n", block.Height)
	bft.ReceivePrepareVote(&ConsensusVote{
		ValidatorAddr: bft.nodeAddr,
		BlockHash:     block.Hash,
		Height:        block.Height,
		Round:         bft.currentRound.Round,
		Phase:         "PREPARE",
		Timestamp:     time.Now().UTC(),
	})
}

func (bft *BFTConsensus) ReceivePrepareVote(vote *ConsensusVote) {
	bft.mu.Lock()
	defer bft.mu.Unlock()
	if bft.currentRound == nil || vote.Height != bft.currentRound.Height {
		return
	}
	bft.currentRound.PrepareVotes[vote.ValidatorAddr] = vote
	bft.currentRound.Phase = PhasePrepare
	if len(bft.currentRound.PrepareVotes) >= bft.quorum {
		bft.currentRound.Phase = PhaseCommit
		go bft.submitCommitVote()
	}
}

func (bft *BFTConsensus) submitCommitVote() {
	bft.mu.RLock()
	if bft.currentRound == nil || bft.currentRound.ProposedBlock == nil {
		bft.mu.RUnlock()
		return
	}
	vote := &ConsensusVote{
		ValidatorAddr: bft.nodeAddr,
		BlockHash:     bft.currentRound.ProposedBlock.Hash,
		Height:        bft.currentRound.Height,
		Round:         bft.currentRound.Round,
		Phase:         "COMMIT",
		Timestamp:     time.Now().UTC(),
	}
	bft.mu.RUnlock()
	bft.ReceiveCommitVote(vote)
}

func (bft *BFTConsensus) ReceiveCommitVote(vote *ConsensusVote) {
	bft.mu.Lock()
	defer bft.mu.Unlock()
	if bft.currentRound == nil || vote.Height != bft.currentRound.Height {
		return
	}
	bft.currentRound.CommitVotes[vote.ValidatorAddr] = vote
	if len(bft.currentRound.CommitVotes) >= bft.quorum && bft.currentRound.Phase == PhaseCommit {
		bft.finalizeBlock()
	}
}

func (bft *BFTConsensus) finalizeBlock() {
	block := bft.currentRound.ProposedBlock
	if block == nil {
		return
	}
	now := time.Now()
	bft.currentRound.FinalizedAt = &now
	bft.currentRound.Phase = PhaseFinalized
	bft.totalBlocks++
	duration := now.Sub(bft.currentRound.StartTime)
	fmt.Printf("[BFT] Block #%d FINALISIERT | %dms\n", block.Height, duration.Milliseconds())
	bft.blockchain.AddBlock(block.Validator, block.Transactions, block.TotalFees, block.BurnedAmount, block.MiningReward)
	if bft.p2p != nil {
		bft.p2p.BroadcastBlock(block)
	}
}

func (bft *BFTConsensus) Run() {
	fmt.Println("[BFT] Loop gestartet...")
	for {
		height := bft.blockchain.Height() + 1
		bft.StartRound(height, 0)
		timeout := time.After(2 * time.Second)
		ticker := time.NewTicker(50 * time.Millisecond)
		done := false
		for !done {
			select {
			case <-ticker.C:
				bft.mu.RLock()
				if bft.currentRound != nil && bft.currentRound.Phase == PhaseFinalized {
					done = true
				}
				bft.mu.RUnlock()
			case <-timeout:
				bft.mu.Lock()
				bft.failedRounds++
				bft.mu.Unlock()
				done = true
			}
		}
		ticker.Stop()
		time.Sleep(400 * time.Millisecond)
	}
}

func (bft *BFTConsensus) selectLeader(height int64, round int) *Validator {
	data := fmt.Sprintf("%d:%d:nuvex-1", height, round)
	hash := sha256.Sum256([]byte(data))
	idx := int(hash[0]) % len(bft.validators)
	return &bft.validators[idx]
}

func (bft *BFTConsensus) hashBlock(b *Block) string {
	data := fmt.Sprintf("%d%s%s%s%d%d%s", b.Height, b.PrevHash, b.Timestamp, b.Validator, b.TxCount, b.MiningReward, b.StateRoot)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (bft *BFTConsensus) calcStateRoot(txHashes []string) string {
	h := sha256.New()
	for _, tx := range txHashes {
		h.Write([]byte(tx))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (bft *BFTConsensus) AddValidator(v Validator) {
	bft.mu.Lock()
	defer bft.mu.Unlock()
	bft.validators = append(bft.validators, v)
	bft.quorum = (2*len(bft.validators)/3) + 1
}

func (bft *BFTConsensus) Stats() BFTStats {
	bft.mu.RLock()
	defer bft.mu.RUnlock()
	stats := BFTStats{
		Quorum:         bft.quorum,
		TotalBlocks:    bft.totalBlocks,
		FailedRounds:   bft.failedRounds,
		ValidatorCount: len(bft.validators),
		Phase:          "IDLE",
	}
	if bft.currentRound != nil {
		stats.Phase        = bft.currentRound.Phase.String()
		stats.Height       = bft.currentRound.Height
		stats.Round        = bft.currentRound.Round
		stats.Leader       = bft.currentRound.LeaderAddr
		stats.PrepareVotes = len(bft.currentRound.PrepareVotes)
		stats.CommitVotes  = len(bft.currentRound.CommitVotes)
	}
	return stats
}
