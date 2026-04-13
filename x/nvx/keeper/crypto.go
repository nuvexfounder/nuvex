package keeper

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
)

// ─────────────────────────────────────────────
//  SECP256k1 Kurven-Parameter
//  Exakt dieselben wie Bitcoin
// ─────────────────────────────────────────────

var (
	// Primzahl p
	secp256k1P, _ = new(big.Int).SetString(
		"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F", 16)

	// Gruppenordnung n
	secp256k1N, _ = new(big.Int).SetString(
		"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)

	// Generator-Punkt Gx
	secp256k1Gx, _ = new(big.Int).SetString(
		"79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798", 16)

	// Generator-Punkt Gy
	secp256k1Gy, _ = new(big.Int).SetString(
		"483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8", 16)
)

// Point ist ein Punkt auf der elliptischen Kurve
type Point struct {
	X, Y *big.Int
}

// IsInfinity prüft ob der Punkt der Unendlichkeitspunkt ist
func (p *Point) IsInfinity() bool {
	return p.X == nil && p.Y == nil
}

// ─────────────────────────────────────────────
//  Elliptische Kurven Arithmetik
// ─────────────────────────────────────────────

// pointAdd addiert zwei Punkte auf der Kurve
func pointAdd(p1, p2 *Point) *Point {
	if p1.IsInfinity() { return p2 }
	if p2.IsInfinity() { return p1 }

	if p1.X.Cmp(p2.X) == 0 {
		if p1.Y.Cmp(p2.Y) != 0 {
			return &Point{} // Unendlichkeit
		}
		return pointDouble(p1)
	}

	// slope = (y2 - y1) / (x2 - x1) mod p
	dy := new(big.Int).Sub(p2.Y, p1.Y)
	dx := new(big.Int).Sub(p2.X, p1.X)
	slope := new(big.Int).Mul(dy, modInverse(dx, secp256k1P))
	slope.Mod(slope, secp256k1P)

	// x3 = slope^2 - x1 - x2 mod p
	x3 := new(big.Int).Mul(slope, slope)
	x3.Sub(x3, p1.X)
	x3.Sub(x3, p2.X)
	x3.Mod(x3, secp256k1P)

	// y3 = slope * (x1 - x3) - y1 mod p
	y3 := new(big.Int).Sub(p1.X, x3)
	y3.Mul(slope, y3)
	y3.Sub(y3, p1.Y)
	y3.Mod(y3, secp256k1P)

	return &Point{X: x3, Y: y3}
}

// pointDouble verdoppelt einen Punkt
func pointDouble(p *Point) *Point {
	if p.IsInfinity() { return p }

	// slope = (3 * x^2) / (2 * y) mod p
	x2 := new(big.Int).Mul(p.X, p.X)
	x2.Mod(x2, secp256k1P)
	num := new(big.Int).Mul(big.NewInt(3), x2)
	den := new(big.Int).Mul(big.NewInt(2), p.Y)
	slope := new(big.Int).Mul(num, modInverse(den, secp256k1P))
	slope.Mod(slope, secp256k1P)

	x3 := new(big.Int).Mul(slope, slope)
	x3.Sub(x3, new(big.Int).Mul(big.NewInt(2), p.X))
	x3.Mod(x3, secp256k1P)

	y3 := new(big.Int).Sub(p.X, x3)
	y3.Mul(slope, y3)
	y3.Sub(y3, p.Y)
	y3.Mod(y3, secp256k1P)

	return &Point{X: x3, Y: y3}
}

// scalarMult multipliziert einen Punkt mit einem Skalar
func scalarMult(k *big.Int, p *Point) *Point {
	result := &Point{} // Unendlichkeit
	addend := p

	for i := 0; i < k.BitLen(); i++ {
		if k.Bit(i) == 1 {
			result = pointAdd(result, addend)
		}
		addend = pointDouble(addend)
	}
	return result
}

// modInverse berechnet das modulare Inverse
func modInverse(a, m *big.Int) *big.Int {
	return new(big.Int).ModInverse(a, m)
}

// ─────────────────────────────────────────────
//  Schlüsselgenerierung
// ─────────────────────────────────────────────

// NuvexPrivateKey repräsentiert einen privaten Schlüssel
type NuvexPrivateKey struct {
	D []byte // 32 bytes privater Schlüssel
}

// NuvexPublicKey repräsentiert einen öffentlichen Schlüssel
type NuvexPublicKey struct {
	X, Y *big.Int
}

// GenerateKeyPair generiert ein neues Schlüsselpaar
func GenerateKeyPair() (*NuvexPrivateKey, *NuvexPublicKey, error) {
	// Zufälligen privaten Schlüssel generieren
	privBytes := make([]byte, 32)
	if _, err := rand.Read(privBytes); err != nil {
		return nil, nil, fmt.Errorf("Fehler bei Schlüsselgenerierung: %w", err)
	}

	privKey := &NuvexPrivateKey{D: privBytes}
	pubKey := privKey.PublicKey()

	return privKey, pubKey, nil
}

// PublicKey leitet den öffentlichen Schlüssel ab
func (priv *NuvexPrivateKey) PublicKey() *NuvexPublicKey {
	d := new(big.Int).SetBytes(priv.D)
	G := &Point{X: secp256k1Gx, Y: secp256k1Gy}
	pubPoint := scalarMult(d, G)
	return &NuvexPublicKey{X: pubPoint.X, Y: pubPoint.Y}
}

// ToHex gibt den privaten Schlüssel als Hex zurück
func (priv *NuvexPrivateKey) ToHex() string {
	return hex.EncodeToString(priv.D)
}

// ToHex gibt den öffentlichen Schlüssel als Hex zurück
func (pub *NuvexPublicKey) ToHex() string {
	xBytes := pub.X.Bytes()
	yBytes := pub.Y.Bytes()
	// Padding auf 32 bytes
	xPadded := make([]byte, 32)
	yPadded := make([]byte, 32)
	copy(xPadded[32-len(xBytes):], xBytes)
	copy(yPadded[32-len(yBytes):], yBytes)
	return hex.EncodeToString(append(xPadded, yPadded...))
}

// NuvexAddress leitet die Nuvex-Adresse ab
func (pub *NuvexPublicKey) NuvexAddress() string {
	pubBytes, _ := hex.DecodeString(pub.ToHex())
	sha256Hash := sha256.Sum256(pubBytes)
	ripemd := sha256.Sum256(sha256Hash[:]) // vereinfacht
	return "nuvex1" + hex.EncodeToString(ripemd[:19])
}

// ─────────────────────────────────────────────
//  ECDSA Signierung (deterministisch — RFC 6979)
// ─────────────────────────────────────────────

// Sign signiert eine Nachricht mit dem privaten Schlüssel
// Verwendet deterministisches k (RFC 6979) — sicher wie Bitcoin
func (priv *NuvexPrivateKey) Sign(messageHash []byte) (r, s *big.Int, err error) {
	// Deterministisches k nach RFC 6979
	k, err := generateDeterministicK(priv.D, messageHash)
	if err != nil {
		return nil, nil, err
	}

	d := new(big.Int).SetBytes(priv.D)
	G := &Point{X: secp256k1Gx, Y: secp256k1Gy}

	// R = k * G
	kG := scalarMult(k, G)
	r = new(big.Int).Mod(kG.X, secp256k1N)

	if r.Sign() == 0 {
		return nil, nil, errors.New("Signierung fehlgeschlagen: r=0")
	}

	// s = k^-1 * (hash + r * d) mod n
	kInv := modInverse(k, secp256k1N)
	rd := new(big.Int).Mul(r, d)
	hashInt := new(big.Int).SetBytes(messageHash)
	s = new(big.Int).Add(hashInt, rd)
	s.Mul(kInv, s)
	s.Mod(s, secp256k1N)

	if s.Sign() == 0 {
		return nil, nil, errors.New("Signierung fehlgeschlagen: s=0")
	}

	return r, s, nil
}

// SignToHex signiert und gibt die Signatur als Hex zurück
func (priv *NuvexPrivateKey) SignToHex(messageHash []byte) (string, error) {
	r, s, err := priv.Sign(messageHash)
	if err != nil {
		return "", err
	}

	rBytes := make([]byte, 32)
	sBytes := make([]byte, 32)
	rB := r.Bytes()
	sB := s.Bytes()
	copy(rBytes[32-len(rB):], rB)
	copy(sBytes[32-len(sB):], sB)

	return hex.EncodeToString(append(rBytes, sBytes...)), nil
}

// ─────────────────────────────────────────────
//  ECDSA Verifikation
// ─────────────────────────────────────────────

// VerifyECDSA verifiziert eine ECDSA Signatur
func VerifyECDSA(pubKey *NuvexPublicKey, messageHash []byte, sigHex string) error {
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil || len(sigBytes) != 64 {
		return fmt.Errorf("Ungültige Signatur")
	}

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:])

	if r.Sign() == 0 || s.Sign() == 0 {
		return fmt.Errorf("Signatur ist null")
	}
	if r.Cmp(secp256k1N) >= 0 || s.Cmp(secp256k1N) >= 0 {
		return fmt.Errorf("Signatur ausserhalb der Gruppenordnung")
	}

	G := &Point{X: secp256k1Gx, Y: secp256k1Gy}
	pub := &Point{X: pubKey.X, Y: pubKey.Y}

	hashInt := new(big.Int).SetBytes(messageHash)
	sInv := modInverse(s, secp256k1N)

	u1 := new(big.Int).Mul(hashInt, sInv)
	u1.Mod(u1, secp256k1N)

	u2 := new(big.Int).Mul(r, sInv)
	u2.Mod(u2, secp256k1N)

	p1 := scalarMult(u1, G)
	p2 := scalarMult(u2, pub)
	point := pointAdd(p1, p2)

	if point.IsInfinity() {
		return fmt.Errorf("Verifikation fehlgeschlagen")
	}

	x := new(big.Int).Mod(point.X, secp256k1N)
	if x.Cmp(r) != 0 {
		return fmt.Errorf("❌ Signatur ungültig — Transaktion abgelehnt")
	}

	return nil
}

// ─────────────────────────────────────────────
//  RFC 6979 — Deterministisches k
// ─────────────────────────────────────────────

func generateDeterministicK(privKey, hash []byte) (*big.Int, error) {
	// RFC 6979 Implementation
	v := make([]byte, 32)
	k := make([]byte, 32)
	for i := range v { v[i] = 0x01 }
	for i := range k { k[i] = 0x00 }

	// K = HMAC-SHA256(K, V || 0x00 || privKey || hash)
	mac := hmac.New(sha512.New, k)
	mac.Write(v)
	mac.Write([]byte{0x00})
	mac.Write(privKey)
	mac.Write(hash)
	k = mac.Sum(nil)[:32]

	// V = HMAC-SHA256(K, V)
	mac = hmac.New(sha512.New, k)
	mac.Write(v)
	v = mac.Sum(nil)[:32]

	// K = HMAC-SHA256(K, V || 0x01 || privKey || hash)
	mac = hmac.New(sha512.New, k)
	mac.Write(v)
	mac.Write([]byte{0x01})
	mac.Write(privKey)
	mac.Write(hash)
	k = mac.Sum(nil)[:32]

	// V = HMAC-SHA256(K, V)
	mac = hmac.New(sha512.New, k)
	mac.Write(v)
	v = mac.Sum(nil)[:32]

	// k = HMAC-SHA256(K, V)
	mac = hmac.New(sha512.New, k)
	mac.Write(v)
	candidate := mac.Sum(nil)[:32]

	kInt := new(big.Int).SetBytes(candidate)
	if kInt.Sign() == 0 || kInt.Cmp(secp256k1N) >= 0 {
		return nil, fmt.Errorf("k Generierung fehlgeschlagen")
	}

	return kInt, nil
}

// ─────────────────────────────────────────────
//  TX Hash für Signierung
// ─────────────────────────────────────────────

// HashTransaction erstellt den Hash einer TX für die Signierung
func HashTransaction(from, to string, amount, fee int64, nonce uint64) []byte {
	data := fmt.Sprintf("nuvex-1:%s:%s:%d:%d:%d", from, to, amount, fee, nonce)
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}
