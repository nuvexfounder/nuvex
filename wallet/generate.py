import hashlib
import secrets
from ecdsa import SigningKey, SECP256k1

def generate_wallet():
    private_key_bytes = secrets.token_bytes(32)
    sk = SigningKey.from_string(private_key_bytes, curve=SECP256k1)
    vk = sk.get_verifying_key()
    public_key_bytes = vk.to_string()
    sha256_hash = hashlib.sha256(public_key_bytes).digest()
    ripemd160 = hashlib.new('ripemd160')
    ripemd160.update(sha256_hash)
    address_bytes = ripemd160.digest()
    address = "nuvex1" + address_bytes.hex()[:38]
    return {
        "address": address,
        "private_key": private_key_bytes.hex(),
        "public_key": public_key_bytes.hex()
    }

def print_wallet(wallet, label, amount):
    print("\n" + "="*60)
    print(f"  {label} — {amount}")
    print("="*60)
    print(f"  Address:     {wallet['address']}")
    print(f"  Private Key: {wallet['private_key']}")
    print("="*60)

print("\n NUVEX WALLET GENERATOR v1.0")
print(" Schreibe alle Private Keys auf Papier!\n")

founder   = generate_wallet()
community = generate_wallet()
ecosystem = generate_wallet()
ico       = generate_wallet()
reserve   = generate_wallet()

print_wallet(founder,   "GRUENDER (Du)", "35,000,000 NVX")
print_wallet(community, "COMMUNITY",     "200,000,000 NVX")
print_wallet(ecosystem, "ECOSYSTEM",     "100,000,000 NVX")
print_wallet(ico,       "ICO",           "75,000,000 NVX")
print_wallet(reserve,   "RESERVE",       "15,000,000 NVX")

print("\n GENESIS ADRESSEN:")
print(f"  Gruender:  {founder['address']}")
print(f"  Community: {community['address']}")
print(f"  Ecosystem: {ecosystem['address']}")
print(f"  ICO:       {ico['address']}")
print(f"  Reserve:   {reserve['address']}")
print("\n Schreibe ALLES auf Papier. Jetzt.\n")
