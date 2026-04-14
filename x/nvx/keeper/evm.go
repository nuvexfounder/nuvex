package keeper

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"github.com/holiman/uint256"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// ─────────────────────────────────────────────
//  Nuvex EVM Engine
//
//  Ermöglicht das Deployen und Ausführen von
//  Solidity Smart Contracts direkt auf Nuvex.
//
//  100% kompatibel mit Ethereum's EVM —
//  jeder Solidity Contract läuft ohne Änderungen.
// ─────────────────────────────────────────────

// EVMContract repräsentiert einen deployed Contract
type EVMContract struct {
	Address  string `json:"address"`
	Deployer string `json:"deployer"`
	Bytecode string `json:"bytecode"`
	ABI      string `json:"abi"`
	Height   int64  `json:"height"`
	TxCount  uint64 `json:"tx_count"`
}

// EVMResult ist das Ergebnis einer EVM Ausführung
type EVMResult struct {
	Success    bool   `json:"success"`
	ReturnData string `json:"return_data"`
	GasUsed    uint64 `json:"gas_used"`
	Error      string `json:"error,omitempty"`
	Logs       []EVMLog `json:"logs"`
}

// EVMLog ist ein Event das ein Contract emittiert
type EVMLog struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

// EVMKeeper verwaltet den EVM State
type EVMKeeper struct {
	mu        sync.RWMutex
	stateDB   *state.StateDB
	contracts map[string]*EVMContract
	chainID   *big.Int
}

// NewEVMKeeper erstellt einen neuen EVM Keeper
func NewEVMKeeper() (*EVMKeeper, error) {
	// In-memory Database für EVM State
	memDB := rawdb.NewMemoryDatabase()
	stateDB, err := state.New(common.Hash{}, state.NewDatabase(memDB), nil)
	if err != nil {
		return nil, fmt.Errorf("EVM State Fehler: %w", err)
	}

	// Nuvex Chain ID: 1317 (einzigartig)
	chainID := big.NewInt(1317)

	fmt.Println("[EVM] ✅ Nuvex EVM Engine gestartet")
	fmt.Printf("[EVM] ✅ Chain ID: %d (EVM kompatibel)\n", chainID)

	return &EVMKeeper{
		stateDB:   stateDB,
		contracts: make(map[string]*EVMContract),
		chainID:   chainID,
	}, nil
}

// ─────────────────────────────────────────────
//  Contract Deployment
// ─────────────────────────────────────────────

// DeployContract deployed einen Solidity Contract auf Nuvex
func (ek *EVMKeeper) DeployContract(
	deployerHex string,
	bytecodeHex string,
	abi string,
	height int64,
	gasLimit uint64,
) (*EVMContract, *EVMResult, error) {
	ek.mu.Lock()
	defer ek.mu.Unlock()

	// Bytecode dekodieren
	bytecode, err := hex.DecodeString(bytecodeHex)
	if err != nil {
		return nil, nil, fmt.Errorf("ungültiger Bytecode: %w", err)
	}

	// Deployer Adresse
	deployer := common.HexToAddress(deployerHex)

	// EVM Konfiguration
	vmConfig := vm.Config{}
	blockCtx := vm.BlockContext{
		CanTransfer: func(db vm.StateDB, addr common.Address, amount *uint256.Int) bool {
			return db.GetBalance(addr).Cmp(amount) >= 0
		},
		Transfer: func(db vm.StateDB, sender, recipient common.Address, amount *uint256.Int) {
			db.SubBalance(sender, amount)
			db.AddBalance(recipient, amount)
		},
		BlockNumber: big.NewInt(height),
		Time:        uint64(height * 400),
		GasLimit:    gasLimit,
		Difficulty:  big.NewInt(0),
		BaseFee:     big.NewInt(1000),
	}

	txCtx := vm.TxContext{
		Origin:   deployer,
		GasPrice: big.NewInt(1000),
	}

	// EVM erstellen
	chainConfig := params.MainnetChainConfig
	chainConfig.ChainID = ek.chainID
	evm := vm.NewEVM(blockCtx, txCtx, ek.stateDB, chainConfig, vmConfig)

	// Contract deployen
	if gasLimit == 0 {
		gasLimit = 3_000_000
	}

	_, contractAddr, gasUsed, vmerr := evm.Create(
		vm.AccountRef(deployer),
		bytecode,
		gasLimit,
		uint256.NewInt(0),
	)

	result := &EVMResult{
		GasUsed: gasUsed,
		Logs:    []EVMLog{},
	}

	if vmerr != nil {
		result.Success = false
		result.Error = vmerr.Error()
		return nil, result, nil
	}

	result.Success = true

	// Contract speichern
	contract := &EVMContract{
		Address:  contractAddr.Hex(),
		Deployer: deployerHex,
		Bytecode: bytecodeHex,
		ABI:      abi,
		Height:   height,
		TxCount:  0,
	}

	ek.contracts[contractAddr.Hex()] = contract

	// State committen
	ek.stateDB.Commit(uint64(height), false)

	fmt.Printf("[EVM] ✅ Contract deployed: %s | Gas: %d | Block: %d\n",
		contractAddr.Hex()[:20]+"...", gasUsed, height)

	return contract, result, nil
}

// ─────────────────────────────────────────────
//  Contract Aufruf
// ─────────────────────────────────────────────

// CallContract ruft eine Funktion eines deployed Contracts auf
func (ek *EVMKeeper) CallContract(
	callerHex string,
	contractAddrHex string,
	calldata string,
	height int64,
	gasLimit uint64,
	value int64,
) (*EVMResult, error) {
	ek.mu.Lock()
	defer ek.mu.Unlock()

	caller := common.HexToAddress(callerHex)
	contractAddr := common.HexToAddress(contractAddrHex)

	// Calldata dekodieren
	data, err := hex.DecodeString(calldata)
	if err != nil {
		return nil, fmt.Errorf("ungültige Calldata: %w", err)
	}

	vmConfig := vm.Config{}
	blockCtx := vm.BlockContext{
		CanTransfer: func(db vm.StateDB, addr common.Address, amount *uint256.Int) bool {
			return db.GetBalance(addr).Cmp(amount) >= 0
		},
		Transfer: func(db vm.StateDB, sender, recipient common.Address, amount *uint256.Int) {
			db.SubBalance(sender, amount)
			db.AddBalance(recipient, amount)
		},
		BlockNumber: big.NewInt(height),
		Time:        uint64(height * 400),
		GasLimit:    gasLimit,
		Difficulty:  big.NewInt(0),
		BaseFee:     big.NewInt(1000),
	}

	txCtx := vm.TxContext{
		Origin:   caller,
		GasPrice: big.NewInt(1000),
	}

	chainConfig := params.MainnetChainConfig
	chainConfig.ChainID = ek.chainID
	evm := vm.NewEVM(blockCtx, txCtx, ek.stateDB, chainConfig, vmConfig)

	if gasLimit == 0 {
		gasLimit = 1_000_000
	}

	ret, gasUsed, vmerr := evm.Call(
		vm.AccountRef(caller),
		contractAddr,
		data,
		gasLimit,
		uint256.NewInt(uint64(value)),
	)

	result := &EVMResult{
		GasUsed:    gasUsed,
		ReturnData: hex.EncodeToString(ret),
		Logs:       []EVMLog{},
	}

	if vmerr != nil {
		result.Success = false
		result.Error = vmerr.Error()
		return result, nil
	}

	result.Success = true

	// Logs sammeln
	for _, log := range ek.stateDB.Logs() {
		topics := make([]string, len(log.Topics))
		for i, t := range log.Topics {
			topics[i] = t.Hex()
		}
		result.Logs = append(result.Logs, EVMLog{
			Address: log.Address.Hex(),
			Topics:  topics,
			Data:    hex.EncodeToString(log.Data),
		})
	}

	if contract, exists := ek.contracts[contractAddrHex]; exists {
		contract.TxCount++
	}

	ek.stateDB.Commit(uint64(height), false)

	fmt.Printf("[EVM] Call: %s → %s | Gas: %d | Success: %v\n",
		callerHex[:10]+"...", contractAddrHex[:10]+"...", gasUsed, result.Success)

	return result, nil
}

// ─────────────────────────────────────────────
//  NVX-20 Token Standard
//
//  Equivalent zu Ethereum's ERC-20
//  Komplett in Solidity geschrieben
// ─────────────────────────────────────────────

// NVX20TokenBytecode ist der kompilierte Bytecode eines Standard NVX-20 Tokens
// (vereinfachter ERC-20 kompatibel)
const NVX20ABI = `[
  {"name":"transfer","type":"function","inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]},
  {"name":"balanceOf","type":"function","inputs":[{"name":"account","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
  {"name":"totalSupply","type":"function","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
  {"name":"approve","type":"function","inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]},
  {"name":"Transfer","type":"event","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"value","type":"uint256","indexed":false}]}
]`

// ─────────────────────────────────────────────
//  Abfragen & Statistiken
// ─────────────────────────────────────────────

// GetContract gibt einen deployed Contract zurück
func (ek *EVMKeeper) GetContract(address string) (*EVMContract, error) {
	ek.mu.RLock()
	defer ek.mu.RUnlock()
	contract, exists := ek.contracts[address]
	if !exists {
		return nil, fmt.Errorf("Contract nicht gefunden: %s", address)
	}
	return contract, nil
}

// GetAllContracts gibt alle deployed Contracts zurück
func (ek *EVMKeeper) GetAllContracts() []*EVMContract {
	ek.mu.RLock()
	defer ek.mu.RUnlock()
	contracts := make([]*EVMContract, 0, len(ek.contracts))
	for _, c := range ek.contracts {
		contracts = append(contracts, c)
	}
	return contracts
}

// GetEVMAddress konvertiert eine Nuvex Adresse zu einer EVM Adresse
func NuvexToEVMAddress(nuvexAddr string) common.Address {
	hash := crypto.Keccak256([]byte(nuvexAddr))
	return common.BytesToAddress(hash)
}

// EVMStats gibt EVM Statistiken zurück
type EVMStats struct {
	TotalContracts int    `json:"total_contracts"`
	ChainID        int64  `json:"chain_id"`
	EVMVersion     string `json:"evm_version"`
	Compatible     string `json:"compatible"`
}

func (ek *EVMKeeper) Stats() EVMStats {
	ek.mu.RLock()
	defer ek.mu.RUnlock()
	return EVMStats{
		TotalContracts: len(ek.contracts),
		ChainID:        ek.chainID.Int64(),
		EVMVersion:     "London (EIP-1559)",
		Compatible:     "Ethereum EVM — Solidity 0.8.x",
	}
}

// ToJSON serialisiert einen Contract zu JSON
func (c *EVMContract) ToJSON() string {
	data, _ := json.MarshalIndent(c, "", "  ")
	return string(data)
}
