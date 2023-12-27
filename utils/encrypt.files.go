package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type CryptFile struct {
	FSize       int64
	Fname       string
	CryperdSize int64
	fpath       string
	key         *[]byte
	Task        string
}

func Encryptor(fpath string, key *[]byte) (*CryptFile, func() error) {
	// Get file size
	fileInfo, err := os.Stat(fpath)
	if err != nil || os.IsNotExist(err) {
		fmt.Printf("Error getting file information: %v\n", err)
		return nil, nil
	}
	obj := &CryptFile{
		fpath: fpath,
		key:   key,
		FSize: fileInfo.Size(),
		Fname: filepath.Base(fpath),
	}
	return obj, obj.encrypt
}

func Decryptor(fpath string, key *[]byte) (*CryptFile, func() error) {
	// Get file size
	fileInfo, err := os.Stat(fpath)
	if err != nil || os.IsNotExist(err) {
		fmt.Printf("Error getting file information: %v\n", err)
		return nil, nil
	}
	obj := &CryptFile{
		fpath: fpath,
		key:   key,
		FSize: fileInfo.Size(),
		Fname: filepath.Base(fpath),
	}
	return obj, obj.decrypt
}

func (cr *CryptFile) encrypt() error {
	// check file already encrypted
	if strings.HasSuffix(cr.fpath, ".crypted") {
		return nil
	}

	// Open input file
	input, err := os.Open(cr.fpath)
	if err != nil {
		return err
	}
	defer input.Close()

	// Create AES cipher block
	block, err := aes.NewCipher(*cr.key)
	if err != nil {
		return err
	}

	// Create AES cipher mode
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}
	stream := cipher.NewCFBEncrypter(block, iv)

	outputFile := cr.fpath + ".crypted"

	// Create output file
	output, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer output.Close()

	// Write IV to output file
	if _, err := output.Write(iv); err != nil {
		return err
	}

	// Encrypt and write each block
	buffer := make([]byte, 4096) // Adjust block size as needed :)
	cr.CryperdSize = 0
	cr.Task = "Encrypting"
	for {
		n, err := input.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		stream.XORKeyStream(buffer[:n], buffer[:n])
		cr.CryperdSize += int64(n) // Update Encrypted Chunk Size
		if _, err := output.Write(buffer[:n]); err != nil {
			return err
		}
	}
	cr.Task = ""

	return nil
}

func (cr *CryptFile) decrypt() error {
	// check file is not encrypted
	if !strings.HasSuffix(cr.fpath, ".crypted") {
		return nil
	}
	// Open input file
	input, err := os.Open(cr.fpath)
	if err != nil {
		return err
	}
	defer input.Close()

	// Read IV from input file
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(input, iv); err != nil {
		return err
	}

	// Create AES cipher block
	block, err := aes.NewCipher(*cr.key)
	if err != nil {
		return err
	}

	// Create AES cipher mode
	stream := cipher.NewCFBDecrypter(block, iv)

	outputFile := strings.TrimSuffix(cr.fpath, ".crypted")

	// Create output file
	output, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer output.Close()

	// Decrypt and write each block
	buffer := make([]byte, 4096) // Adjust block size as needed
	cr.CryperdSize = 0
	cr.Task = "Decrypting"
	for {
		n, err := input.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		stream.XORKeyStream(buffer[:n], buffer[:n])
		cr.CryperdSize += int64(n)
		if _, err := output.Write(buffer[:n]); err != nil {
			return err
		}
	}
	cr.Task = ""

	return nil
}

func (cr *CryptFile) Percentage() int {
	return int((cr.CryperdSize / cr.FSize) * 100)
}
