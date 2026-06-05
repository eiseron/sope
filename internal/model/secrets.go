package model

import (
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
