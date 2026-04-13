package tests

import (
	"fmt"
	"testing"
	"os"
	"github.com/nuvex-foundation/nuvex/x/nvx/keeper"
	"github.com/nuvex-foundation/nuvex/x/nvx/types"
)

// ─────────────────────────────────────────────
//  Test Suite fuer Nuvex
//  Fuehre aus mit: go test ./tests/...
// ─────────────────────────────────────────────

// ── Kryptographie Tests ───────────────────────

func TestKeyGeneration(t *testing.T) {
	fmt.Println("Test: Schluesselgenerierung...")
	priv, pub, err := keeper.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Schluesselgenerierung fehlgeschlagen: %v", err)
	}
	if len(priv.ToHex()) != 64 {
		t.Fatalf("Private Key falsche Laenge: %d", len(priv.ToHex()))
	}
	if len(pub.ToHex()) != 128 {
		t.Fatalf("Public Key falsche Laenge: %d", len(pub.ToHex()))
	}
	fmt.Println("  Schluesselgenerierung: OK")
}

func TestAddressGeneration(t *testing.T) {
	fmt.Println("Test: Adressgenerierung...")
	_, pub, _ := keeper.GenerateKeyPair()
	addr := pub.NuvexAddress()
	if len(addr) < 10 {
		t.Fatalf("Adresse zu kurz: %s", addr)
	}
	if addr[:6] != "nuvex1" {
		t.Fatalf("Adresse hat falschen Prefix: %s", addr[:6])
	}
	fmt.Printf("  Adresse: %s OK\n", addr[:20])
}

func TestSignAndVerify(t *testing.T) {
	fmt.Println("Test: Signierung und Verifikation...")
	priv, pub, _ := keeper.GenerateKeyPair()
	txHash := keeper.HashTransaction(
		pub.NuvexAddress(),
		"nuvex1bc045405ac3fb399aafbe8bb5295919e483154",
		1000000000, 5000, 1,
	)
	sig, err := priv.SignToHex(txHash)
	if err != nil {
		t.Fatalf("Signierung fehlgeschlagen: %v", err)
	}
	if err := keeper.VerifyECDSA(pub, txHash, sig); err != nil {
		t.Fatalf("Verifikation fehlgeschlagen: %v", err)
	}
	fmt.Println("  Signierung + Verifikation: OK")
}

func TestInvalidSignature(t *testing.T) {
	fmt.Println("Test: Ungueltige Signatur wird abgelehnt...")
	_, pub, _ := keeper.GenerateKeyPair()
	priv2, _, _ := keeper.GenerateKeyPair()
	txHash := keeper.HashTransaction("nuvex1test", "nuvex1test2", 1000, 100, 1)
	sig, _ := priv2.SignToHex(txHash)
	err := keeper.VerifyECDSA(pub, txHash, sig)
	if err == nil {
		t.Fatal("Ungueltige Signatur wurde akzeptiert!")
	}
	fmt.Println("  Ungueltige Signatur abgelehnt: OK")
}

// ── Blockchain Tests ──────────────────────────

func TestGenesisBlock(t *testing.T) {
	fmt.Println("Test: Genesis Block...")
	bc := keeper.NewBlockchain("/tmp/test_chain.db")
	defer os.Remove("/tmp/test_chain.db")
	if bc.TotalBlocks() == 0 {
		t.Fatal("Genesis Block nicht erstellt")
	}
	genesis := bc.GetBlock(0)
	if genesis == nil {
		t.Fatal("Genesis Block nicht abrufbar")
	}
	if genesis.Height != 0 {
		t.Fatalf("Genesis Block Hoehe falsch: %d", genesis.Height)
	}
	fmt.Printf("  Genesis Hash: %s... OK\n", genesis.Hash[:16])
}

func TestBlockAddition(t *testing.T) {
	fmt.Println("Test: Block hinzufuegen...")
	bc := keeper.NewBlockchain("/tmp/test_chain2.db")
	defer os.Remove("/tmp/test_chain2.db")
	initialHeight := bc.Height()
	bc.AddBlock("nuvex1test", []string{"tx1", "tx2"}, 5000, 5, 50000000)
	if bc.Height() != initialHeight+1 {
		t.Fatalf("Block nicht hinzugefuegt: %d", bc.Height())
	}
	fmt.Printf("  Block hinzugefuegt: Height %d OK\n", bc.Height())
}

func TestChainValidation(t *testing.T) {
	fmt.Println("Test: Chain Validierung...")
	bc := keeper.NewBlockchain("/tmp/test_chain3.db")
	defer os.Remove("/tmp/test_chain3.db")
	bc.AddBlock("nuvex1test", []string{}, 0, 0, 50000000)
	bc.AddBlock("nuvex1test", []string{}, 0, 0, 50000000)
	bc.AddBlock("nuvex1test", []string{}, 0, 0, 50000000)
	if err := bc.ValidateChain(); err != nil {
		t.Fatalf("Chain Validierung fehlgeschlagen: %v", err)
	}
	fmt.Println("  Chain Validierung: OK")
}

func TestPersistence(t *testing.T) {
	fmt.Println("Test: Persistenz...")
	bc1 := keeper.NewBlockchain("/tmp/test_persist.db")
	defer os.Remove("/tmp/test_persist.db")
	bc1.AddBlock("nuvex1test", []string{"tx1"}, 1000, 1, 50000000)
	height1 := bc1.Height()
	bc2 := keeper.NewBlockchain("/tmp/test_persist.db")
	if bc2.Height() != height1 {
		t.Fatalf("Persistenz fehlgeschlagen: %d != %d", bc2.Height(), height1)
	}
	fmt.Printf("  Persistenz: Height %d geladen OK\n", bc2.Height())
}

// ── Mempool Tests ─────────────────────────────

func TestMempoolSubmit(t *testing.T) {
	fmt.Println("Test: Mempool Submit...")
	mp := keeper.NewMempool()
	tx, err := mp.Submit("nuvex1from", "nuvex1to", 1000000, 1000, 1)
	if err != nil {
		t.Fatalf("Mempool Submit fehlgeschlagen: %v", err)
	}
	if mp.Size() != 1 {
		t.Fatalf("Mempool Groesse falsch: %d", mp.Size())
	}
	if tx.Status != "pending" {
		t.Fatalf("TX Status falsch: %s", tx.Status)
	}
	fmt.Println("  Mempool Submit: OK")
}

func TestMempoolMinFee(t *testing.T) {
	fmt.Println("Test: Mempool Mindest-Fee...")
	mp := keeper.NewMempool()
	_, err := mp.Submit("nuvex1from", "nuvex1to", 1000000, 10, 1)
	if err == nil {
		t.Fatal("Zu niedrige Fee wurde akzeptiert!")
	}
	fmt.Println("  Mindest-Fee Pruefung: OK")
}

func TestMempoolPriority(t *testing.T) {
	fmt.Println("Test: Mempool Prioritaet...")
	mp := keeper.NewMempool()
	mp.Submit("nuvex1a", "nuvex1b", 1000, 500, 1)
	mp.Submit("nuvex1c", "nuvex1d", 1000, 9000, 2)
	mp.Submit("nuvex1e", "nuvex1f", 1000, 1000, 3)
	selected := mp.SelectForBlock(3)
	if len(selected) != 3 {
		t.Fatalf("Falsche Anzahl TXs: %d", len(selected))
	}
	if selected[0].Fee < selected[1].Fee {
		t.Fatal("Prioritaet falsch sortiert!")
	}
	fmt.Println("  Prioritaet: OK")
}

// ── Tokenomics Tests ──────────────────────────

func TestMaxSupply(t *testing.T) {
	fmt.Println("Test: Max Supply...")
	if types.MaxSupply != int64(500_000_000_000_000) {
		t.Fatalf("Max Supply falsch: %d", types.MaxSupply)
	}
	fmt.Printf("  Max Supply: %d unvx OK\n", types.MaxSupply)
}

func TestBurnRate(t *testing.T) {
	fmt.Println("Test: Burn Rate...")
	fee := int64(10_000)
	burned := types.CalculateBurnAmount(fee)
	expected := int64(10)
	if burned != expected {
		t.Fatalf("Burn Amount falsch: %d (erwartet %d)", burned, expected)
	}
	fmt.Printf("  Burn Rate: %d fee -> %d burned OK\n", fee, burned)
}

func TestHalvingSchedule(t *testing.T) {
	fmt.Println("Test: Halving Schedule...")
	schedule := keeper.GetHalvingSchedule()
	if len(schedule) == 0 {
		t.Fatal("Halving Schedule leer")
	}
	if schedule[0].Reward != 50.0 {
		t.Fatalf("Initiale Belohnung falsch: %f", schedule[0].Reward)
	}
	if schedule[1].Reward != 25.0 {
		t.Fatalf("Erste Halbierung falsch: %f", schedule[1].Reward)
	}
	fmt.Printf("  Halving: %d Epochen OK\n", len(schedule))
}

// ── State Tests ───────────────────────────────

func TestGenesisBalances(t *testing.T) {
	fmt.Println("Test: Genesis Balances...")
	state := keeper.NewStateKeeper("/tmp/test_state.json")
	defer os.Remove("/tmp/test_state.json")
	state.InitGenesis([]keeper.GenesisEntry{
		{Address: "nuvex1founder", Amount: 25_000_000_000_000, PublicKey: ""},
		{Address: "nuvex1community", Amount: 275_000_000_000_000, PublicKey: ""},
	})
	if state.GetBalance("nuvex1founder") != 25_000_000_000_000 {
		t.Fatal("Founder Balance falsch")
	}
	if state.GetBalance("nuvex1community") != 275_000_000_000_000 {
		t.Fatal("Community Balance falsch")
	}
	fmt.Println("  Genesis Balances: OK")
}
