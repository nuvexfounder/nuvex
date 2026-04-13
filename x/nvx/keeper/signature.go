package keeper

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

// TxPayload ist was signiert wird
type TxPayload struct {
	From   string
	To     string
	Amount int64
	Fee    int64
	Nonce  uint64
}

// Hash des Payloads generieren
func HashPayload(p TxPayload) []byte {
	raw := fmt.Sprintf("%s:%s:%d:%d:%d:nuvex-1",
		p.From, p.To, p.Amount, p.Fee, p.Nonce)
	hash := sha256.Sum256([]byte(raw))
	return hash[:]
}

// Signatur verifizieren
func VerifySignature(publicKeyHex string, payload TxPayload, signatureHex string) error {
	// Public Key dekodieren
	pubKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return fmt.Errorf("ungültiger public key: %w", err)
	}

	if len(pubKeyBytes) != 64 {
		return fmt.Errorf("public key muss 64 bytes sein, hat %d", len(pubKeyBytes))
	}

	// SECP256k1 Kurve (gleich wie Bitcoin)
	curve := elliptic.P256()
	x := new(big.Int).SetBytes(pubKeyBytes[:32])
	y := new(big.Int).SetBytes(pubKeyBytes[32:])

	pubKey := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}

	// Signatur dekodieren
	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return fmt.Errorf("ungültige signatur: %w", err)
	}

	if len(sigBytes) != 64 {
		return fmt.Errorf("signatur muss 64 bytes sein")
	}

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:])

	// Hash des Payloads
	hash := HashPayload(payload)

	// Signatur prüfen
	if !ecdsa.Verify(pubKey, hash, r, s) {
		return fmt.Errorf("❌ UNGÜLTIGE SIGNATUR — Transaktion abgelehnt")
	}

	return nil
}

// Adresse aus Public Key ableiten (zur Verifikation)
func AddressFromPublicKey(publicKeyHex string) (string, error) {
	pubKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return "", err
	}
	sha256Hash := sha256.Sum256(pubKeyBytes)
	h := sha256.Sum256(sha256Hash[:])
	return "nuvex1" + hex.EncodeToString(h[:19]), nil
}
