package server

import "crypto/rand"

const codeLength = 8

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func generateCode() (string, error) {
	code := make([]byte, codeLength)

	if _, err := rand.Read(code); err != nil {
		return "", err
	}

	for i := range code {
		code[i] = alphabet[code[i]%byte(len(alphabet))]
	}

	return string(code), nil
}
