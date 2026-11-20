package crypto

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCipherEncryptDecryptRoundTrip(t *testing.T) {
	hexKey, err := GenerateMasterKeyHex()
	require.NoError(t, err)
	require.Len(t, hexKey, 64)

	keyBytes, err := DecodeMasterKey(hexKey)
	require.NoError(t, err)
	require.Len(t, keyBytes, 32)

	b64Key, err := DecodeMasterKey(base64.StdEncoding.EncodeToString(keyBytes))
	require.NoError(t, err)
	require.Equal(t, keyBytes, b64Key)

	rawB64Key, err := DecodeMasterKey(base64.RawStdEncoding.EncodeToString(keyBytes))
	require.NoError(t, err)
	require.Equal(t, keyBytes, rawB64Key)

	c, err := NewCipher(keyBytes)
	require.NoError(t, err)

	emptyEnc, err := c.EncryptString("")
	require.NoError(t, err)
	require.Empty(t, emptyEnc)

	emptyDec, err := c.DecryptString("")
	require.NoError(t, err)
	require.Empty(t, emptyDec)

	enc, err := c.EncryptString("sk_super_secret_api_key_123456")
	require.NoError(t, err)
	require.NotEmpty(t, enc)
	require.NotContains(t, enc, "sk_super_secret")

	dec, err := c.DecryptString(enc)
	require.NoError(t, err)
	require.Equal(t, "sk_super_secret_api_key_123456", dec)

	// 错误密钥解密失败
	otherHex, err := GenerateMasterKeyHex()
	require.NoError(t, err)
	otherBytes, err := DecodeMasterKey(otherHex)
	require.NoError(t, err)
	otherCipher, err := NewCipher(otherBytes)
	require.NoError(t, err)

	_, err = otherCipher.DecryptString(enc)
	require.Error(t, err)
}

func TestDecodeMasterKeyRejectsInvalidInput(t *testing.T) {
	_, err := DecodeMasterKey("too-short")
	require.Error(t, err)

	_, err = NewCipher([]byte("short"))
	require.Error(t, err)
}
