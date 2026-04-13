# NUVEX (NVX)
### Next Generation Layer-1 Blockchain

> *"The idea must stand on its own. No founder's reputation. No central authority. Only code, cryptography, and community."*
> — Nuvex Foundation, 2025

---

## What is Nuvex?

Nuvex is a next-generation Layer-1 blockchain that combines Bitcoin's scarcity with Ethereum's programmability — faster, greener, and built for the next billion users.

| Feature | Bitcoin | Ethereum | **Nuvex** |
|---------|---------|----------|-----------|
| Hard Supply Cap | ✅ 21M | ❌ None | ✅ **500M** |
| Smart Contracts | ❌ | ✅ | ✅ |
| TPS | 7 | ~30 | **100,000+** |
| Finality | ~60 min | ~12 sec | **< 1 second** |
| Energy/TX | ~700 kWh | ~0.03 kWh | **< 0.00001 kWh** |
| Halving | Every 4yr | None | **Every 5yr** |
| Fair Launch | ✅ | ❌ | ✅ |

---

## Key Features

- **dBFT Proof-of-Stake** — Sub-400ms block finality, no mining
- **500M Hard Cap** — With halving every 5 years
- **0.1% Burn per TX** — Deflationary by design
- **EVM Compatible** — Deploy Solidity contracts directly
- **SECP256k1 Cryptography** — Same security as Bitcoin
- **P2P Network** — Fully decentralized, anyone can run a node
- **Green** — 99.99% less energy than Bitcoin

---

## Mainnet

| | |
|---|---|
| Chain ID | `nuvex-1` |
| Native Token | `NVX` (micro: `unvx`) |
| Max Supply | 500,000,000 NVX |
| Block Time | ~400ms |
| API | `http://nuvex-chain.io:1317` |
| P2P | `nuvex-chain.io:26656` |
| Explorer | `http://nuvex-chain.io` |
| Wallet | `http://nuvex-chain.io/wallet.html` |

---

## Tokenomics — Fair Launch

| Category | % | Amount | Vesting |
|----------|---|--------|---------|
| Community Fair Launch | 55% | 275,000,000 NVX | Immediate |
| Ecosystem Fund | 20% | 100,000,000 NVX | Governance |
| Staking Rewards | 20% | 100,000,000 NVX | 10yr linear |
| Founders (Anonymous) | 5% | 25,000,000 NVX | 4yr vesting |

**No ICO. No presale. No VC allocation.**

---

## Halving Schedule

| Year | Reward/Block |
|------|-------------|
| 2025 | 50.0000 NVX |
| 2030 | 25.0000 NVX |
| 2035 | 12.5000 NVX |
| 2040 | 6.2500 NVX |
| 2045 | 3.1250 NVX |

---

## Run a Node

```bash
# Requirements: Ubuntu 20.04+, Go 1.21+, 2GB RAM, 50GB disk

git clone https://github.com/nuvexfounder/nuvex
cd nuvex
go build -o nuvexd cmd/nuvexd/main.go

# Start and connect to mainnet
./nuvexd start nuvex-chain.io:26656
```

---

## API

```bash
# Chain status
curl http://nuvex-chain.io:1317/status

# Balance
curl http://nuvex-chain.io:1317/balance/YOUR_ADDRESS

# Submit TX
curl -X POST http://nuvex-chain.io:1317/mempool/submit \
  -H "Content-Type: application/json" \
  -d '{"from":"nuvex1...","to":"nuvex1...","amount":1000000,"fee":1000,"nonce":1}'
```

---

## Architecture
ssh root@187.124.184.226
cd ~/nuvex
cat > README.md << 'EOF'
# NUVEX (NVX) — Next Generation Layer-1 Blockchain

Nuvex combines Bitcoin scarcity with Ethereum programmability. Faster, greener, built for the next billion users.

## Key Stats
- Max Supply: 500,000,000 NVX
- Block Time: 400ms
- Finality: Sub-second
- Energy: 99.99% less than Bitcoin
- Consensus: dBFT Proof-of-Stake
- Halving: Every 5 years

## Fair Launch
No ICO. No presale. Community first.

## Links
- Explorer: http://nuvex-chain.io
- Wallet: http://nuvex-chain.io/wallet.html
- Docs: http://nuvex-chain.io/docs.html

## Run a Node
git clone https://github.com/nuvexfounder/nuvex
cd nuvex
go build -o nuvexd cmd/nuvexd/main.go
./nuvexd start

*Nuvex Foundation — Anonymous — 2025*
