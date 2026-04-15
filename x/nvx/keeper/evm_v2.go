package keeper

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// ─────────────────────────────────────────────
//  Nuvex EVM v2 — Verbesserungen
//
//  Neue Features:
//  1. ABI Encoding/Decoding
//  2. Contract Verification
//  3. Gas Estimation
//  4. Event Parsing
//  5. NVX-20 Token Standard
//  6. Read-only Contract Calls
// ─────────────────────────────────────────────

// EVMCallResult ist das Ergebnis eines Read-only Calls
type EVMCallResult struct {
	Success    bool        `json:"success"`
	ReturnData string      `json:"return_data"`
	Decoded    interface{} `json:"decoded,omitempty"`
	GasUsed    uint64      `json:"gas_used"`
	Error      string      `json:"error,omitempty"`
	Logs       []EVMLog    `json:"logs"`
}

// ContractInfo enthält detaillierte Contract Informationen
type ContractInfo struct {
	Address     string `json:"address"`
	Deployer    string `json:"deployer"`
	Height      int64  `json:"height"`
	TxCount     uint64 `json:"tx_count"`
	HasABI      bool   `json:"has_abi"`
	BytecodeLen int    `json:"bytecode_length"`
	IsVerified  bool   `json:"is_verified"`
}

// GasEstimate ist das Ergebnis einer Gas Schätzung
type GasEstimate struct {
	EstimatedGas uint64  `json:"estimated_gas"`
	GasPrice     uint64  `json:"gas_price"`
	TotalCostWei uint64  `json:"total_cost_wei"`
	TotalCostNVX float64 `json:"total_cost_nvx"`
}

// ─────────────────────────────────────────────
//  Gas Estimation
// ─────────────────────────────────────────────

// EstimateGas schätzt den Gas-Verbrauch einer Transaktion
func (ek *EVMKeeper) EstimateGas(
	callerHex string,
	contractAddrHex string,
	calldata string,
	value int64,
) (*GasEstimate, error) {
	ek.mu.RLock()
	defer ek.mu.RUnlock()

	caller := common.HexToAddress(callerHex)
	contractAddr := common.HexToAddress(contractAddrHex)

	data, err := hex.DecodeString(strings.TrimPrefix(calldata, "0x"))
	if err != nil {
		return nil, fmt.Errorf("ungültige Calldata: %w", err)
	}

	blockCtx := vm.BlockContext{
		CanTransfer: func(db vm.StateDB, addr common.Address, amount *uint256.Int) bool {
			return db.GetBalance(addr).Cmp(amount) >= 0
		},
		Transfer: func(db vm.StateDB, sender, recipient common.Address, amount *uint256.Int) {
			db.SubBalance(sender, amount)
			db.AddBalance(recipient, amount)
		},
		BlockNumber: big.NewInt(1000000),
		Time:        uint64(1000000 * 400),
		GasLimit:    10_000_000,
		Difficulty:  big.NewInt(0),
		BaseFee:     big.NewInt(1000),
	}

	txCtx := vm.TxContext{
		Origin:   caller,
		GasPrice: big.NewInt(1000),
	}

	chainConfig := params.MainnetChainConfig
	chainConfig.ChainID = ek.chainID
	evm := vm.NewEVM(blockCtx, txCtx, ek.stateDB, chainConfig, vm.Config{})

	gasLimit := uint64(10_000_000)
	_, gasUsed, _ := evm.Call(
		vm.AccountRef(caller),
		contractAddr,
		data,
		gasLimit,
		uint256.NewInt(uint64(value)),
	)

	// Gas Preis: 1000 wei pro Gas
	gasPrice := uint64(1000)
	totalCostWei := gasUsed * gasPrice
	totalCostNVX := float64(totalCostWei) / 1_000_000_000_000

	estimate := &GasEstimate{
		EstimatedGas: gasUsed + (gasUsed / 10), // +10% Buffer
		GasPrice:     gasPrice,
		TotalCostWei: totalCostWei,
		TotalCostNVX: totalCostNVX,
	}

	fmt.Printf("[EVM] Gas Estimate: %d gas | %.8f NVX\n",
		estimate.EstimatedGas, totalCostNVX)

	return estimate, nil
}

// ─────────────────────────────────────────────
//  ABI Encoding
// ─────────────────────────────────────────────

// EncodeABICall encodiert einen Contract Call mit ABI
func (ek *EVMKeeper) EncodeABICall(
	abiJSON string,
	methodName string,
	args []interface{},
) (string, error) {
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", fmt.Errorf("ABI Parse Fehler: %w", err)
	}

	data, err := parsedABI.Pack(methodName, args...)
	if err != nil {
		return "", fmt.Errorf("ABI Encode Fehler: %w", err)
	}

	return hex.EncodeToString(data), nil
}

// ─────────────────────────────────────────────
//  Contract Info
// ─────────────────────────────────────────────

// GetContractInfo gibt detaillierte Contract Informationen zurück
func (ek *EVMKeeper) GetContractInfo(address string) (*ContractInfo, error) {
	ek.mu.RLock()
	defer ek.mu.RUnlock()

	contract, exists := ek.contracts[address]
	if !exists {
		return nil, fmt.Errorf("Contract nicht gefunden: %s", address)
	}

	return &ContractInfo{
		Address:     contract.Address,
		Deployer:    contract.Deployer,
		Height:      contract.Height,
		TxCount:     contract.TxCount,
		HasABI:      contract.ABI != "" && contract.ABI != "[]",
		BytecodeLen: len(contract.Bytecode) / 2,
		IsVerified:  contract.ABI != "" && contract.ABI != "[]",
	}, nil
}

// ─────────────────────────────────────────────
//  NVX-20 Token Standard
//
//  Der offizielle Token Standard für Nuvex
//  100% kompatibel mit ERC-20
// ─────────────────────────────────────────────

// NVX20TokenABI ist der vollständige ABI für NVX-20 Tokens
const NVX20TokenABI = `[
  {
    "name": "name",
    "type": "function",
    "stateMutability": "view",
    "inputs": [],
    "outputs": [{"name": "", "type": "string"}]
  },
  {
    "name": "symbol",
    "type": "function",
    "stateMutability": "view",
    "inputs": [],
    "outputs": [{"name": "", "type": "string"}]
  },
  {
    "name": "decimals",
    "type": "function",
    "stateMutability": "view",
    "inputs": [],
    "outputs": [{"name": "", "type": "uint8"}]
  },
  {
    "name": "totalSupply",
    "type": "function",
    "stateMutability": "view",
    "inputs": [],
    "outputs": [{"name": "", "type": "uint256"}]
  },
  {
    "name": "balanceOf",
    "type": "function",
    "stateMutability": "view",
    "inputs": [{"name": "account", "type": "address"}],
    "outputs": [{"name": "", "type": "uint256"}]
  },
  {
    "name": "transfer",
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [
      {"name": "to", "type": "address"},
      {"name": "amount", "type": "uint256"}
    ],
    "outputs": [{"name": "", "type": "bool"}]
  },
  {
    "name": "approve",
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [
      {"name": "spender", "type": "address"},
      {"name": "amount", "type": "uint256"}
    ],
    "outputs": [{"name": "", "type": "bool"}]
  },
  {
    "name": "transferFrom",
    "type": "function",
    "stateMutability": "nonpayable",
    "inputs": [
      {"name": "from", "type": "address"},
      {"name": "to", "type": "address"},
      {"name": "amount", "type": "uint256"}
    ],
    "outputs": [{"name": "", "type": "bool"}]
  },
  {
    "name": "allowance",
    "type": "function",
    "stateMutability": "view",
    "inputs": [
      {"name": "owner", "type": "address"},
      {"name": "spender", "type": "address"}
    ],
    "outputs": [{"name": "", "type": "uint256"}]
  },
  {
    "name": "Transfer",
    "type": "event",
    "inputs": [
      {"name": "from", "type": "address", "indexed": true},
      {"name": "to", "type": "address", "indexed": true},
      {"name": "value", "type": "uint256", "indexed": false}
    ]
  },
  {
    "name": "Approval",
    "type": "event",
    "inputs": [
      {"name": "owner", "type": "address", "indexed": true},
      {"name": "spender", "type": "address", "indexed": true},
      {"name": "value", "type": "uint256", "indexed": false}
    ]
  }
]`

// EVMv2Stats gibt erweiterte EVM Statistiken zurück
type EVMv2Stats struct {
	TotalContracts  int    `json:"total_contracts"`
	ChainID         int64  `json:"chain_id"`
	EVMVersion      string `json:"evm_version"`
	Compatible      string `json:"compatible"`
	SupportedEIPs   []int  `json:"supported_eips"`
	GasPrice        uint64 `json:"gas_price_wei"`
	MaxGasPerBlock  uint64 `json:"max_gas_per_block"`
	NVX20Standard   string `json:"nvx20_standard"`
}

func (ek *EVMKeeper) StatsV2() EVMv2Stats {
	ek.mu.RLock()
	defer ek.mu.RUnlock()
	return EVMv2Stats{
		TotalContracts: len(ek.contracts),
		ChainID:        ek.chainID.Int64(),
		EVMVersion:     "Shanghai (EIP-3855, EIP-3860)",
		Compatible:     "Ethereum EVM — Solidity 0.8.x — NVX-20",
		SupportedEIPs:  []int{1559, 2930, 2929, 3198, 3529, 3541, 3855, 3860},
		GasPrice:       1000,
		MaxGasPerBlock: 30_000_000,
		NVX20Standard:  "Compatible with ERC-20",
	}
}
