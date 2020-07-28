package sts

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/require"
)

func TestStsClient_Retrieve_ReturnsErrorWithNilCredentials(t *testing.T) {
	dummyProvider := &CredentialsProvider{}

	_, err := dummyProvider.Retrieve(context.Background())

	require.Equal(t, ErrNoSTSCredentialsFound, err)
}

func TestStsClient_Retrieve_ReturnsValidCredentials(t *testing.T) {
	dummyProvider := &CredentialsProvider{
		Credentials: &sts.Credentials{
			AccessKeyId:     aws.String("abcd"),
			Expiration:      &time.Time{},
			SecretAccessKey: aws.String("efgh"),
			SessionToken:    aws.String("ijkl"),
		},
	}

	expected := aws.Credentials{
		AccessKeyID:     "abcd",
		SecretAccessKey: "efgh",
		SessionToken:    "ijkl",
		Source:          "",
		CanExpire:       false,
		Expires:         time.Time{},
	}

	result, err := dummyProvider.Retrieve(context.Background())

	require.NoError(t, err)
	require.Equal(t, expected, result)
}
