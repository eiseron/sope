package model

import "filippo.io/age"

type Identity struct {
	Secret    string
	Recipient string
}

func GenerateIdentity() (Identity, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return Identity{}, err
	}
	return Identity{
		Secret:    id.String(),
		Recipient: id.Recipient().String(),
	}, nil
}
