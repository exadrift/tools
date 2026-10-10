package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

type Crypto struct {
	privKey *rsa.PrivateKey
}

func Deserialize(pkString string) (*Crypto, error) {
	pemBytes, err := base64.StdEncoding.DecodeString(pkString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse base64 encoded string: %w", err)
	}

	// Decode the first available PEM block found in the data
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("failed to decode valid PEM block containing 'PRIVATE KEY'")
	}

	// Parse the raw ASN.1 DER bytes back into a Go interface
	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	// Explicitly cast the interface to an RSA-specific Private Key pointer
	rsaKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}

	return &Crypto{
		privKey: rsaKey,
	}, nil
}

func New() (*Crypto, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	return &Crypto{
		privKey: privateKey,
	}, nil
}

func (c *Crypto) Serialize() (string, error) {
	// Convert the RSA key to a standardized PKCS#8 ASN.1 DER format byte slice
	privBytes, err := x509.MarshalPKCS8PrivateKey(c.privKey)
	if err != nil {
		return "", err
	}

	// Configure the PEM block structure
	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	}

	// Write the encoded text format directly to the open file
	pemBytes := pem.EncodeToMemory(pemBlock)
	return base64.StdEncoding.EncodeToString(pemBytes), nil
}

func (c *Crypto) Encrypt(plaintext string) (string, error) {
	// Encrypt the string text using the Public Key
	ciphertext, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		&c.privKey.PublicKey,
		[]byte(plaintext),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %w", err)
	}

	// For display purposes, encode the binary ciphertext to a readable Base64 string
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (c *Crypto) Decrypt(encrypted string) (string, error) {
	decodedCiphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("unable to parse base64 encoded string: %w", err)
	}

	decryptedBytes, err := rsa.DecryptOAEP(
		sha256.New(),
		rand.Reader,
		c.privKey,
		decodedCiphertext,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(decryptedBytes), nil
}
