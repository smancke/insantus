package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CertCheck_Success(t *testing.T) {
	check, _ := NewCertCheck(
		"prod",
		"test-check",
		"test check",
		map[string]string{
			"host":        "valid-isrgrootx1.letsencrypt.org",
			"minValidFor": "1s",
		},
	)

	results := check.Check()

	require.Equal(t, 1, len(results))

	result := results[0]
	assert.Equal(t, StatusUp, result.Status)
}

func Test_CertCheck_WillExpire(t *testing.T) {
	check, err := NewCertCheck(
		"prod",
		"test-check",
		"test check",
		map[string]string{
			"host":        "valid-isrgrootx1.letsencrypt.org",
			"port":        "443",
			"minValidFor": "87600h", // 10 years
		},
	)
	require.NoError(t, err)

	results := check.Check()

	require.Equal(t, 1, len(results))

	result := results[0]
	assert.Equal(t, StatusDown, result.Status)
}

func Test_CertCheck_Expired(t *testing.T) {
	check, err := NewCertCheck(
		"prod",
		"test-check",
		"test check",
		map[string]string{
			"host":        "expired-isrgrootx1.letsencrypt.org",
			"port":        "443",
			"minValidFor": "1h", // 10 years
		},
	)
	require.NoError(t, err)

	results := check.Check()

	require.Equal(t, 1, len(results))

	result := results[0]
	assert.Equal(t, StatusDown, result.Status)
}
