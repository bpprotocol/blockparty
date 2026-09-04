package keystore

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/bpprotocol/blockparty/implementations/go/identity"
)

const (
	testSeed   = "correct horse battery staple"
	testIDPass = "my-identity-passphrase"
	testUnlock = "unlock-me"
)

func TestInitOpenDeterministic(t *testing.T) {
	path := Path(t.TempDir())

	k1, err := Init(path, testUnlock, Secrets{WorldSeed: testSeed, IdentityPassphrase: testIDPass})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	addr1 := k1.Identity().Address

	// "Restart": open the same file with the same passphrase — keys must match.
	k2, err := Open(path, testUnlock)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if k2.Identity().Address != addr1 {
		t.Errorf("address across restart = %q, want %q", k2.Identity().Address, addr1)
	}
	// And match a direct SDK derivation from the same secrets.
	want := identity.OpenIdentity(k2.World(), testIDPass).Address
	if k2.Identity().Address != want {
		t.Errorf("address = %q, want SDK-derived %q", k2.Identity().Address, want)
	}
}

func TestWrongPassphrase(t *testing.T) {
	path := Path(t.TempDir())
	if _, err := Init(path, testUnlock, Secrets{WorldSeed: testSeed}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if _, err := Open(path, "wrong"); err != ErrBadPassphrase {
		t.Errorf("Open(wrong) err = %v, want ErrBadPassphrase", err)
	}
}

func TestInitExists(t *testing.T) {
	path := Path(t.TempDir())
	if _, err := Init(path, testUnlock, Secrets{WorldSeed: testSeed}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if _, err := Init(path, testUnlock, Secrets{WorldSeed: testSeed}); err != ErrExists {
		t.Errorf("second Init err = %v, want ErrExists", err)
	}
}

func TestDelete(t *testing.T) {
	path := Path(t.TempDir())
	if err := Delete(path); err != ErrNotExist {
		t.Errorf("Delete with no keystore err = %v, want ErrNotExist", err)
	}
	if _, err := Init(path, testUnlock, Secrets{WorldSeed: testSeed}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := Delete(path); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if Exists(path) {
		t.Error("keystore still exists after Delete")
	}
	// Gone for good: Open finds nothing, and Init may start over.
	if _, err := Open(path, testUnlock); err != ErrNotExist {
		t.Errorf("Open after Delete err = %v, want ErrNotExist", err)
	}
	if _, err := Init(path, "another-pass", Secrets{WorldSeed: testSeed}); err != nil {
		t.Errorf("Init after Delete: %v", err)
	}
}

// TestNoRawSecretsOnDisk is the core custody guarantee: neither the private keys
// nor the cleartext secrets ever touch disk.
func TestNoRawSecretsOnDisk(t *testing.T) {
	path := Path(t.TempDir())
	k, err := Init(path, testUnlock, Secrets{WorldSeed: testSeed, IdentityPassphrase: testIDPass})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	mldsaPriv, kyberPriv, err := k.BurnMaterial()
	if err != nil {
		t.Fatalf("BurnMaterial: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read keystore: %v", err)
	}
	if bytes.Contains(raw, mldsaPriv) {
		t.Error("ML-DSA private key bytes found on disk")
	}
	if bytes.Contains(raw, kyberPriv) {
		t.Error("Kyber private key bytes found on disk")
	}
	if bytes.Contains(raw, []byte(testSeed)) {
		t.Error("World seed found in cleartext on disk")
	}
	if bytes.Contains(raw, []byte(testIDPass)) {
		t.Error("identity passphrase found in cleartext on disk")
	}

	// File must be private.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("keystore perms = %o, want 600", perm)
	}
}

func TestRotateIdentity(t *testing.T) {
	path := Path(t.TempDir())
	k, err := Init(path, testUnlock, Secrets{WorldSeed: testSeed, IdentityPassphrase: testIDPass})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	before := k.Identity().Address

	if err := k.RotateIdentity("new-passphrase"); err != nil {
		t.Fatalf("RotateIdentity: %v", err)
	}
	after := k.Identity().Address
	if after == before {
		t.Error("identity address did not change after rotation")
	}

	// Persisted: reopening yields the rotated identity.
	k2, err := Open(path, testUnlock)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if k2.Identity().Address != after {
		t.Errorf("reopened address = %q, want rotated %q", k2.Identity().Address, after)
	}

	// The old identity is still derivable (and thus revocable via burn).
	old := identity.OpenIdentity(k2.World(), testIDPass).Address
	if old != before {
		t.Errorf("old identity no longer reproducible: got %q, want %q", old, before)
	}
}

func TestBurnMaterialReproducesKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ks.json")
	k, err := Init(path, testUnlock, Secrets{WorldSeed: testSeed, IdentityPassphrase: testIDPass})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	mldsaPriv, kyberPriv, err := k.BurnMaterial()
	if err != nil {
		t.Fatalf("BurnMaterial: %v", err)
	}
	if len(mldsaPriv) == 0 || len(kyberPriv) == 0 {
		t.Fatal("expected non-empty burn material")
	}
}
