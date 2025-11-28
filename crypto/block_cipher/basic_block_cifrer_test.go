package block_cipher

import (
	"bytes"
	"crypto/aes"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"
)

type md5Sum []byte

func (m md5Sum) String() string {
	hashString := hex.EncodeToString(m)
	return fmt.Sprintf("MD5: %s", hashString)
}

type test struct {
	dir       string
	plainText string
	size      int
	// Plain text MD5
	plainTextMD5 md5Sum
	// Encrypted MD5
	encryptWithOpensslMD5 md5Sum
	encryptWithGo         md5Sum
	//
	decryptWithOpensslMD5 md5Sum
	decryptWithGo         md5Sum
}

func (t test) plainTextFile() string {
	return filepath.Join(t.dir, t.plainText)
}

func (t test) encryptedFileWithOpenssl() string {
	return filepath.Join(t.dir, t.plainText+"encrypted_withopenssl.enc")
}

func (t test) decryptedFileWithOpenssl() string {
	return filepath.Join(t.dir, t.plainText+"decrypted_withopenssl.bin")
}

func (t test) encryptedFileWithGo() string {
	return filepath.Join(t.dir, t.plainText+"encrypted_withgo.enc")
}

func (t test) decryptedFileWithGo() string {
	return filepath.Join(t.dir, t.plainText+"decrypted_withgo.bin")
}

func encryptFileWithOpenSSL(t test, key string) (md5Sum, error) {
	// openssl aes-256-cbc -in encrypted_withGO.enc -out decrypted_withopenssl.bin -k password -pbkdf2

	app := "openssl"
	aes256CBC := "aes-256-cbc"
	oFile := t.encryptedFileWithOpenssl()
	inFile := t.plainTextFile()

	cmd := exec.Command(app, aes256CBC, "-in", inFile, "-out", oFile, "-k", key, "-pbkdf2")

	// Cattura stdout e stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("Errore durante l'esecuzione di openssl:", err)
		fmt.Println("stderr:", stderr.String())
		return nil, err
	}

	encryptMd5, err := calculateMD5(oFile)
	if err != nil {
		return nil, err
	}
	return encryptMd5, err
}

func decryptFileWithOpenSSL(t test, key string) (md5Sum, error) {
	// openssl aes-256-cbc -in encrypted_withGO.enc -out decrypted_withopenssl.bin -k password -pbkdf2
	app := "openssl"
	aes256CBC := "aes-256-cbc"
	inFile := t.encryptedFileWithGo()
	oFile := t.decryptedFileWithOpenssl()

	// Prepara il commando
	cmd := exec.Command(app, aes256CBC, "-d", "-in", inFile, "-out", oFile, "-k", key, "-pbkdf2")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Esegui il commando
	err := cmd.Run()
	if err != nil {
		fmt.Println("Errore durante l'esecuzione di openssl:", err)
		fmt.Println("stderr:", stderr.String())
		return nil, err
	}

	encryptMd5, err := calculateMD5(oFile)
	if err != nil {
		return nil, err
	}

	return encryptMd5, err
}

func encryptFileWithGo(t test, key string) (md5Sum, error) {
	inFile := t.plainTextFile()
	oFile := t.encryptedFileWithGo()
	// Open file to read
	r, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	// Create New file to write encrypted data
	w, err := os.Create(oFile)
	if err != nil {
		return nil, err
	}
	defer w.Close()

	EncryptStream(w, r, []byte(key))

	// Calculate MD5
	md5, err := calculateMD5(oFile)
	if err != nil {
		return nil, err
	}

	return md5, nil
}

func decryptFileWithGo(t test, key string) (md5Sum, error) {
	inFile := t.encryptedFileWithGo()
	oFile := t.decryptedFileWithGo()

	// Open file to read
	r, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	// Create New file to write decrypted data
	w, err := os.Create(oFile)
	if err != nil {
		return nil, err
	}
	defer w.Close()

	// Decrypt data
	DecryptStream(w, r, []byte(key))

	// Calculate MD5
	md5, err := calculateMD5(oFile)
	if err != nil {
		return nil, err
	}

	return md5, nil
}

// Funzione per creare un file con nome e dimensione specificati.
func CreateFileWithSize(filename string, size int) error {
	// Crea il file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("errore nella creazione del file: %w", err)
	}
	defer file.Close()

	// Buffer per scrivere dati casuali
	buffer := make([]byte, 4096) // buffer di 4KB
	bytesWritten := 0

	// Continua a scrivere dati casuali finché non si raggiunge la dimensione desiderata
	for bytesWritten < size {
		n, err := rand.Read(buffer)
		if err != nil {
			return fmt.Errorf("errore nella lettura dei dati casuali: %w", err)
		}

		// Se ci sono meno byte da scrivere rispetto alla dimensione del buffer
		if bytesWritten+n > size {
			n = size - bytesWritten
		}

		_, err = file.Write(buffer[:n]) // Scrivi i dati nel file
		if err != nil {
			return fmt.Errorf("errore nella scrittura nel file: %w", err)
		}

		bytesWritten += n
	}

	fmt.Printf("File %s creato con successo, dimensione: %d bytes\n", filename, size)
	return nil
}

func TestCreateFileWithSize(t *testing.T) {
	const pwgen = "s3cr3t"

	// Crea una directory temporanea nella directory corrente
	dir, err := os.MkdirTemp(".", "prefix-")
	if err != nil {
		fmt.Println("Errore nella creazione della directory temporanea:", err)
		return
	}

	// Stampa il percorso della directory temporanea
	fmt.Println("Directory temporanea creata nella directory corrente:", dir)

	// Pulizia: rimuovi la directory quando non serve più
	defer os.RemoveAll(dir)

	voidMd5 := []byte{0}

	definedTestList := []test{
		{dir, "test1_.bin", 0, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test2_.bin", 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test3_.bin", 2, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test4_5.bin", aes.BlockSize - 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test5_6.bin", aes.BlockSize, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test6_7.bin", aes.BlockSize + 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test7_7.bin", aes.BlockSize*3 - 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test8_8.bin", aes.BlockSize * 3, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test9_9.bin", aes.BlockSize*3 + 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test10_1024k-1.bin", 1024 - 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test11_1024k.bin", 1024, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test12_1024k+1.bin", 1024 + 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test13_1024k+16-1.bin", 1024 + aes.BlockSize - 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test14_1024k+16.bin", 1024 + aes.BlockSize, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test15_1024k+16+1.bin", 1024 + aes.BlockSize + 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test16_1m-1.bin", 1024*1024 - 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test17_1m.bin", 1024 * 1024, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test18_1m_1.bin", 1024*1024 + 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test19_1m+16-1.bin", 1024*1024 + aes.BlockSize - 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test20_1m+16.bin", 1024*1024 + aes.BlockSize, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test21_1m+16+1.bin", 1024*1024 + aes.BlockSize + 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test22_2m-1.bin", 1024*1024*2 - 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test23_2m.bin", 1024 * 1024 * 2, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test24_2m+1.bin", 1024*1024*2 + 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test25_2m+16-1.bin", 1024*1024*2 + aes.BlockSize - 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test26_2m+16.bin", 1024*1024*2 + aes.BlockSize, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
		{dir, "test27_2m+16+1.bin", 1024*1024*2 + aes.BlockSize + 1, voidMd5, voidMd5, voidMd5, voidMd5, voidMd5},
	}

	// Enable this additional test cases for make fuzzing test. It will create random test cases with random size
	additionalRandomTestCases := 0
	totalTestCase := len(definedTestList) + additionalRandomTestCases
	testList := make([]test, totalTestCase)
	// Copi definedTestList in testList
	copy(testList, definedTestList)

	// Aggiungi test random
	startIndex := len(definedTestList)

	for i := startIndex; i < totalTestCase; i++ {
		maxSize := big.NewInt(1024 * 1024)
		randomNumber, err := rand.Int(rand.Reader, maxSize)
		if err != nil {
			t.Fatalf("Errore: %v", err)
		}
		size := int(randomNumber.Int64())

		testStruct := test{
			dir:                   dir,
			plainText:             fmt.Sprintf("random_test_%d.bin", size),
			size:                  size,
			plainTextMD5:          voidMd5,
			encryptWithOpensslMD5: voidMd5,
			encryptWithGo:         voidMd5,
			decryptWithOpensslMD5: voidMd5,
			decryptWithGo:         voidMd5,
		}
		testList[i] = testStruct
	}

	for _, tc := range testList {
		path := path.Join(dir, tc.plainText)
		err := CreateFileWithSize(path, tc.size)
		if err != nil {
			t.Fatalf("Errore: %v", err)
		}

		// Plain Text and MD5
		plainMd5, err := calculateMD5(path)
		if err != nil {
			t.Fatalf("Errore: %v", err)
		}
		tc.plainTextMD5 = plainMd5

		// encrypt with openssl and MD5
		md5, err := encryptFileWithOpenSSL(tc, pwgen)
		if err != nil {
			t.Fatalf("error during encryption with openssl: %v", err)
		}
		tc.encryptWithOpensslMD5 = md5

		// encrypt with go and MD5
		md5, err = encryptFileWithGo(tc, pwgen)
		if err != nil {
			t.Fatalf("error during encryption with Golang: %v", err)
		}
		tc.encryptWithGo = md5

		// decrypt with openssl and MD5
		md5, err = decryptFileWithOpenSSL(tc, pwgen)
		if err != nil {
			t.Fatalf("error during decryption with openssl: %v", err)
		}
		tc.decryptWithOpensslMD5 = md5

		// decrypt with go and MD5
		md5, err = decryptFileWithGo(tc, pwgen)
		if err != nil {
			t.Fatalf("error during decryption with Golang: %v", err)
		}
		tc.decryptWithGo = md5

		println(tc.plainTextMD5.String(), "plainTextMD5")
		println(tc.decryptWithOpensslMD5.String(), "decryptWithOpensslMD5")
		println(tc.decryptWithGo.String(), "decryptWithGo MD5")

		if !bytes.Equal(tc.plainTextMD5, tc.decryptWithOpensslMD5) {
			t.Errorf("decrypt with openssl and MD5 not equal")
		}

		if !bytes.Equal(tc.plainTextMD5, tc.decryptWithGo) {
			t.Errorf("decrypt with go and MD5 not equal")
		}
	}
}

func calculateMD5(filename string) ([]byte, error) {
	// Apri il file
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Errore durante l'apertura del file:", err)
		return nil, err
	}
	defer file.Close()

	// Crea un nuovo oggetto hash MD5
	hash := md5.New()

	// Copia i dati del file nel calcolatore di hash a blocchi (chunk)
	if _, err := io.Copy(hash, file); err != nil {
		fmt.Println("Errore durante la lettura del file:", err)
		return nil, err
	}

	// Ottieni il risultato del calcolo dell'hash
	hashInBytes := hash.Sum(nil) // Restituisce il digest come byte

	return hashInBytes, nil
}
