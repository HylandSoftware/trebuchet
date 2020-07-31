package ecr

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/aws/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/hylandsoftware/trebuchet/internal/sts"
	log "github.com/sirupsen/logrus"
)

var (
	ErrNoTokenOrProxyEndpoint  = errors.New("no authorization token or proxy endpoint obtained when requesting token")
	ErrNoCredentials           = errors.New("no credentials provided")
)

type Client interface {
	RepositoryExists(repository string) (bool, error)
	CreateRepository(repository string) error
	GetRepositoryURI(repository string) (string, error)
	GetAuthorizationToken() (*RegistryAuth, error)
}

type RegistryAuth struct {
	ProxyEndpoint string
	Username      string
	Password      string
}

type ecrClient struct {
	*ecr.Client
	log *log.Entry
}

func NewClient(region string, assumeRole string, profile string) (Client, error) {
	config, err := getClientConfig(region, assumeRole, profile, sts.NewRoleAssumer(), external.LoadDefaultAWSConfig)
	if err != nil {
		return nil, err
	}

	return &ecrClient{
		Client: ecr.New(config),
		log: log.WithField("component", "ecr"),
	}, nil
}

type configLoaderFunc func(configs ...external.Config) (aws.Config, error)

func (c *ecrClient) RepositoryExists(repository string) (bool, error) {
	_, err := c.DescribeRepositoriesRequest(&ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{repository},
	}).Send(context.Background())

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == ecr.ErrCodeRepositoryNotFoundException {
			c.log.WithField("repository", repository).Info("Repository does not exist")
			return false, nil
		}
		return false, err
	}

	c.log.WithField("repository", repository).Info("Repository exists")
	return true, nil
}

func (c *ecrClient) CreateRepository(repository string) error {
	_, err := c.CreateRepositoryRequest(&ecr.CreateRepositoryInput{
		RepositoryName: &repository,
	}).Send(context.Background())

	if err != nil {
		c.log.WithField("repository", repository).Info("Error in creating repository")
		return err
	}

	c.log.WithField("repository", repository).Info("Successfully created repository")
	return nil
}

func (c *ecrClient) GetRepositoryURI(repository string) (string, error) {
	result, err := c.DescribeRepositoriesRequest(&ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{repository},
	}).Send(context.Background())

	if err != nil {
		return "", err
	}

	c.log.WithField("uri", *result.Repositories[0].RepositoryUri).Info("Repository URI")
	return *result.Repositories[0].RepositoryUri, nil
}

func (c *ecrClient) GetAuthorizationToken() (*RegistryAuth, error) {
	c.log.Debug("Getting authorization token")
	result, err := c.GetAuthorizationTokenRequest(&ecr.GetAuthorizationTokenInput{}).Send(context.Background())

	if err != nil {
		return nil, err
	}

	authorizationData := result.AuthorizationData[0]
	if authorizationData.ProxyEndpoint == nil && authorizationData.AuthorizationToken == nil {
		return nil, ErrNoTokenOrProxyEndpoint
	}

	auth, err := extractToken(*authorizationData.AuthorizationToken, *authorizationData.ProxyEndpoint)

	if err != nil {
		return nil, err
	}
	return auth, nil
}

// SetupRepository will check if a repository exists, create it if it does not,
// and then return the repository URI to access to repository.
func SetupRepository(c Client, repository string) (string, error) {
	repositoryExists, err := c.RepositoryExists(repository)
	if err != nil {
		return "", err
	}

	if !repositoryExists {
		if err := c.CreateRepository(repository); err != nil {
			return "", err
		}
	}

	repositoryURI, err := c.GetRepositoryURI(repository)
	if err != nil {
		return "", err
	}

	return repositoryURI, nil
}

func extractToken(token string, proxyEndpoint string) (*RegistryAuth, error) {
	decodedToken, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(string(decodedToken), ":", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid token: expected two parts, got %d", len(parts))
	}

	return &RegistryAuth{
		ProxyEndpoint: proxyEndpoint,
		Username:      parts[0],
		Password:      parts[1],
	}, nil
}

func getClientConfig(region string, assumeRole string, profile string, assumer sts.RoleAssumer, configLoader configLoaderFunc) (cfg aws.Config, err error) {
	if profile == "" {
		cfg, err = configLoader()
		if err != nil {
			return aws.Config{}, err
		}
	} else {
		log.WithField("profile", profile).Debug("Explicitly setting profile")
		cfg, err = configLoader(external.WithSharedConfigProfile(profile))
		if err != nil {
			return aws.Config{}, err
		}
	}

	if cfg.Credentials == nil {
		return aws.Config{}, ErrNoCredentials
	}

	if region != "" {
		log.WithField("region", region).Debug("Explicitly setting region")
		cfg.Region = region
	}

	// If assumeRole is specified, assume that role - except for when a role has already been assumed.
	if _, ok := cfg.Credentials.(*stscreds.AssumeRoleProvider); assumeRole != "" && !ok {
		newCredentials, err := assumer.AssumeRole(cfg, assumeRole)

		if err != nil {
			return aws.Config{}, err
		}

		cfg.Credentials = newCredentials
	}

	// Ensure the set region is valid and exists for the ECR service
	resolver := endpoints.NewDefaultResolver()
	resolver.StrictMatching = true
	if _, err = resolver.ResolveEndpoint("api.ecr", cfg.Region); err != nil {
		return aws.Config{}, err
	}

	return cfg, nil
}


