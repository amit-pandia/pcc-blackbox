package main

import (
	"fmt"
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

func testKMKeys(t *testing.T) {
	fmt.Printf("\n\nKEYS:\n")
	var (
		key pcc.SecurityKey
		err error
	)

	alias := "key-bb-test"

	pcc.Pcc.DeleteKey(alias) // delete if exist
	fileName := fmt.Sprintf("%s.pem", alias)
	fmt.Println(fmt.Sprintf("generating the key %s %s", alias, fileName))
	cmd := exec.Command("/usr/bin/openssl", "genrsa", "-out", fileName, "2048")
	if err = cmd.Run(); err != nil {
		t.Fatal(err)
	}

	fmt.Println("uploading the key", alias)
	if key, err = pcc.Pcc.UploadKey(fileName, alias, pcc.PRIVATE_KEY, ""); err == nil { // TODO check if the key already exist
		fmt.Println("Added the key", key)
	} else {
		t.Fatal(err)
	}
	fmt.Println("uploaded the key", key.Id, key.Alias)

	defer func() {
		fmt.Println("deleting the key", key.Id, key.Alias)
		pcc.Pcc.DeleteKey(key.Id) // delete at the end
	}()

	fmt.Println("comparing the content for the key", key)
	if content, err := pcc.Pcc.DownloadKey(alias, pcc.PRIVATE_KEY); err == nil { // compare the content
		readFileAndCompare(t, content, fileName)
	} else {
		t.Fatal(err)
	}

	fmt.Println(fmt.Sprintf("looking for the key %s", alias))
	if items, err := pcc.Pcc.GetSecurityKeys(); err == nil {
		for _, c := range items {
			if c.Alias == alias {
				goto cont
			}
		}
		t.Fatal(fmt.Sprintf("not able to found the key %s", alias))
	} else {
		t.Fatal(err)
	}

cont:
	fmt.Println("getting the key", key.Alias)
	if k, err := pcc.Pcc.GetSecurityKey(alias); err == nil {
		if k.Tenant != key.Tenant || k.Alias != key.Alias || k.Description != key.Description || k.Protect != key.Protect {
			t.Fatal("the describe returned some different values", key, k)
		}
	} else {
		t.Fatal(err)
	}

	fmt.Println("updating the key", key.Alias)
	previous := key.Description
	key.Description = key.Description + "new"
	if err := pcc.Pcc.UpdateSecurityKey(key); err == nil {
		if previous == key.Description {
			t.Fatalf("the description does not change for the key %s", key.Alias)
		}
	} else {
		t.Fatal(err)
	}
}

func testKMCertificates(t *testing.T) {
	fmt.Printf("\n\nCERTIFICATES:\n")
	var err error

	alias := "certificate-bb-test"
	fmt.Println(fmt.Sprintf("looking for the certificate %s and deleting if exists", alias))
	var cert pcc.Certificate
	if items, err := pcc.Pcc.GetCertificates(); err == nil {
		for _, c := range items {
			if c.Alias == alias {
				pcc.Pcc.DeleteCertificate(c.Id)
				break
			}
		}
	} else {
		t.Fatal(err)
	}

	fileName := fmt.Sprintf("%s.crt", alias)
	keyName := fmt.Sprintf("%s.pem", alias)
	fmt.Println(fmt.Sprintf("generating the certificate %s %s", alias, fileName))
	cmd := exec.Command("/usr/bin/openssl", "req", "-nodes", "-new", "-x509", "-keyout", keyName, "-out", fileName, "--subj", "/C=US/ST=SanFrancisco/L=SanFrancisco/O=Global Security/OU=IT Department/CN=platinasystems.net")
	if err = cmd.Run(); err != nil {
		t.Fatal(err)
	}

	fmt.Println("uploading the certificate", fileName)
	if cert, err = pcc.Pcc.UploadCert(fileName, alias, ""); err != nil {
		t.Fatal(err)
	}

	fmt.Println("uploaded the certificate", cert.Id, cert.Alias)
	defer func() {
		fmt.Println("deleting the certificate", cert.Id, cert.Alias)
		pcc.Pcc.DeleteCertificate(cert.Id) // delete at the end
	}()

	if c, err := pcc.Pcc.GetCertificate(cert.Id); err == nil {
		if c.Id != cert.Id || c.Description != cert.Description || c.Tenant != cert.Tenant || c.Protect != cert.Protect {
			t.Fatal("the describe returned some different values", c, cert)
		}
	} else {
		t.Fatal(err)
	}

	fmt.Println("comparing the content for the certificate", cert)
	if content, err := pcc.Pcc.DownloadCertificate(cert.Id); err == nil { // compare the content
		readFileAndCompare(t, content, fileName)
	} else {
		t.Fatal(err)
	}
}

// read from file and compare the content
func readFileAndCompare(t *testing.T, content string, fileName string) {
	if file, err := os.Open(fileName); err == nil {
		defer file.Close()
		if b, err := ioutil.ReadAll(file); err == nil {
			if string(b) != content {
				t.Fatal("the downloaded file is different from", fileName)
			}
		} else {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}
}
