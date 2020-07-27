package ecr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/hylandsoftware/trebuchet/internal/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRoleAssumer struct {
	mock.Mock
}

func (m *mockRoleAssumer) AssumeRole(config aws.Config, arnRole string) (*sts.CredentialsProvider, error) {
	args := m.Called(config, arnRole)
	return args.Get(0).(*sts.CredentialsProvider), args.Error(1)
}

type mockECRClient struct {
	mock.Mock
}

func (m *mockECRClient) RepositoryExists(repository string) (bool, error) {
	args := m.Called(repository)
	return args.Bool(0), args.Error(1)
}

func (m *mockECRClient) CreateRepository(repository string) error {
	args := m.Called(repository)
	return args.Error(0)
}

func (m *mockECRClient) GetRepositoryURI(repository string) (string, error) {
	args := m.Called(repository)
	return args.String(0), args.Error(1)
}

func (m *mockECRClient) GetAuthorizationToken() (*RegistryAuth, error) {
	args := m.Called()
	return args.Get(0).(*RegistryAuth), args.Error(1)
}

func TestEcrClient_GetClientConfig_AssumeRoleUpdatesNewCredentials(t *testing.T) {
	m := &mockRoleAssumer{}
	dummyCredProvider := &sts.CredentialsProvider{}
	m.On("AssumeRole", mock.Anything, "testing").Return(dummyCredProvider, nil)

	result, err := getClientConfig("us-east-1", "testing", m, func(configs ...external.Config) (aws.Config, error) {
		return aws.Config{
			Region:      "us-east-1",
			Credentials: dummyCredProvider,
		}, nil
	})

	require.NoError(t, err)
	require.Equal(t, dummyCredProvider, result.Credentials)
}

func TestEcrClient_GetClientConfig_ReturnsErrorOnBadAssumeRole(t *testing.T) {
	m := &mockRoleAssumer{}
	dummyCredProvider := &sts.CredentialsProvider{}
	m.On("AssumeRole", mock.Anything, "testing").Return(dummyCredProvider, errors.New("some error"))

	_, err := getClientConfig("us-east-1", "testing", m, func(configs ...external.Config) (aws.Config, error) {
		return aws.Config{
			Region:      "us-east-1",
			Credentials: dummyCredProvider,
		}, nil
	})

	require.EqualError(t, err, "some error")
}

func TestEcrClient_GetClientConfig_RegionFlagUpdatesConfigRegion(t *testing.T) {
	m := &mockRoleAssumer{}
	dummyCredProvider := &sts.CredentialsProvider{}

	result, err := getClientConfig("us-east-2", "", m, func(configs ...external.Config) (aws.Config, error) {
		return aws.Config{
			Region:      "us-east-1",
			Credentials: dummyCredProvider,
		}, nil
	})

	require.NoError(t, err)
	require.Equal(t, "us-east-2", result.Region)
}

func TestEcrClient_GetClientConfig_ReturnsErrOnBadConfigLoad(t *testing.T) {
	m := &mockRoleAssumer{}

	_, err := getClientConfig("us-east-1", "", m, func(configs ...external.Config) (aws.Config, error) {
		return aws.Config{}, errors.New("some error")
	})

	require.EqualError(t, err, "some error")
}

func TestEcrClient_GetClientConfig_ReturnsErrNoCredentials(t *testing.T) {
	m := &mockRoleAssumer{}

	_, err := getClientConfig("us-east-1", "", m, func(configs ...external.Config) (aws.Config, error) {
		return aws.Config{
			Credentials: nil,
		}, nil
	})

	require.Equal(t, ErrNoCredentials, err)
}

func TestEcrClient_GetClientConfig_ReturnsErrorOnBadService(t *testing.T) {
	m := &mockRoleAssumer{}
	dummyCredProvider := &sts.CredentialsProvider{}

	_, err := getClientConfig("", "", m, func(configs ...external.Config) (aws.Config, error) {
		return aws.Config{
			Region:      "macho-man-randy-savage",
			Credentials: dummyCredProvider,
		}, nil
	})

	require.Error(t, err)
}

func TestEcrClient_NewClient_ReturnsValidClient(t *testing.T) {
	_, err := NewClient("us-east-1", "")

	assert.NoError(t, err)
}

func TestEcrClient_NewClient_ReturnsErrorForBadConfig(t *testing.T) {
	_, err := NewClient("macho-man-randy-savage", "")

	require.Error(t, err)
}

func TestEcrClient_ExtractToken_ReturnsValidToken(t *testing.T) {
	// AWS:ecrregistrycredentials
	token := "QVdTOmVjcnJlZ2lzdHJ5Y3JlZGVudGlhbHM="

	result, err := extractToken(token, "")

	require.NoError(t, err)
	require.Equal(t, "AWS", result.Username)
	require.Equal(t, "ecrregistrycredentials", result.Password)
}

func TestEcrClient_ExtractToken_ReturnsInvalidTokenErrorOnWrongNumberOfParts(t *testing.T) {
	// AWSecrregistrycredentials
	token := "QVdTZWNycmVnaXN0cnljcmVkZW50aWFscw=="

	_, err := extractToken(token, "")

	require.EqualError(t, err, fmt.Sprintf("invalid token: expected two parts, got %d", 1))
}

func TestEcrClient_SetupRepository_ReturnsValidRepositoryWhenNotExists(t *testing.T) {
	m := mockECRClient{}
	m.On("RepositoryExists", mock.Anything).Return(false, nil)
	m.On("CreateRepository", mock.Anything).Return(nil)
	m.On("GetRepositoryURI", mock.Anything).Return("someurl", nil)

	result, err := SetupRepository(&m, "myrepository")

	require.NoError(t, err)
	require.Equal(t, "someurl", result)
	require.Equal(t, true, m.AssertCalled(t, "CreateRepository", "myrepository"))
}

func TestEcrClient_SetupRepository_DoesNotCreateRepositoryWhenRepositoryExists(t *testing.T) {
	m := mockECRClient{}
	m.On("RepositoryExists", mock.Anything).Return(true, nil)
	m.On("GetRepositoryURI", mock.Anything).Return("someurl", nil)

	result, err := SetupRepository(&m, "myrepository")

	m.AssertNotCalled(t, "CreateRepository")
	require.NoError(t, err)
	require.Equal(t, "someurl", result)
}

func TestEcrClient_SetupRepository_ReturnsErrorOnRepositoryExistsError(t *testing.T) {
	m := mockECRClient{}
	m.On("RepositoryExists", mock.Anything).Return(false, errors.New("error"))

	result, err := SetupRepository(&m, "myrepository")

	require.EqualError(t, err, "error")
	require.Empty(t, result)
}

func TestEcrClient_SetupRepository_ReturnsErrorOnCreateRepositoryExistsError(t *testing.T) {
	m := mockECRClient{}
	m.On("RepositoryExists", mock.Anything).Return(false, nil)
	m.On("CreateRepository", mock.Anything).Return(errors.New("error"))

	result, err := SetupRepository(&m, "myrepository")

	require.EqualError(t, err, "error")
	require.Empty(t, result)
}

func TestEcrClient_SetupRepository_ReturnsErrorOnGetRepositoryURIError(t *testing.T) {
	m := mockECRClient{}
	m.On("RepositoryExists", mock.Anything).Return(true, nil)
	m.On("GetRepositoryURI", mock.Anything).Return("", errors.New("error"))

	result, err := SetupRepository(&m, "myrepository")

	require.EqualError(t, err, "error")
	require.Empty(t, result)
}
