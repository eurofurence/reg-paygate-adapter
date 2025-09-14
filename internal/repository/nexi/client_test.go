package nexi

import (
	"crypto/hmac"
	"encoding/base64"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSignRequest(t *testing.T) {
	request := "a=b&c=d"

	signature := signRequest(request, "secret")
	decodedSig, _ := base64.StdEncoding.DecodeString(signature)

	// echo -n "a=b&c=d" | openssl dgst -sha256 -hmac "secret" -binary | openssl enc -base64
	compareTo := `dteWgtm+Tl2TM7dYXTCS9c6atObKTQpGtEk6b+vpVKY=`
	decodedCompare, _ := base64.StdEncoding.DecodeString(compareTo)

	require.True(t, hmac.Equal(decodedSig, decodedCompare))
}

func TestSignEmpty(t *testing.T) {
	request := ""

	signature := signRequest(request, "secret")
	decodedSig, _ := base64.StdEncoding.DecodeString(signature)

	// echo -n "" | openssl dgst -sha256 -hmac "secret" -binary | openssl enc -base64
	compareTo := `+eZuF5tnR65UEI+C+K3os8Jddv0wr95sOVgixTAZYWk=`
	decodedCompare, _ := base64.StdEncoding.DecodeString(compareTo)

	require.True(t, hmac.Equal(decodedSig, decodedCompare))
}
