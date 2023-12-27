package utils

import (
	"os"
	"testing"
)

func TestNewEncryptor(t *testing.T) {
	os.Setenv("ENCRYPT_KEY", "DINUTHINDUWARA12")
	keyBytes := []byte(os.Getenv("ENCRYPT_KEY"))
	t.Log(keyBytes)
	_, retfun := Encryptor("../static/tt.txt", &keyBytes)
	err := retfun()
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewDecryptor(t *testing.T) {
	os.Setenv("ENCRYPT_KEY", "DINUTHINDUWARA12")
	keyBytes := []byte(os.Getenv("ENCRYPT_KEY"))
	t.Log(keyBytes)
	_, retfun := Decryptor("../static/tt.txt.crypted", &keyBytes)
	err := retfun()
	if err != nil {
		t.Fatal(err)
	}
}
