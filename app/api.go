package app

import (
	"encoding/json"
	"github.com/nuvex-foundation/nuvex/x/nvx/keeper"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type APIServer struct {
	app  *NuvexApp
	port string
}

func NewAPIServer(app *NuvexApp, port string) *APIServer {
	return &APIServer{app: app, port: port}
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
}

func (s *APIServer) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/block/latest", s.handleLatestBlock)
	mux.HandleFunc("/block/", s.handleBlock)
	mux.HandleFunc("/blocks", s.handleBlocks)
	mux.HandleFunc("/supply", s.handleSupply)
	mux.HandleFunc("/validators", s.handleValidators)
	mux.HandleFunc("/balance/", s.handleBalance)
	mux.HandleFunc("/txs", s.handleTxs)
	mux.HandleFunc("/accounts", s.handleAccounts)
	mux.HandleFunc("/tx/send", s.handleSend)
	mux.HandleFunc("/mining", s.handleMining)
	mux.HandleFunc("/mempool", s.handleMempool)
	mux.HandleFunc("/mempool/submit", s.handleMempoolSubmit)
	mux.HandleFunc("/p2p", s.handleP2P)
	mux.HandleFunc("/p2p/connect", s.handleP2PConnect)
	mux.HandleFunc("/consensus", s.handleConsensus)
	mux.HandleFunc("/evm/contracts", s.handleEVMContracts)
	mux.HandleFunc("/evm/deploy", s.handleEVMDeploy)
	mux.HandleFunc("/evm/call", s.handleEVMCall)
	mux.HandleFunc("/evm/stats", s.handleEVMStats)
	mux.HandleFunc("/dex/stats", s.handleDEXStats)
	mux.HandleFunc("/dex/pools", s.handleDEXPools)
	mux.HandleFunc("/dex/create", s.handleDEXCreatePool)
	mux.HandleFunc("/dex/swap", s.handleDEXSwap)
	mux.HandleFunc("/dex/quote", s.handleDEXQuote)
	mux.HandleFunc("/dex/liquidity/add", s.handleDEXAddLiquidity)
	mux.HandleFunc("/dex/history", s.handleDEXHistory)
	mux.HandleFunc("/staking/stats", s.handleStakingStats)
	mux.HandleFunc("/staking/stake", s.handleStake)
	mux.HandleFunc("/staking/unstake", s.handleUnstake)
	mux.HandleFunc("/staking/positions", s.handleStakingPositions)
	mux.HandleFunc("/staking/pending", s.handleStakingPending)
	mux.HandleFunc("/governance/stats", s.handleGovStats)
	mux.HandleFunc("/governance/proposals", s.handleGovProposals)
	mux.HandleFunc("/governance/proposal", s.handleGovProposal)
	mux.HandleFunc("/governance/create", s.handleGovCreate)
	mux.HandleFunc("/governance/vote", s.handleGovVote)
	mux.HandleFunc("/evm/stats/v2", s.handleEVMStatsV2)
	mux.HandleFunc("/evm/estimate", s.handleEVMEstimateGas)
	mux.HandleFunc("/evm/contract/info", s.handleEVMContractInfo)
	mux.HandleFunc("/evm/nvx20/abi", s.handleNVX20ABI)
	mux.HandleFunc("/storage/stats", s.handleLevelDBStats)
	mux.HandleFunc("/tx/address", s.handleTxByAddress)
	mux.HandleFunc("/tx/hash", s.handleTxByHash)
	
	fmt.Printf("[Nuvex API] Running on port %s\n", s.port)
	http.ListenAndServe(":"+s.port, mux)
}

func (s *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	cors(w)
	burned, circ := s.app.Burns.Stats()
	latest := s.app.Blockchain.GetLatestBlock()
	latestHash := ""
	if latest != nil {
		latestHash = latest.Hash
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"chain_id":     s.app.ChainID,
		"version":      s.app.Version,
		"height":       s.app.Blockchain.Height(),
		"total_blocks": s.app.Blockchain.TotalBlocks(),
		"latest_hash":  latestHash,
		"circulating":  circ,
		"burned":       burned,
		"mempool_size": s.app.Mempool.Size(),
		"timestamp":    time.Now().UTC(),
		"status":       "running",
	})
}

func (s *APIServer) handleLatestBlock(w http.ResponseWriter, r *http.Request) {
	cors(w)
	block := s.app.Blockchain.GetLatestBlock()
	if block == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Kein Block"})
		return
	}
	json.NewEncoder(w).Encode(block)
}

func (s *APIServer) handleBlock(w http.ResponseWriter, r *http.Request) {
	cors(w)
	heightStr := strings.TrimPrefix(r.URL.Path, "/block/")
	if heightStr == "latest" {
		block := s.app.Blockchain.GetLatestBlock()
		json.NewEncoder(w).Encode(block)
		return
	}
	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Ungültige Höhe"})
		return
	}
	block := s.app.Blockchain.GetBlock(height)
	if block == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Block nicht gefunden"})
		return
	}
	json.NewEncoder(w).Encode(block)
}

func (s *APIServer) handleBlocks(w http.ResponseWriter, r *http.Request) {
	cors(w)
	blocks := s.app.Blockchain.GetRecentBlocks(20)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"blocks": blocks,
		"total":  s.app.Blockchain.TotalBlocks(),
	})
}

func (s *APIServer) handleSupply(w http.ResponseWriter, r *http.Request) {
	cors(w)
	burned, circ := s.app.Burns.Stats()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"max_supply":  500_000_000_000_000,
		"circulating": circ,
		"burned":      burned,
		"denom":       "unvx",
		"display":     "nvx",
	})
}

func (s *APIServer) handleValidators(w http.ResponseWriter, r *http.Request) {
	cors(w)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total": 1,
		"validators": []map[string]interface{}{
			{
				"address":      "nuvex19d85718c8da8f4213e1a2a41fe894ba928b9c9",
				"voting_power": 25_000_000,
				"is_green":     true,
				"commission":   "5%",
				"status":       "active",
			},
		},
	})
}

func (s *APIServer) handleBalance(w http.ResponseWriter, r *http.Request) {
	cors(w)
	address := strings.TrimPrefix(r.URL.Path, "/balance/")
	if address == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "Adresse fehlt"})
		return
	}
	balance := s.app.State.GetBalance(address)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"address":      address,
		"balance_unvx": balance,
		"balance_nvx":  float64(balance) / 1_000_000,
	})
}

func (s *APIServer) handleSend(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		From      string `json:"from"`
		To        string `json:"to"`
		Amount    int64  `json:"amount"`
		Fee       int64  `json:"fee"`
		PublicKey string `json:"public_key"`
		Signature string `json:"signature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if req.Signature == "" || req.PublicKey == "" {
		json.NewEncoder(w).Encode(map[string]string{
			"error": "ABGELEHNT: public_key und signature Pflicht.",
		})
		return
	}
	if req.Fee == 0 {
		req.Fee = 1000
	}
	tx, err := s.app.State.SignedTransfer(
		req.From, req.To, req.Amount, req.Fee,
		s.app.Height, req.PublicKey, req.Signature,
	)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(tx)
}

func (s *APIServer) handleTxs(w http.ResponseWriter, r *http.Request) {
	cors(w)
	txs := s.app.State.GetTransactions(20)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transactions": txs,
		"total":        len(txs),
	})
}

func (s *APIServer) handleAccounts(w http.ResponseWriter, r *http.Request) {
	cors(w)
	accounts := s.app.State.GetAllAccounts()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts": accounts,
		"total":    len(accounts),
	})
}

func (s *APIServer) handleMining(w http.ResponseWriter, r *http.Request) {
	cors(w)
	stats := s.app.Engine.GetMiningStats(s.app.Height)
	json.NewEncoder(w).Encode(stats)
}

func (s *APIServer) handleMempool(w http.ResponseWriter, r *http.Request) {
	cors(w)
	stats := s.app.Mempool.Stats()
	pending := s.app.Mempool.GetPending(20)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stats":   stats,
		"pending": pending,
	})
}

func (s *APIServer) handleMempoolSubmit(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount int64  `json:"amount"`
		Fee    int64  `json:"fee"`
		Nonce  uint64 `json:"nonce"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	tx, err := s.app.Mempool.Submit(req.From, req.To, req.Amount, req.Fee, req.Nonce)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(tx)
}

func (s *APIServer) handleP2P(w http.ResponseWriter, r *http.Request) {
	cors(w)
	stats := s.app.P2P.Stats()
	json.NewEncoder(w).Encode(stats)
}

func (s *APIServer) handleP2PConnect(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if err := s.app.P2P.Connect(req.Address); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "connecting", "address": req.Address})
}

func (s *APIServer) handleConsensus(w http.ResponseWriter, r *http.Request) {
	cors(w)
	stats := s.app.BFT.Stats()
	json.NewEncoder(w).Encode(stats)
}

func (s *APIServer) handleEVMStats(w http.ResponseWriter, r *http.Request) {
	cors(w)
	stats := s.app.EVM.Stats()
	json.NewEncoder(w).Encode(stats)
}

func (s *APIServer) handleEVMContracts(w http.ResponseWriter, r *http.Request) {
	cors(w)
	contracts := s.app.EVM.GetAllContracts()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"contracts": contracts,
		"total":     len(contracts),
	})
}

func (s *APIServer) handleEVMDeploy(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		Deployer string `json:"deployer"`
		Bytecode string `json:"bytecode"`
		ABI      string `json:"abi"`
		GasLimit uint64 `json:"gas_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if req.GasLimit == 0 { req.GasLimit = 3_000_000 }
	contract, result, err := s.app.EVM.DeployContract(
		req.Deployer, req.Bytecode, req.ABI,
		s.app.Blockchain.Height(), req.GasLimit,
	)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"contract": contract,
		"result":   result,
	})
}

func (s *APIServer) handleEVMCall(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		Caller   string `json:"caller"`
		Contract string `json:"contract"`
		Calldata string `json:"calldata"`
		GasLimit uint64 `json:"gas_limit"`
		Value    int64  `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if req.GasLimit == 0 { req.GasLimit = 1_000_000 }
	result, err := s.app.EVM.CallContract(
		req.Caller, req.Contract, req.Calldata,
		s.app.Blockchain.Height(), req.GasLimit, req.Value,
	)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) handleDEXStats(w http.ResponseWriter, r *http.Request) {
	cors(w)
	json.NewEncoder(w).Encode(s.app.DEX.Stats())
}

func (s *APIServer) handleDEXPools(w http.ResponseWriter, r *http.Request) {
	cors(w)
	pools := s.app.DEX.GetAllPools()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pools": pools,
		"total": len(pools),
	})
}

func (s *APIServer) handleDEXCreatePool(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		TokenA   string  `json:"token_a"`
		TokenB   string  `json:"token_b"`
		AmountA  float64 `json:"amount_a"`
		AmountB  float64 `json:"amount_b"`
		Creator  string  `json:"creator"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	pool, err := s.app.DEX.CreatePool(req.TokenA, req.TokenB, req.AmountA, req.AmountB, req.Creator)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(pool)
}

func (s *APIServer) handleDEXSwap(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		PoolID       string  `json:"pool_id"`
		TokenIn      string  `json:"token_in"`
		AmountIn     float64 `json:"amount_in"`
		MinAmountOut float64 `json:"min_amount_out"`
		Trader       string  `json:"trader"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	result, err := s.app.DEX.Swap(req.PoolID, req.TokenIn, req.AmountIn, req.MinAmountOut, req.Trader)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) handleDEXQuote(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		PoolID   string  `json:"pool_id"`
		TokenIn  string  `json:"token_in"`
		AmountIn float64 `json:"amount_in"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	quote, err := s.app.DEX.GetQuote(req.PoolID, req.TokenIn, req.AmountIn)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(quote)
}

func (s *APIServer) handleDEXAddLiquidity(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		PoolID   string  `json:"pool_id"`
		AmountA  float64 `json:"amount_a"`
		AmountB  float64 `json:"amount_b"`
		Provider string  `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	result, err := s.app.DEX.AddLiquidity(req.PoolID, req.AmountA, req.AmountB, req.Provider)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) handleDEXHistory(w http.ResponseWriter, r *http.Request) {
	cors(w)
	history := s.app.DEX.GetSwapHistory(20)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"swaps": history,
		"total": len(history),
	})
}

func (s *APIServer) handleStakingStats(w http.ResponseWriter, r *http.Request) {
	cors(w)
	json.NewEncoder(w).Encode(s.app.Staking.GetStats())
}

func (s *APIServer) handleStake(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		Owner   string `json:"owner"`
		Amount  uint64 `json:"amount"`
		IsGreen bool   `json:"is_green"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	position, err := s.app.Staking.Stake(req.Owner, req.Amount, req.IsGreen)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(position)
}

func (s *APIServer) handleUnstake(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		Owner      string `json:"owner"`
		PositionID string `json:"position_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	position, err := s.app.Staking.Unstake(req.Owner, req.PositionID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(position)
}

func (s *APIServer) handleStakingPositions(w http.ResponseWriter, r *http.Request) {
	cors(w)
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "owner required"})
		return
	}
	positions := s.app.Staking.GetPositionsByOwner(owner)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"positions": positions,
		"total":     len(positions),
	})
}

func (s *APIServer) handleStakingPending(w http.ResponseWriter, r *http.Request) {
	cors(w)
	positionID := r.URL.Query().Get("id")
	if positionID == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "id required"})
		return
	}
	pending := s.app.Staking.PendingRewards(positionID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"position_id":     positionID,
		"pending_rewards": pending,
	})
}

func (s *APIServer) handleGovStats(w http.ResponseWriter, r *http.Request) {
	cors(w)
	json.NewEncoder(w).Encode(s.app.Governance.GetStats())
}

func (s *APIServer) handleGovProposals(w http.ResponseWriter, r *http.Request) {
	cors(w)
	proposals := s.app.Governance.GetAllProposals()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"proposals": proposals,
		"total":     len(proposals),
	})
}

func (s *APIServer) handleGovProposal(w http.ResponseWriter, r *http.Request) {
	cors(w)
	idStr := r.URL.Query().Get("id")
	var id uint64
	fmt.Sscanf(idStr, "%d", &id)
	proposal, err := s.app.Governance.GetProposal(id)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(proposal)
}

func (s *APIServer) handleGovCreate(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		Proposer    string `json:"proposer"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Type        string `json:"type"`
		Deposit     uint64 `json:"deposit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	proposal, err := s.app.Governance.CreateProposal(
		req.Proposer, req.Title, req.Description,
		keeper.ProposalType(req.Type), req.Deposit,
	)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(proposal)
}

func (s *APIServer) handleGovVote(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		ProposalID uint64 `json:"proposal_id"`
		Voter      string `json:"voter"`
		Option     string `json:"option"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	vote, err := s.app.Governance.Vote(req.ProposalID, req.Voter, keeper.VoteOption(req.Option))
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(vote)
}

func (s *APIServer) handleEVMStatsV2(w http.ResponseWriter, r *http.Request) {
	cors(w)
	json.NewEncoder(w).Encode(s.app.EVM.StatsV2())
}

func (s *APIServer) handleEVMEstimateGas(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		json.NewEncoder(w).Encode(map[string]string{"error": "POST required"})
		return
	}
	var req struct {
		Caller   string `json:"caller"`
		Contract string `json:"contract"`
		Calldata string `json:"calldata"`
		Value    int64  `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	estimate, err := s.app.EVM.EstimateGas(req.Caller, req.Contract, req.Calldata, req.Value)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(estimate)
}

func (s *APIServer) handleEVMContractInfo(w http.ResponseWriter, r *http.Request) {
	cors(w)
	address := r.URL.Query().Get("address")
	if address == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "address required"})
		return
	}
	info, err := s.app.EVM.GetContractInfo(address)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(info)
}

func (s *APIServer) handleNVX20ABI(w http.ResponseWriter, r *http.Request) {
	cors(w)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"standard": "NVX-20",
		"compatible_with": "ERC-20",
		"abi": keeper.NVX20TokenABI,
		"description": "Official Nuvex token standard — deploy ERC-20 compatible tokens on Nuvex",
	})
}

func (s *APIServer) handleLevelDBStats(w http.ResponseWriter, r *http.Request) {
	cors(w)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"storage": "LevelDB",
		"status": "active",
		"description": "High-performance key-value storage — same as Bitcoin and Ethereum",
	})
}

func (s *APIServer) handleTxByAddress(w http.ResponseWriter, r *http.Request) {
	cors(w)
	address := r.URL.Query().Get("address")
	if address == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "address required"})
		return
	}
	txs := s.app.State.GetTransactions(50)
	filtered := make([]interface{}, 0)
	for _, tx := range txs {
		if tx.From == address || tx.To == address {
			filtered = append(filtered, tx)
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"address": address,
		"transactions": filtered,
		"total": len(filtered),
	})
}

func (s *APIServer) handleTxByHash(w http.ResponseWriter, r *http.Request) {
	cors(w)
	hash := r.URL.Query().Get("hash")
	if hash == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "hash required"})
		return
	}
	txs := s.app.State.GetTransactions(1000)
	for _, tx := range txs {
		if tx.Hash == hash {
			json.NewEncoder(w).Encode(tx)
			return
		}
	}
	json.NewEncoder(w).Encode(map[string]string{"error": "TX nicht gefunden"})
}
