// Package keystore is the personal-mode secret store (#28), implementing the
// key-custody decision in #25.
//
// It persists only secrets — the World seed phrase and the identity passphrase —
// encrypted at rest (XChaCha20-Poly1305 under an Argon2id-derived key). Private
// keys are NEVER written to disk: they are re-derived in memory at unlock via the
// SDK (OpenWorld / OpenIdentity), exactly reproduced from the same secrets.
//
// On a multi-user/remote host you would back this with the OS keychain
// (Keychain / libsecret / DPAPI); that backend can slot in behind the same API
// without changing callers. The portable encrypted-file backend here is the
// default so the node runs and is testable everywhere.
package keystore

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"

	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/implementations/go/identity"
)

const (
	fileName        = "keystore.json"
	envelopeVersion = 1
	saltLen         = 16

	// Argon2id parameters used for new keystores. Existing keystores carry their
	// own parameters in the envelope so they keep unlocking after a change here.
	argonTime    = 1
	argonMemory  = 64 * 1024 // KiB → 64 MiB
	argonThreads = 4
)

var (
	// ErrExists is returned by Init when a keystore already exists.
	ErrExists = errors.New("keystore: already exists")
	// ErrNotExist is returned by Open when no keystore is present.
	ErrNotExist = errors.New("keystore: does not exist")
	// ErrBadPassphrase is returned when the unlock passphrase is wrong or the
	// keystore is corrupt (the two are indistinguishable by design).
	ErrBadPassphrase = errors.New("keystore: incorrect passphrase or corrupt keystore")
)

// Secrets are the values a keystore protects. Nothing else is persisted.
type Secrets struct {
	WorldSeed          string
	IdentityPassphrase string
}

// Path returns the conventional keystore location within a data directory.
func Path(dataDir string) string { return filepath.Join(dataDir, fileName) }

// Exists reports whether a keystore file is present at path.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Keystore is an unlocked keystore: it holds the decrypted secrets and the
// in-memory keys derived from them.
type Keystore struct {
	path   string
	master []byte // Argon2id-derived encryption key (zeroed by Lock)
	salt   []byte
	aTime  uint32
	aMem   uint32
	aProcs uint8

	secrets Secrets
	world   derive.World
	ident   identity.Identity
}

// envelope is the on-disk format. The block of secrets is encrypted; everything
// else is public KDF/AEAD parameters needed to unlock.
type envelope struct {
	Version int    `json:"version"`
	KDF     string `json:"kdf"`
	Salt    []byte `json:"salt"`
	Time    uint32 `json:"argon_time"`
	Memory  uint32 `json:"argon_memory"`
	Threads uint8  `json:"argon_threads"`
	Nonce   []byte `json:"nonce"`
	Cipher  []byte `json:"ciphertext"`
}

type secretsFile struct {
	WorldSeed          string `json:"world_seed"`
	IdentityPassphrase string `json:"identity_passphrase"`
}

// Init creates a new keystore at path, encrypted under unlock, storing the given
// secrets. It errors if a keystore already exists there.
func Init(path, unlock string, s Secrets) (*Keystore, error) {
	if s.WorldSeed == "" {
		return nil, errors.New("keystore: world seed required to initialize")
	}
	if Exists(path) {
		return nil, ErrExists
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	k := &Keystore{
		path:    path,
		salt:    salt,
		aTime:   argonTime,
		aMem:    argonMemory,
		aProcs:  argonThreads,
		secrets: s,
	}
	k.master = argon2.IDKey([]byte(unlock), salt, k.aTime, k.aMem, k.aProcs, chacha20poly1305.KeySize)
	k.derive()
	if err := k.save(); err != nil {
		return nil, err
	}
	return k, nil
}

// Open unlocks the keystore at path with unlock.
func Open(path, unlock string) (*Keystore, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("keystore: parse: %w", err)
	}
	if env.Version != envelopeVersion {
		return nil, fmt.Errorf("keystore: unsupported version %d", env.Version)
	}
	master := argon2.IDKey([]byte(unlock), env.Salt, env.Time, env.Memory, env.Threads, chacha20poly1305.KeySize)
	plain, err := decrypt(master, env.Nonce, env.Cipher)
	if err != nil {
		return nil, ErrBadPassphrase
	}
	var sf secretsFile
	if err := json.Unmarshal(plain, &sf); err != nil {
		return nil, ErrBadPassphrase
	}
	k := &Keystore{
		path:    path,
		master:  master,
		salt:    env.Salt,
		aTime:   env.Time,
		aMem:    env.Memory,
		aProcs:  env.Threads,
		secrets: Secrets{WorldSeed: sf.WorldSeed, IdentityPassphrase: sf.IdentityPassphrase},
	}
	k.derive()
	return k, nil
}

// World returns the unlocked World.
func (k *Keystore) World() derive.World { return k.world }

// Identity returns the unlocked identity (including its in-memory private keys).
func (k *Keystore) Identity() identity.Identity { return k.ident }

// RotateIdentity replaces the identity passphrase — yielding a new identity —
// and re-encrypts the keystore. The previous identity remains derivable from the
// old passphrase and revocable via an identity.burn (see identity-burn-rfc.md).
func (k *Keystore) RotateIdentity(newPassphrase string) error {
	k.secrets.IdentityPassphrase = newPassphrase
	k.derive()
	return k.save()
}

// BurnMaterial returns the current identity's root private keys — the bytes an
// identity.burn block reveals (revealed_ml_dsa, revealed_kyber).
func (k *Keystore) BurnMaterial() (mldsaPriv, kyberPriv []byte, err error) {
	if mldsaPriv, err = marshalBinary(k.ident.MLDSA.Private); err != nil {
		return nil, nil, err
	}
	if kyberPriv, err = marshalBinary(k.ident.Kyber.Private); err != nil {
		return nil, nil, err
	}
	return mldsaPriv, kyberPriv, nil
}

// Lock zeroizes the in-memory encryption key and drops the secrets. The derived
// private keys are managed by the SDK/CIRCL and reclaimed by the GC; Go offers
// no guaranteed zeroization for them.
func (k *Keystore) Lock() {
	for i := range k.master {
		k.master[i] = 0
	}
	k.secrets = Secrets{}
}

func (k *Keystore) derive() {
	k.world = derive.OpenWorld(k.secrets.WorldSeed)
	k.ident = identity.OpenIdentity(k.world, k.secrets.IdentityPassphrase)
}

func (k *Keystore) save() error {
	plain, err := json.Marshal(secretsFile{
		WorldSeed:          k.secrets.WorldSeed,
		IdentityPassphrase: k.secrets.IdentityPassphrase,
	})
	if err != nil {
		return err
	}
	nonce, ct, err := encrypt(k.master, plain)
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(envelope{
		Version: envelopeVersion,
		KDF:     "argon2id",
		Salt:    k.salt,
		Time:    k.aTime,
		Memory:  k.aMem,
		Threads: k.aProcs,
		Nonce:   nonce,
		Cipher:  ct,
	}, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(k.path, out)
}

func encrypt(key, plain []byte) (nonce, ct []byte, err error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return nonce, aead.Seal(nil, nonce, plain, nil), nil
}

func decrypt(key, nonce, ct []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	return aead.Open(nil, nonce, ct, nil)
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".keystore-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Rename(name, path)
}

type binaryMarshaler interface{ MarshalBinary() ([]byte, error) }

func marshalBinary(v any) ([]byte, error) {
	m, ok := v.(binaryMarshaler)
	if !ok {
		return nil, fmt.Errorf("keystore: key does not support MarshalBinary (%T)", v)
	}
	return m.MarshalBinary()
}
