# Trebuchet - Launch container images into Amazon ECR
[![Build Status](https://travis-ci.org/HylandSoftware/trebuchet.svg?branch=master)](https://travis-ci.org/HylandSoftware/trebuchet) [![Coverage Status](https://coveralls.io/repos/github/HylandSoftware/trebuchet/badge.svg?branch=master)](https://coveralls.io/github/HylandSoftware/trebuchet?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/hylandsoftware/spot)](https://goreportcard.com/report/github.com/hylandsoftware/trebuchet)

![](logo/trebuchet_200x200.png)

----

The purpose of Trebuchet is to improve the quality of life for pushing Docker images to Amazon Elastic Container Registry (ECR).

## Usage
`Trebuchet` is shipped as a single binary (Linux/Windows) and as a Docker image. All images can be found [here](https://hub.docker.com/r/hylandsoftware/trebuchet).

### Commands
`push`:

```
Pushes a Docker image into ECR

Region:
        Region is required to be set as a flag, as an AWS environment variable (AWS_DEFAULT_REGION), or in the AWS config.

Amazon Resource Name (ARN):
        Passing in a valid ARN allows trebuchet to assume a role to perform actions within AWS. A typical use-case for this
        would be a service account to use in a software pipeline to push images to ECR.

Aliases:
        trebuchet push can also be used as 'treb launch' or 'treb fling' for a more authentic experience.

Usage:
  treb push NAME[:TAG] [flags]

Aliases:
  push, launch, fling

Examples:
treb push -v --region us-east-1 helloworld:1.2.3
treb launch -v --as arn:aws:iam::112233445566:role/PushToECR --region us-west-1 hello/world:3.4-beta
treb push helloworld:latest

Flags:
  -h, --help   help for push

Global Flags:
  -a, --as string       Amazon Resource Name (ARN) specifying the role to be assumed.
  -r, --region string   AWS region to be used. Supported as flag, AWS_DEFAULT_REGION environment variable or AWS Config File.
  -v, --verbose         Enables verbose logging.
```

`repository`: 
```
Get the full URL of a repository in Amazon ECR. The repository command will lookup the repository passed in
to see if it exists in Amazon ECR and return it to be used for deployment or reference purposes.

Region:
        Region is required to be set as a flag, as an AWS environment variable (AWS_DEFAULT_REGION), or in the AWS config.

Amazon Resource Name (ARN):
        Passing in a valid ARN allows trebuchet to assume a role to perform actions within AWS. A typical use-case for this
        would be a service account to use in a software pipeline to interact with ECR.

Usage:
  treb repository REPOSITORY [flags]

Aliases:
  repository, repo

Examples:
treb repository helloworld --region us-east-1
treb repo some/project/helloworld --region us-west-2 --as arn:aws:iam::112233445566
treb repo my/repository

Flags:
  -h, --help   help for repository

Global Flags:
  -a, --as string       Amazon Resource Name (ARN) specifying the role to be assumed.
  -r, --region string   AWS region to be used. Supported as flag, AWS_DEFAULT_REGION environment variable or AWS Config File.
  -v, --verbose         Enables verbose logging.
```

### AWS Authentication and Settings Precedence
`Trebuchet` uses the default AWS credentials chain and supports flags for specifying region and/or a role to assume.
Precedence of credentials and configuration that are loaded in `Trebuchet`:
1. Flags passed to Trebuchet
   - Example: `treb push --as arn:aws:iam::112233445566:role/JenkinsPushToECR --region us-east-1 [some-image]`
2. Environment variables. For more information, reference the [Environment Variables](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html)
   section of the AWS Command Line documentation.
   - Examples: `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` for authentication or `AWS_DEFAULT_REGION` to specify region
3. AWS config/credentials files located at `~/.aws/config` and `~/.aws/credentials`.
For more information, reference the [Named Profiles](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html)
section of the AWS Command Line documentation.
    - Examples: `aws_access_key_id` and `aws_secret_access_key` in the credentials file or `region` and `role_arn` in the config file

#### IAM Permissions

The User or IAM Role you are assuming needs at least the following permissions
to create the repository if it doesn't exist and push images into ECR:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ecr:CreateRepository",
                "ecr:BatchGetImage",
                "ecr:CompleteLayerUpload",
                "ecr:GetAuthorizationToken",
                "ecr:DescribeRepositories",
                "ecr:UploadLayerPart",
                "ecr:InitiateLayerUpload",
                "ecr:BatchCheckLayerAvailability",
                "ecr:PutImage"
            ],
            "Resource": "*"
        }
    ]
}
```

### `Push` Command Usage in a Jenkins pipeline
Example usage in Jenkins using the [kubenetes-plugin](https://github.com/jenkinsci/kubernetes-plugin)

`build-spec.yml`:

``` yml
spec:
  containers:
  - name: jnlp
    image: jenkins/jnlp-slave
  - name: trebuchet
    image: hylandsoftware/trebuchet
    tty: true
    securityContext:
      privileged: true
```

> **NOTE**: The trebuchet docker image uses a `docker:dind` daemon as its
> entrypoint. **You do not need to include a separate docker container in your
> jenkins build spec**. All tasks interacting with docker should execute inside
> the trebuchet container

#### Using `WithCredentials` to provide AWS credentials
`Jenkinsfile`:

``` groovy
pipeline {
    agent {
        kubernetes {
            label "Push-To-ECR-Example"
            yamlFile 'build-spec.yml'
        }
    }

    environment {
        AWS_DEFAULT_REGION = 'us-east-1'
    }

    stages {
        stage('Build Image') {
            steps {
                container('trebuchet') {
                    sh 'docker build . -t hello-world:1.2.3'
                }
            }
        }
        stage('Push Docker Image to ECR') {
            steps {
                withCredentials([[$class: 'AmazonWebServicesCredentialsBinding', credentialsId: 'aws_credentials_id']) {
                    container('trebuchet') {
                        sh 'treb push hello-world:1.2.3 -v --as arn:aws:iam:1122334455:role/JenkinsPushToECR'
                    }
                }
            }
        }
    }
}
```

#### Using an AWS Credentials File
In this example, using an AWS credential file, we can use profiles from the file to set region, access keys, and roles to assume.
In the Jenkinsfile, we set [2 AWS environment variables](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html): `AWS_PROFILE` (the profile we want to use) and `AWS_SHARED_CREDENTIALS_FILE` to tell
Trebuchet where to find the credentials file as it not in the default `~/.aws/credentials` location.

Credentials file stored in Jenkins as a secret file with id `aws-credentials-file`:

```
[default-jenkins]
region = us-east-1
aws_access_key_id = ABCDEFGHIJKLMNOP
aws_secret_access_key = *******************

[some-profile]
output = json
region = us-east-1
role_arn = arn:aws:iam::112233445566:role/JenkinsPushToECR
source_profile = default-jenkins
```
`Jenkinsfile`:

```groovy
pipeline {
    agent {
        kubernetes {
            label "Push-To-ECR-Example"
            yamlFile 'build-spec.yml'
        }
    }

    stages {
        stage('Build Image') {
            steps {
                container('trebuchet') {
                    sh 'docker build . -t hello-world:1.2.3'
                }
            }
        }
        stage('Push Docker Image to ECR') {
            environment {
                AWS_PROFILE = 'some-profile'
                AWS_SHARED_CREDENTIALS_FILE = credentials('aws-credentials-file')
            }
        
            steps {
                container('trebuchet') {
                    sh 'treb fling hello-world:1.2.3'
                }
            }
        }
    }
}
```

### `Repository` Command Usage in a Jenkins Pipeline
The `repository` command will return the full URL to a repository given as an argument to `trebuchet`. This allows
a pipeline to save this output as an environment variable and pass it to a deployment tool like [Helm](https://helm.sh/)
to specify the location of an image to pull instead of hard-coding it in the Helm chart.

Here's a shortened example of getting the full repository URL and upgrading a deployment via Helm and setting the 
`image.repository` to the environment variable created in the `Get Repository URL` stage.
```groovy
pipeline {
    agent {
        ...
    }
    
    stages {
        // build...
        
        // trebuchet push...
        
        stage('Get Repository URL') {
            container('trebuchet') {
                script {
                    REPOSITORY_URL = sh(returnStdout: true, script: 'treb repo hello-world --region us-east-1').trim()
                }
            }
        }
        
        stage('Deploy') {
            container('helm') {
                sh '''
                    helm upgrade hello-world deployments/helloworld --namespace helloworld --set image.repository=${REPOSITORY_URL}
                '''
            }
        }
    }
}
```


## Building
Requirements:
- Go version `1.11+` or `vgo`, for Go Modules support

``` bash
# linux
go build -o treb -v main.go

# windows
go build -o treb.exe -v main.go
```
