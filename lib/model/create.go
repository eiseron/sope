package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/aes"
	sopsage "github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/config"
	"github.com/getsops/sops/v3/keyservice"
	"github.com/getsops/sops/v3/stores/dotenv"
	"github.com/getsops/sops/v3/version"
)

func CreateFile(recipients []string, entries []Entry, now time.Time) ([]byte, error) {
	if len(recipients) == 0 {
		return nil, fmt.Errorf("no age recipients to encrypt to")
	}
	masterKeys, err := sopsage.MasterKeysFromRecipients(strings.Join(recipients, ","))
	if err != nil {
		return nil, err
	}
	group := make(sops.KeyGroup, 0, len(masterKeys))
	for _, mk := range masterKeys {
		group = append(group, mk)
	}
	store := dotenv.NewStore(&config.DotenvStoreConfig{})
	branches, err := store.LoadPlainFile(formatDotenv(entries))
	if err != nil {
		return nil, err
	}
	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			KeyGroups:    []sops.KeyGroup{group},
			Version:      version.Version,
			LastModified: now.UTC(),
		},
	}
	dataKey, errs := tree.GenerateDataKeyWithKeyServices([]keyservice.KeyServiceClient{keyservice.NewLocalClient()})
	if len(errs) > 0 {
		return nil, errs[0]
	}
	cipher := aes.NewCipher()
	mac, err := tree.Encrypt(dataKey, cipher)
	if err != nil {
		return nil, err
	}
	encryptedMac, err := cipher.Encrypt(mac, dataKey, tree.Metadata.LastModified.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	tree.Metadata.MessageAuthenticationCode = encryptedMac
	return store.EmitEncryptedFile(tree)
}
