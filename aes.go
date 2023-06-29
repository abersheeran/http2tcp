package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

func GenerateKey(str string) []byte {
	sha256 := sha256.New()
	sha256.Write([]byte(str))
	return sha256.Sum(nil)
}

func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("pkcs7UnPadding error: data is empty")
	}
	unPadding := int(data[length-1])
	return data[:(length - unPadding)], nil
}

func aes_cbc_encrypt(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	encryptBytes := pkcs7Padding(data, blockSize)
	crypted := make([]byte, len(encryptBytes))
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	blockMode.CryptBlocks(crypted, encryptBytes)
	return crypted, nil
}

func aes_cbc_decrypt(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	plaintext := make([]byte, len(data))
	blockMode.CryptBlocks(plaintext, data)
	plaintext, err = pkcs7UnPadding(plaintext)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func EncryptAndBase64(data string, key []byte) (string, error) {
	res, err := aes_cbc_encrypt([]byte(data), key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(res), nil
}

func DecryptFromBase64(data string, key []byte) (string, error) {
	dataByte, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	res, err := aes_cbc_decrypt(dataByte, key)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func EncryptStream(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	crypted := gcm.Seal(
		data[:0], []byte(strings.Repeat("9", gcm.NonceSize())),
		data, []byte("http2tcp"),
	)
	return crypted, nil
}

func DecryptStream(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	decrypted, err := gcm.Open(
		nil, []byte(strings.Repeat("9", gcm.NonceSize())),
		data, []byte("http2tcp"),
	)
	if err != nil {
		return nil, err
	}
	return decrypted, nil
}
