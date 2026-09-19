package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// EncryptAESGCM cifra los datos con AES-256-GCM y antepone el nonce de 12 bytes.
func EncryptAESGCM(plainData []byte, secretKey []byte) ([]byte, error) {
	if len(secretKey) != 32 {
		return nil, errors.New("clave AES debe ser de 32 bytes (AES-256)")
	}
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	cipherText := gcm.Seal(nonce, nonce, plainData, nil)
	return cipherText, nil
}

// DecryptAESGCM extrae el nonce y descifra los bytes en memoria.
func DecryptAESGCM(cipherText []byte, secretKey []byte) ([]byte, error) {
	if len(secretKey) != 32 {
		return nil, errors.New("clave AES debe ser de 32 bytes (AES-256)")
	}
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return nil, errors.New("ciphertext demasiado corto")
	}
	nonce, actualCipher := cipherText[:nonceSize], cipherText[nonceSize:]
	return gcm.Open(nil, nonce, actualCipher, nil)
}
