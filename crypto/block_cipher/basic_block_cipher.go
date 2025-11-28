package block_cipher

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	bufferSize = 1024 * 1024 // 1MB

	// Encryption header constants
	headerSize  = 16
	saltSize    = 8
	magicHeader = "Salted__"

	// PBKDF2 constants
	pbkdf2Iterations = 10000
	aes256KeySize    = 32

	// IV offset in derived key
	ivOffset    = 32
	ivEndOffset = 48
)

func readAndValidateHeader(inputFile io.Reader) ([]byte, error) {
	header := make([]byte, headerSize)
	_, err := io.ReadFull(inputFile, header)
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	if string(header[:saltSize]) != magicHeader {
		return nil, errors.New("invalid file format")
	}

	return header[saltSize:headerSize], nil
}

func deriveKeyAndIV(password, salt []byte) ([]byte, []byte) {
	keyIv := pbkdf2.Key(password, salt, pbkdf2Iterations, aes256KeySize+aes.BlockSize, sha256.New)
	key := keyIv[:aes256KeySize]
	iv := keyIv[ivOffset : ivOffset+aes.BlockSize]
	return key, iv
}

func removePKCS7Padding(data []byte, bytesRead int) []byte {
	if bytesRead == 0 {
		return data
	}
	paddingLength := int(data[bytesRead-1])
	if paddingLength > bytesRead || paddingLength > aes.BlockSize {
		return data
	}
	return data[:bytesRead-paddingLength]
}

// processDecryptionBlock handles decryption and writing of a single block
func processDecryptionBlock(
	outputFile io.Writer, mode cipher.BlockMode,
	encryptedBuffer []byte, bytesRead int,
	isLastBlock bool, previousDecryptedData []byte,
) ([]byte, error) {

	currentDecrypted := make([]byte, bytesRead)
	mode.CryptBlocks(currentDecrypted, encryptedBuffer[:bytesRead])

	if len(previousDecryptedData) > 0 {
		if _, err := outputFile.Write(previousDecryptedData); err != nil {
			return nil, fmt.Errorf("failed to write decrypted block: %w", err)
		}
	}

	if isLastBlock {
		finalData := removePKCS7Padding(currentDecrypted, bytesRead)
		if _, err := outputFile.Write(finalData); err != nil {
			return nil, fmt.Errorf("failed to write final block: %w", err)
		}
		return nil, nil // Signal completion
	}

	nextPreviousData := make([]byte, len(currentDecrypted))
	copy(nextPreviousData, currentDecrypted)
	return nextPreviousData, nil
}

func handleEndOfFile(outputFile io.Writer, previousDecryptedData []byte) error {
	if len(previousDecryptedData) > 0 {
		finalData := removePKCS7Padding(previousDecryptedData, len(previousDecryptedData))
		if _, err := outputFile.Write(finalData); err != nil {
			return fmt.Errorf("failed to write final block: %w", err)
		}
	}
	return nil
}

func DecryptStream(outputFile io.Writer, inputFile io.Reader, password []byte) error {
	salt, err := readAndValidateHeader(inputFile)
	if err != nil {
		return err
	}

	key, iv := deriveKeyAndIV(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	encryptedBuffer := make([]byte, bufferSize)
	var previousDecryptedData []byte

	for {
		bytesRead, readErr := io.ReadFull(inputFile, encryptedBuffer)
		isEOF := readErr == io.EOF || readErr == io.ErrUnexpectedEOF
		isLastBlock := bytesRead < bufferSize || isEOF

		if bytesRead == 0 {
			return handleEndOfFile(outputFile, previousDecryptedData)
		}

		nextPreviousData, err := processDecryptionBlock(outputFile, mode, encryptedBuffer, bytesRead, isLastBlock, previousDecryptedData)
		if err != nil {
			return err
		}

		if isLastBlock {
			return nil // Processing complete
		}

		previousDecryptedData = nextPreviousData

		if readErr != nil && !isEOF {
			return fmt.Errorf("failed to read encrypted data: %w", readErr)
		}
	}
}

func writeEncryptedHeader(w io.Writer) ([]byte, error) {
	salt := make([]byte, saltSize)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		return nil, fmt.Errorf("error generating salt: %w", err)
	}

	_, err = w.Write([]byte(magicHeader))
	if err != nil {
		return nil, fmt.Errorf("error writing header to file: %w", err)
	}

	_, err = w.Write(salt)
	if err != nil {
		return nil, fmt.Errorf("error writing salt to file: %w", err)
	}

	return salt, nil
}

func setupEncryption(password, salt []byte) (cipher.BlockMode, error) {
	key, iv := deriveKeyAndIV(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("error creating AES cipher: %w", err)
	}

	return cipher.NewCBCEncrypter(block, iv), nil
}

func writeEncryptedBlock(w io.Writer, cbc cipher.BlockMode, data []byte) error {
	encBlock := make([]byte, len(data))
	cbc.CryptBlocks(encBlock, data)

	_, err := w.Write(encBlock)
	if err != nil {
		return fmt.Errorf("error writing encrypted block to file: %w", err)
	}

	return nil
}

func processFinalBlock(w io.Writer, cbc cipher.BlockMode, data []byte, bytesRead int, hasWrittenData bool) error {
	if bytesRead == 0 {
		if !hasWrittenData {
			paddedBlock := applyPKCS7Padding(data[:0])
			return writeEncryptedBlock(w, cbc, paddedBlock)
		} else {
			padding := bytes.Repeat([]byte{byte(aes.BlockSize)}, aes.BlockSize)
			return writeEncryptedBlock(w, cbc, padding)
		}
	}

	paddedBlock := applyPKCS7Padding(data[:bytesRead])
	return writeEncryptedBlock(w, cbc, paddedBlock)
}

func EncryptStream(w io.Writer, r io.Reader, password []byte) error {
	salt, err := writeEncryptedHeader(w)
	if err != nil {
		return err
	}

	cbc, err := setupEncryption(password, salt)
	if err != nil {
		return err
	}

	readBuffer := make([]byte, bufferSize)
	hasWrittenData := false

	for {
		bytesRead, readErr := io.ReadFull(r, readBuffer)

		isEOF := readErr == io.EOF || readErr == io.ErrUnexpectedEOF
		isLastBlock := bytesRead < bufferSize || isEOF

		if isLastBlock {
			// Process the final block with proper padding
			err = processFinalBlock(w, cbc, readBuffer, bytesRead, hasWrittenData)
			if err != nil {
				return err
			}
			break
		}

		err = writeEncryptedBlock(w, cbc, readBuffer[:bytesRead])
		if err != nil {
			return err
		}

		hasWrittenData = true

		if readErr != nil && !isEOF {
			return fmt.Errorf("failed to read input data: %w", readErr)
		}
	}

	return nil
}

func applyPKCS7Padding(data []byte) []byte {
	padding := aes.BlockSize - len(data)%aes.BlockSize
	paddingBytes := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, paddingBytes...)
}
