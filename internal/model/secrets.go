package model

import (
	"time"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	"github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/stores/dotenv"
)

func (k *Keyring) DecryptFile(ciphertext []byte) ([]Entry, error) {
	plain, err := k.decrypt(ciphertext)
	if err != nil {
		return nil, err
	}
	return parseDotenv(plain), nil
}

func (k *Keyring) decrypt(ciphertext []byte) ([]byte, error) {
	store := dotenv.NewStore(&config.DotenvStoreConfig{})
	tree, err := store.LoadEncryptedFile(ciphertext)
	if err != nil {
		return nil, err
	}
	dataKey, err := decryptDataKey(&tree.Metadata, k.identities)
	if err != nil {
		return nil, err
	}
	if _, err := tree.Decrypt(dataKey, aes.NewCipher()); err != nil {
		return nil, err
	}
	return store.EmitPlainFile(tree.Branches)
}

func (k *Keyring) EncryptFile(ciphertext []byte, entries []Entry, now time.Time) ([]byte, error) {
	store := dotenv.NewStore(&config.DotenvStoreConfig{})
	tree, err := store.LoadEncryptedFile(ciphertext)
	if err != nil {
		return nil, err
	}
	dataKey, err := decryptDataKey(&tree.Metadata, k.identities)
	if err != nil {
		return nil, err
	}
	branches, err := store.LoadPlainFile(formatDotenv(entries))
	if err != nil {
		return nil, err
	}
	cipher := aes.NewCipher()
	newTree := sops.Tree{Branches: branches, Metadata: tree.Metadata}
	newTree.Metadata.LastModified = now.UTC()
	mac, err := newTree.Encrypt(dataKey, cipher)
	if err != nil {
		return nil, err
	}
	encryptedMac, err := cipher.Encrypt(mac, dataKey, newTree.Metadata.LastModified.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	newTree.Metadata.MessageAuthenticationCode = encryptedMac
	return store.EmitEncryptedFile(newTree)
}

func decryptDataKey(m *sops.Metadata, ids age.ParsedIdentities) ([]byte, error) {
	for _, group := range m.KeyGroups {
		for _, mk := range group {
			ak, ok := mk.(*age.MasterKey)
			if !ok {
				continue
			}
			ids.ApplyToMasterKey(ak)
			if dataKey, err := ak.Decrypt(); err == nil {
				return dataKey, nil
			}
		}
	}
	return nil, ErrNeedUnlock
}
