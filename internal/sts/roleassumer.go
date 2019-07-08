package sts

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	log "github.com/sirupsen/logrus"
)

var ErrNoSTSCredentialsFound = errors.New("no STS credentials were found")

type RoleAssumer interface {
	AssumeRole(config aws.Config, assumeRole string) (*CredentialsProvider, error)
}

type stsRoleAssumer struct {
	log *log.Entry
}

func (r *stsRoleAssumer) AssumeRole(config aws.Config, assumeRole string) (*CredentialsProvider, error) {
	stsClient := sts.New(config)

	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(assumeRole),
		RoleSessionName: aws.String("TrebuchetAssumedRole"),
	}
	out, err := stsClient.AssumeRoleRequest(input).Send()

	if err != nil {
		r.log.WithField("role", assumeRole).Info("Error attempting to assume role")
		return nil, err
	}

	r.log.WithField("role", assumeRole).Info("Successfully assumed role")
	return &CredentialsProvider{Credentials: out.Credentials}, nil
}

func NewRoleAssumer() RoleAssumer {
	return &stsRoleAssumer{
		log: log.WithField("component", "sts"),
	}
}

type CredentialsProvider struct {
	*sts.Credentials
}

func (s CredentialsProvider) Retrieve() (aws.Credentials, error) {
	if s.Credentials == nil {
		return aws.Credentials{}, ErrNoSTSCredentialsFound
	}

	return aws.Credentials{
		AccessKeyID:     aws.StringValue(s.AccessKeyId),
		SecretAccessKey: aws.StringValue(s.SecretAccessKey),
		SessionToken:    aws.StringValue(s.SessionToken),
		Expires:         aws.TimeValue(s.Expiration),
	}, nil
}
