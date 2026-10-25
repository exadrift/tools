package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
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

func (c *Crypto) Encrypt(plaintext []byte) (string, error) {
	// 1. Generate a random 256-bit (32-byte) AES key
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return "", fmt.Errorf("failed to generate random AES key: %w", err)
	}

	// 2. Initialize the AES-GCM cipher block
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher block: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM block: %w", err)
	}

	// 3. Generate a unique nonce (Initialization Vector) for GCM
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate GCM nonce: %w", err)
	}

	// 4. Encrypt the data payload using AES-GCM (prepends nonce to the resulting ciphertext)
	aesCiphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	// 5. Encrypt the AES key using the RSA Public Key (OAEP with SHA-256)
	label := []byte("hybrid-encryption")
	rsaEncryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &c.privKey.PublicKey, aesKey, label)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt AES key with RSA: %w", err)
	}

	// 6. Pack everything into a single slice: [Length prefix (4B)] + [RSA Key] + [AES Ciphertext]
	rsaKeyLen := uint32(len(rsaEncryptedKey))
	finalPayload := make([]byte, 4+rsaKeyLen+uint32(len(aesCiphertext)))

	// Write the 4-byte length prefix (BigEndian ensures architecture independence)
	binary.BigEndian.PutUint32(finalPayload[0:4], rsaKeyLen)

	// Copy the RSA-encrypted key and the ciphertext into place
	copy(finalPayload[4:4+rsaKeyLen], rsaEncryptedKey)
	copy(finalPayload[4+rsaKeyLen:], aesCiphertext)

	// For display purposes, encode the binary ciphertext to a readable Base64 string
	return base64.StdEncoding.EncodeToString(finalPayload), nil
}

func (c *Crypto) Decrypt(encrypted []byte) ([]byte, error) {
	decodedCiphertext, err := base64.StdEncoding.DecodeString(string(encrypted))
	if err != nil {
		return nil, fmt.Errorf("unable to parse base64 encoded string: %w", err)
	}

	// 1. Ensure the payload has at least the minimum length header
	if len(decodedCiphertext) < 4 {
		return nil, fmt.Errorf("payload is too short to contain a valid header")
	}

	// 2. Read the RSA key length prefix
	rsaKeyLen := binary.BigEndian.Uint32(decodedCiphertext[0:4])

	// 3. Verify total bounds check to avoid slice panic bounds errors
	if uint32(len(decodedCiphertext)) < 4+rsaKeyLen {
		return nil, fmt.Errorf("payload data is truncated or corrupt")
	}

	// 4. Extract the RSA-encrypted key slice and the remaining AES ciphertext
	rsaEncryptedKey := decodedCiphertext[4 : 4+rsaKeyLen]
	aesCiphertext := decodedCiphertext[4+rsaKeyLen:]

	// 5. Decrypt the AES symmetric key using the RSA Private Key
	label := []byte("hybrid-encryption")
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, c.privKey, rsaEncryptedKey, label)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt envelope AES key: %w", err)
	}

	// 6. Initialize the AES-GCM cipher block using the extracted key
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher block: %w", err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM block: %w", err)
	}

	// 7. Extract the GCM nonce (which was prepended during encryption step 4)
	nonceSize := aesGCM.NonceSize()
	if len(aesCiphertext) < nonceSize {
		return nil, fmt.Errorf("inner ciphertext too short to contain nonce")
	}
	nonce, actualCiphertext := aesCiphertext[:nonceSize], aesCiphertext[nonceSize:]

	// 8. Decrypt the actual data payload back into plaintext
	plaintext, err := aesGCM.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data payload: message authentication failed")
	}

	return plaintext, nil
}
