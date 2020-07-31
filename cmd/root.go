package cmd

import (
	"fmt"
	"os"

	"github.com/mattn/go-colorable"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version string
	rootCmd = &cobra.Command{
		Use:     "treb",
		Version: version,
		Short:   "Easily interact with Amazon ECR.",
		Long: `Trebuchet is a tool used to improve the quality of life for pushing Docker images to 
Amazon Elastic Container Registry (ECR). It launches images into ECR.

Authentication:
	In order for trebuchet to authenticate with AWS services, use the default credentials options that the 
	AWS CLI supports. This includes the ~/.aws/credentials file or AWS environment variables.

	For environment variables, ensure AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are set.
	For the credentials file, ensure the file exists at ~/.aws/credentials and the above properties are set.

	If the AWS credentials or config file are in non-standard locations (~/.aws), the AWS_SHARED_CREDENTIALS_FILE
	or AWS_CONFIG_FILE environment variables can be set to point to the location of those files.

Verbose:
	The verbose flag is a global flag that enables debug logging. The default is false.`,
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	flags := rootCmd.PersistentFlags()

	flags.BoolP("verbose", "v", false, "Enables verbose logging.")
	flags.StringP("as", "a", "",
		"Amazon Resource Name (ARN) specifying the role to be assumed.")
	flags.StringP("region", "r", "",
		"AWS region to be used. Supported as flag, AWS_DEFAULT_REGION environment variable or AWS Config File.")
	flags.StringP("profile", "p", "",
		"AWS Shared Credentials profile to be used.")
	_ = viper.BindPFlags(flags)

	cobra.OnInitialize(initLogrus)
}

func initLogrus() {
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetOutput(colorable.NewColorableStdout())
	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}
}
