package docker

import (
	"testing"

	"github.com/hylandsoftware/trebuchet/internal/ecr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDockerClient struct {
	mock.Mock
}

func (m *mockDockerClient) ImageExists(image string) error {
	args := m.Called(image)
	return args.Error(0)
}

func (m *mockDockerClient) ImagePush(image string, auth ecr.RegistryAuth) error {
	args := m.Called(image, auth)
	return args.Error(0)
}

func (m *mockDockerClient) ImagePull(image string, auth ecr.RegistryAuth) error {
	args := m.Called(image, auth)
	return args.Error(0)
}

func (m *mockDockerClient) ImageTag(source string, target string) error {
	args := m.Called(source, target)
	return args.Error(0)
}

func (m *mockDockerClient) ImageRemove(image string) error {
	args := m.Called(image)
	return args.Error(0)
}

func TestDockerClient_Push_ValidPush(t *testing.T) {
	m := &mockDockerClient{}
	m.On("ImageTag", mock.Anything, mock.Anything).Return(nil)
	m.On("ImagePush", mock.Anything, ecr.RegistryAuth{}).Return(nil)
	m.On("ImageRemove", mock.Anything).Return(nil)

	err := TagAndPush(m, "", "", ecr.RegistryAuth{})

	require.NoError(t, err)
}

func TestDockerClient_Push_ImageTagReturnsError(t *testing.T) {
	m := &mockDockerClient{}
	m.On("ImageTag", mock.Anything, mock.Anything).Return(errors.New("error"))

	err := TagAndPush(m, "", "", ecr.RegistryAuth{})

	require.EqualError(t, err, "error")
}

func TestDockerClient_Push_ImagePushReturnsError(t *testing.T) {
	m := &mockDockerClient{}
	m.On("ImageTag", mock.Anything, mock.Anything).Return(nil)
	m.On("ImageRemove", mock.Anything).Return(nil)
	m.On("ImagePush", mock.Anything, ecr.RegistryAuth{}).Return(errors.New("error"))

	err := TagAndPush(m, "", "", ecr.RegistryAuth{})

	require.EqualError(t, err, "error: %!s(<nil>)")
}

func TestDockerClient_Pull_ValidPull(t *testing.T) {
	m := &mockDockerClient{}
	m.On("ImagePull", mock.Anything, ecr.RegistryAuth{}).Return(nil)

	err := Pull(m, "", "", false, ecr.RegistryAuth{})

	require.NoError(t, err)
}

func TestDockerClient_Pull_ImageTagReturnsError(t *testing.T) {
	m := &mockDockerClient{}
	m.On("ImagePull", mock.Anything, ecr.RegistryAuth{}).Return(nil)
	m.On("ImageTag", mock.Anything, mock.Anything).Return(errors.New("error"))

	err := Pull(m, "", "", true, ecr.RegistryAuth{})

	require.EqualError(t, err, "error")
}

func TestDockerClient_Pull_ImagePullReturnsError(t *testing.T) {
	m := &mockDockerClient{}
	m.On("ImagePull", mock.Anything, ecr.RegistryAuth{}).Return(errors.New("error"))

	err := Pull(m, "", "", false, ecr.RegistryAuth{})

	require.EqualError(t, err, "error")
}

func TestDockerClient_GetFullECRImageReference_EmptySuffixReturnsOnlyURI(t *testing.T) {
	result := getFullECRImageReference("https://ecr.com/repository/image", "")

	require.Equal(t, "https://ecr.com/repository/image", result)
}

func TestDockerClient_GetFullECRImageReference_NotEmptySuffixReturnsBothURIAndSuffix(t *testing.T) {
	result := getFullECRImageReference("https://ecr.com/repository/image", "image:v1.2.3")

	require.Equal(t, "https://ecr.com/repository/image:v1.2.3", result)
}

func TestEncodeRegistryAuthentication_ValidAuth(t *testing.T) {
	auth := ecr.RegistryAuth{
		Username: "AWS",
		Password: "somepassword",
	}

	result, err := encodeRegistryAuthentication(auth)

	require.NoError(t, err)
	require.Equal(t, "eyJ1c2VybmFtZSI6IkFXUyIsInBhc3N3b3JkIjoic29tZXBhc3N3b3JkIn0=", result)
}
