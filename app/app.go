package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/nuvex-foundation/nuvex/x/nvx/keeper"
	"github.com/nuvex-foundation/nuvex/x/nvx/types"
)

type NuvexApp struct {
	ChainID    string
	Version    string
	Burns      *keeper.BurnKeeper
	State      *keeper.StateKeeper
	Blockchain *keeper.Blockchain
	Mempool    *keeper.Mempool
	P2P        *keeper.P2PNode
	BFT        *keeper.BFTConsensus
	Engine     *keeper.ConsensusEngine
	EVM        *keeper.EVMKeeper
	DEX        *keeper.DEXKeeper
	Staking    *keeper.StakingKeeper
	Governance *keeper.GovernanceKeeper
	Height     int64
}

func NewNuvexApp() *NuvexApp {
	PrintGenesis()

	bc      := keeper.NewBlockchain("/root/nuvex/chain.db")
	state   := keeper.NewStateKeeper("/root/nuvex/state.json")
	mempool := keeper.NewMempool()

	if state.GetBalance("nuvex19d85718c8da8f4213e1a2a41fe894ba928b9c9") == 0 {
		state.InitGenesis([]keeper.GenesisEntry{
			{Address: "nuvex1cac179655ab9fd3543fe537675a3c3eb447691", Amount: 275_000_000_000_000, PublicKey: ""},
			{Address: "nuvex1f81358e9a3840c5ca4647f2bc90b0e8b7b7a70", Amount: 100_000_000_000_000, PublicKey: ""},
			{Address: "nuvex1c22e41daadec8a5856631fbdd11a40b69b0248", Amount: 100_000_000_000_000, PublicKey: ""},
			{Address: "nuvex19d85718c8da8f4213e1a2a41fe894ba928b9c9", Amount:  25_000_000_000_000, PublicKey: ""},
			{Address: "nuvex1af7c2f39fd1751be635ff6141a4c4aea8cda91", Amount:  25_000_000_000_000, PublicKey: ""},
		})
	}

	validators := []keeper.Validator{{
		Address:     "nuvex19d85718c8da8f4213e1a2a41fe894ba928b9c9",
		VotingPower: 25_000_000,
		IsActive:    true,
		IsGreen:     true,
	}}

	h      := sha256.Sum256([]byte("nuvex19d85718c8da8f4213e1a2a41fe894ba928b9c9"))
	nodeID := hex.EncodeToString(h[:])

	burns  := keeper.NewBurnKeeper()
	engine := keeper.NewConsensusEngine(validators)
	p2p    := keeper.NewP2PNode(nodeID, bc, mempool)
	bft    := keeper.NewBFTConsensus(validators, bc, mempool, p2p,
		"nuvex19d85718c8da8f4213e1a2a41fe894ba928b9c9")
	evm, _ := keeper.NewEVMKeeper()
	dex     := keeper.NewDEXKeeper()
	staking    := keeper.NewStakingKeeper(state)
	governance := keeper.NewGovernanceKeeper(state)

	if err := bc.ValidateChain(); err != nil {
		fmt.Printf("[Nuvex] Chain Fehler: %v\n", err)
	}

	fmt.Printf("[Nuvex] Chain Height:  %d\n", bc.Height())
	fmt.Printf("[Nuvex] DEX bereit\n")
	fmt.Printf("[Nuvex] EVM Chain ID: 1317\n\n")

	return &NuvexApp{
		ChainID:    types.ChainID,
		Version:    "1.0.0",
		Burns:      burns,
		State:      state,
		Blockchain: bc,
		Mempool:    mempool,
		P2P:        p2p,
		BFT:        bft,
		Engine:     engine,
		EVM:        evm,
		DEX:        dex,
		Staking:    staking,
		Governance: governance,
		Height:     bc.Height(),
	}
}

func (a *NuvexApp) Run(blocks int) {
	for i := 1; i <= blocks; i++ {
		a.Height++
		a.BFT.StartRound(a.Height, 0)
		a.Engine.ProcessBlock(a.Height, 0, 0)
	}
}
