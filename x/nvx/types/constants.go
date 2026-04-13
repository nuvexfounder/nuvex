package types

const (
	Name        = "Nuvex"
	ChainID     = "nuvex-1"
	Bech32Prefix = "nuvex"
	BondDenom   = "unvx"
	DisplayDenom = "nvx"
	MaxSupply   = int64(500_000_000_000_000)
	BurnRateBps = int64(10)
	MinStakeMicroNVX = int64(100_000_000)
	TargetBlockTimeMs = 400
	MaxValidators = 512
)

func NVXToMicro(nvx int64) int64 { return nvx * 1_000_000 }
func MicroToNVX(unvx int64) float64 { return float64(unvx) / 1_000_000 }

func CalculateBurnAmount(fee int64) int64 {
	return fee * BurnRateBps / 10_000
}
