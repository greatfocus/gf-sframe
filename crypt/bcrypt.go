package crypt

import (
	"golang.org/x/crypto/bcrypt"
)

// NewHash encrypts value
func NewHash(value []byte) (string, error) {
	// Use GenerateFromPassword to hash & salt pwd
	// MinCost is just an integer constant provided by the bcrypt
	// package along with DefaultCost & MaxCost.
	// The cost can be any value you want provided it isn't lower
	// than the MinCost (4)
	hash, err := bcrypt.GenerateFromPassword(value, bcrypt.MinCost)
	if err != nil {
		return "", err
	}
	// GenerateFromPassword returns a byte slice so we need to
	// convert the bytes to a string and return it
	return string(hash), nil
}

// CompareHash Compare string
func CompareHash(hashed string, plain []byte) (bool, error) {
	// Since we'll be getting the hashed string from the DB it
	// will be a string so we'll need to convert it to a byte slice
	byteHash := []byte(hashed)
	err := bcrypt.CompareHashAndPassword(byteHash, plain)
	if err != nil {
		return false, err
	}

	return true, nil
}
