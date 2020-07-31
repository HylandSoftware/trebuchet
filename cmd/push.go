package cmd

import (
	"github.com/hylandsoftware/trebuchet/internal/docker"
	"github.com/hylandsoftware/trebuchet/internal/ecr"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"strings"
)

var pushCmd = &cobra.Command{
	Use:     "push NAME[:TAG]",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"launch", "fling"},
	Example: `treb push -v --region us-east-1 helloworld:1.2.3
treb launch -v --as arn:aws:iam::112233445566:role/PushToECR --region us-west-1 hello/world:3.4-beta
treb push helloworld:latest`,
	Short: "Pushes a Docker image into ECR",
	Long: `Pushes a Docker image into ECR

Region:
	Region is required to be set as a flag, as an AWS environment variable (AWS_DEFAULT_REGION), or in the AWS config.

Amazon Resource Name (ARN):
	Passing in a valid ARN allows trebuchet to assume a role to perform actions within AWS. A typical use-case for this
	would be a service account to use in a software pipeline to push images to ECR.

Aliases:
	trebuchet push can also be used as 'treb launch' or 'treb fling' for a more authentic experience.`,
	Run: push,
}

func push(cmd *cobra.Command, args []string) {
	ecrClient, err := ecr.NewClient(viper.GetString("region"), viper.GetString("as"), viper.GetString("profile"))
	if err != nil {
		log.WithError(err).Fatal("Error in creation of ECR client")
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		log.WithError(err).Fatal("Error creating Docker client")
	}

	if err := dockerClient.ImageExists(args[0]); err != nil {
		log.WithError(err).WithField("image", args[0]).Fatal("Error validating Docker image")
	}

	dockerImage := args[0]
	repository := parseDockerRepositoryFromImage(dockerImage)

	repositoryURI, err := ecr.SetupRepository(ecrClient, repository)
	if err != nil {
		log.WithError(err).WithField("image", repository).Fatal("Error setting up repository for image")
	}

	auth, err := ecrClient.GetAuthorizationToken()
	if err != nil {
		log.WithError(err).Fatal("Error getting authorization token for ECR")
	}

	if err := docker.TagAndPush(dockerClient, dockerImage, repositoryURI, *auth); err != nil {
		log.WithError(err).WithField("image", dockerImage).Fatal("Error pushing Docker image")
	}
}

func parseDockerRepositoryFromImage(image string) string {
	if strings.Contains(image, ":") {
		return strings.Split(image, ":")[0]
	}
	return image
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
