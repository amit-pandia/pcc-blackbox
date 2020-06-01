package main

import (
	"fmt"
	"testing"

	"github.com/mitchellh/mapstructure"
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
)

const (
	LDAP_CERT_FILENAME = "test_ldap_crt"
)

var CurrentAuthProfileName string

func AddAuthenticationProfile(t *testing.T) {
	t.Run("addAuthProfile", addAuthProfile)
}

func UploadSecurityAuthProfileCert(t *testing.T) {
	t.Run("uploadSecurityAuthProfileCert", uploadCertificate_AuthProfile)
}

func uploadCertificate_AuthProfile(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	err := CreateFileAndUpload(LDAP_CERT_FILENAME, LDAP_CERT, pcc.CERT)
	if err != nil {
		assert.Fatalf(err.Error())
	}
}

func addAuthProfile(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	var (
		authProfile pcc.AuthenticationProfile
	)

	if Env.AuthenticationProfile.Name == "" {
		fmt.Printf("Authenticatiom Profile is not defined in the" +
			" configuration file\n")
		return
	}
	authProfile = Env.AuthenticationProfile

	exist, certificate, err := pcc.Pcc.FindCertificate(LDAP_CERT_FILENAME)
	if err != nil {
		assert.Fatalf("Get certificate %s failed\n%v\n",
			LDAP_CERT_FILENAME, err)
		return
	} else if exist {
		if authProfile.Type == "LDAP" {
			var ldapConfiguration pcc.LDAPConfiguration

			decodeError := mapstructure.Decode(authProfile.Profile,
				&ldapConfiguration)
			if decodeError == nil {
				ldapConfiguration.CertificateId = &certificate.Id
				authProfile.Profile = ldapConfiguration
			}
		}
	}

	var label string
	for i := 1; ; i++ {
		label = fmt.Sprintf(authProfile.Name+"_%d", i)
		CurrentAuthProfileName = label
		existingProfile, _ := pcc.Pcc.GetAuthProfileByName(label)
		if existingProfile == nil {
			break
		}
	}
	authProfile.Name = label

	err = pcc.Pcc.AddAuthProfile(authProfile)
	if err != nil {
		assert.Fatalf("Error: %v\n", err)
		return
	}
}

func delAllProfiles(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	var (
		authProfiles []pcc.AuthenticationProfile
		err          error
		id           uint64
	)

	authProfiles, err = pcc.Pcc.GetAuthProfiles()
	if err != nil {
		assert.Fatalf("Failed to get auth profiles: %v\n", err)
		return
	}

	for _, aP := range authProfiles {
		id = aP.ID
		fmt.Printf("Deleting auth profile %v\n", aP.Name)
		err = pcc.Pcc.DelAuthProfile(id)
		if err != nil {
			assert.Fatalf("Failed to delete auth profile %v: %v\n",
				id, err)
		}
		// seems to be syncronous. API should document
	}
}
