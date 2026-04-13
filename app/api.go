package app

import (
	"encoding/json"
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
