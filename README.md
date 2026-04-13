# NUVEX (NVX) — Next Generation Layer-1 Blockchain

> *"The idea must stand on its own. No founder reputation. No central authority. Only code, cryptography, and community."*
> — Nuvex Foundation, 2025

## Current Status: Mainnet v1.0 — Active Development

## What Works Today
- Blockchain running 24/7 with persistent storage
- dBFT Proof-of-Stake consensus
- SECP256k1 cryptography (same as Bitcoin)
- Hard supply cap: 500,000,000 NVX
- Halving every 5 years
- 0.1% fee burn per transaction
- P2P network (port 26656)
- Mempool with fee-based priority
- Web Wallet & Block Explorer
- REST API
- 14 automated tests — all passing

## Roadmap
- Multi-node testing and TPS benchmarks
- Full EVM compatibility
- Mobile wallet iOS/Android
- DEX listing
- Independent security audit

## Fair Launch — No ICO — No Presale

| Category | % | Amount | Vesting |
|----------|---|--------|---------|
| Community Fair Launch | 55% | 275,000,000 NVX | Immediate |
| Ecosystem Fund | 20% | 100,000,000 NVX | Governance |
| Staking Rewards | 20% | 100,000,000 NVX | 10yr linear |
| Founders Anonymous | 5% | 25,000,000 NVX | 4yr vesting |

## Halving Schedule

| Year | Reward per Block |
|------|-----------------|
| 2025 | 50.0000 NVX |
| 2030 | 25.0000 NVX |
| 2035 | 12.5000 NVX |
| 2040 | 6.2500 NVX |
| 2045 | 3.1250 NVX |

## Links

| | |
|---|---|
| Website | https://nuvex-chain.io |
| Explorer | https://nuvex-chain.io/index.html |
| Wallet | https://nuvex-chain.io/wallet.html |
| Docs | https://nuvex-chain.io/docs.html |
| API | https://nuvex-chain.io:1317/status |
| P2P | node.nuvex-chain.io:26656 |

## Run a Node

Requirements: Ubuntu 20.04+, Go 1.21+, 2GB RAM, 50GB disk

```bash
git clone https://github.com/nuvexfounder/nuvex
cd nuvex
go build -o nuvexd cmd/nuvexd/main.go
./nuvexd start node.nuvex-chain.io:26656
```

## License

MIT — Open Source forever.

---
*Nuvex Foundation — Anonymous — https://nuvex-chain.io*
