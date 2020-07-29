package cmd

import (
	"github.com/hylandsoftware/trebuchet/internal/docker"
	"github.com/hylandsoftware/trebuchet/internal/ecr"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pullCmd = &cobra.Command{
	Use:     "pull NAME[:TAG]",
	Args:    cobra.ExactArgs(1),
	Example: `treb pull -v --region us-east-1 helloworld:1.2.3
treb pull -v --as arn:aws:iam::112233445566:role/PullFromECR --region us-west-1 hello/world:3.4-beta
treb pull --strip helloworld:latest`,
	Short: "Pulls a Docker image from ECR",
	Long: `Pulls a Docker image from ECR

Strip:
	Strip is a boolean flag. When set, it removes all ECR-specific elements from the image name. For example, 
	112233445566.dkr.ecr.us-east-1.amazonaws.com/hello-world:latest would be pulled as hello-world:latest.

Region:
	Region is required to be set as a flag, as an AWS environment variable (AWS_DEFAULT_REGION), or in the AWS config.

Amazon Resource Name (ARN):
	Passing in a valid ARN allows trebuchet to assume a role to perform actions within AWS. A typical use-case for this
	would be a service account to use in a software pipeline to push images to ECR.`,
	Run: pull,
}

func pull(cmd *cobra.Command, args []string) {
	ecrClient, err := ecr.NewClient(viper.GetString("region"), viper.GetString("as"))
	if err != nil {
		log.WithError(err).Fatal("Error in creation of ECR client")
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		log.WithError(err).Fatal("Error creating Docker client")
	}

	dockerImage := args[0]
	repository := parseDockerRepositoryFromImage(dockerImage)

	if ok, _ := ecrClient.RepositoryExists(repository); !ok {
		log.Fatal("ECR repository does not exist")
	}

	repositoryURI, err := ecrClient.GetRepositoryURI(repository)
	if err != nil {
		log.WithError(err).Fatal("Error retrieving full repository name")
	}

	auth, err := ecrClient.GetAuthorizationToken()
	if err != nil {
		log.WithError(err).Fatal("Error getting authorization token for ECR")
	}

	if err := docker.Pull(dockerClient, dockerImage, repositoryURI, viper.GetBool("strip"), *auth); err != nil {
		log.WithError(err).WithField("image", dockerImage).Fatal("Error pulling Docker image")
	}
}

func init() {
	flags := pullCmd.Flags()
	flags.BoolP("strip", "s", true, "strip the image name of ECR-specific elements")
	_ = viper.BindPFlags(flags)
	rootCmd.AddCommand(pullCmd)
}