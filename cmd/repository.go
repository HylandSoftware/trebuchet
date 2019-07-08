package cmd

import (
	"fmt"

	"github.com/hylandsoftware/trebuchet/internal/ecr"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var repositoryCmd = &cobra.Command{
	Use:     "repository REPOSITORY",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"repo"},
	Example: `treb repository helloworld --region us-east-1
treb repo some/project/helloworld --region us-west-2 --as arn:aws:iam::112233445566
treb repo my/repository`,
	Short: "Get the full URL of a repository in Amazon ECR",
	Long: `Get the full URL of a repository in Amazon ECR. The repository command will lookup the repository passed in
to see if it exists in Amazon ECR and return it to be used for deployment or reference purposes. 

Region:
	Region is required to be set as a flag, as an AWS environment variable (AWS_DEFAULT_REGION), or in the AWS config.

Amazon Resource Name (ARN):
	Passing in a valid ARN allows trebuchet to assume a role to perform actions within AWS. A typical use-case for this
	would be a service account to use in a software pipeline to interact with ECR.
	`,
	Run: repository,
}

func repository(cmd *cobra.Command, args []string) {
	log.SetLevel(log.ErrorLevel)

	ecrClient, err := ecr.NewClient(viper.GetString("region"), viper.GetString("as"))
	if err != nil {
		log.WithError(err).Fatal("Error in creation of ECR client")
	}

	repositoryURI, err := ecrClient.GetRepositoryURI(args[0])
	if err != nil {
		log.WithError(err).Fatal("Error getting repository URI")
	}

	fmt.Println(repositoryURI)
}

func init() {
	rootCmd.AddCommand(repositoryCmd)
}
