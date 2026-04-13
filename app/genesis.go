package app

import "fmt"

type GenesisAccount struct {
	Address string
	Amount  int64
	Label   string
}

var GenesisAccounts = []GenesisAccount{
	{
		Address: "nuvex1cac179655ab9fd3543fe537675a3c3eb447691",
		Amount:  275_000_000_000_000,
		Label:   "Community Fair Launch — 275,000,000 NVX (55%)",
	},
	{
		Address: "nuvex1f81358e9a3840c5ca4647f2bc90b0e8b7b7a70",
		Amount:  100_000_000_000_000,
		Label:   "Ecosystem Fund — 100,000,000 NVX (20%)",
	},
	{
		Address: "nuvex1c22e41daadec8a5856631fbdd11a40b69b0248",
		Amount:  100_000_000_000_000,
		Label:   "Staking Rewards — 100,000,000 NVX (20%)",
	},
	{
		Address: "nuvex19d85718c8da8f4213e1a2a41fe894ba928b9c9",
		Amount:  25_000_000_000_000,
		Label:   "Founders Anonymous — 25,000,000 NVX (5%)",
	},
}

func PrintGenesis() {
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║        NUVEX GENESIS BLOCK — FAIR LAUNCH                 ║")
	fmt.Println("║        Chain ID: nuvex-1                                 ║")
	fmt.Println("║        No ICO. No Presale. Community First.              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	var total int64
	for _, acc := range GenesisAccounts {
		fmt.Printf("  ✅ %s\n", acc.Label)
		fmt.Printf("     Address: %s\n\n", acc.Address)
		total += acc.Amount
	}

	fmt.Printf("  TOTAL: %d unvx = 500,000,000 NVX\n", total)
	fmt.Println("  Fair Launch. Anonym. Dezentral. Für immer.")
}
