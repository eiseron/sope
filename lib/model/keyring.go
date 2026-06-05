package model

import (
	"errors"

	"github.com/getsops/sops/v3/age"
)

var ErrNeedUnlock = errors.New("no age identity in the keyring can decrypt this file")

type Keyring struct {
	identities age.ParsedIdentities
}

func NewKeyring() *Keyring {
	return &Keyring{}
}

func (k *Keyring) Unlock(secret string) error {
	return k.identities.Import(secret)
}

func (k *Keyring) Empty() bool {
	return len(k.identities) == 0
}
