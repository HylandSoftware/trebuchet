package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/hylandsoftware/trebuchet/internal/ecr"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var ErrImageNotFound = errors.New("image not found on Docker host")

type Client interface {
	ImageExists(image string) error
	ImagePush(image string, auth ecr.RegistryAuth) error
	ImageTag(source string, target string) error
	ImageRemove(image string) error
}

type dockerClient struct {
	*client.Client
	log *logrus.Entry
}

// NewClient creates a new Docker client and logger to interact with the Docker API
func NewClient() (Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return &dockerClient{
		Client: cli,
		log:    log.WithField("component", "docker"),
	}, nil
}

// ImageExists verifies that the passed image exists on the Docker host
func (c *dockerClient) ImageExists(image string) error {
	imageFilters := filters.NewArgs()
	imageFilters.Add("reference", image)

	images, err := c.Client.ImageList(context.Background(), types.ImageListOptions{
		Filters: imageFilters,
	})

	if err != nil {
		return err
	}

	if len(images) == 0 {
		return ErrImageNotFound
	}

	return nil
}

// ImagePush pushes a Docker image from the Docker host to ECR
func (c *dockerClient) ImagePush(image string, auth ecr.RegistryAuth) error {
	token, err := encodeRegistryAuthentication(auth)
	if err != nil {
		return err
	}

	c.log.WithField("image", image).Info("Pushing image")
	output, err := c.Client.ImagePush(context.Background(), image, types.ImagePushOptions{
		RegistryAuth: token,
	})
	if err != nil {
		return err
	}

	defer output.Close()

	return jsonmessage.DisplayJSONMessagesStream(output, os.Stdout, 0, false, nil)
}

// ImageTag tags a Docker image on the Docker host with a new image name provided as the 'target' argument
func (c *dockerClient) ImageTag(source string, target string) error {
	c.log.WithFields(log.Fields{
		"source": source,
		"target": target,
	}).Info("Tagging image for ECR")
	return c.Client.ImageTag(context.Background(), source, target)
}

// ImageRemove removes an image from the Docker host
func (c *dockerClient) ImageRemove(image string) error {
	images, err := c.Client.ImageRemove(context.Background(), image, types.ImageRemoveOptions{})
	if err != nil {
		return err
	}

	for _, image := range images {
		c.log.WithField("image", image.Untagged).Info("Removed image")
	}

	return nil
}

// TagAndPush will tag the image using the 'repositoryURI' and tag in the 'image' on the Docker host, then push the image
// to ECR. The tagged image is always cleaned up, even if pushing the image fails.
func TagAndPush(dockerClient Client, image string, repositoryURI string, auth ecr.RegistryAuth) (err error) {
	fullRepositoryURI := getFullECRImageReference(repositoryURI, image)

	if err = dockerClient.ImageTag(image, fullRepositoryURI); err != nil {
		return err
	}

	defer func() {
		removeErr := dockerClient.ImageRemove(fullRepositoryURI)
		if err != nil {
			err = fmt.Errorf("%s: %s", err, removeErr)
		} else {
			err = removeErr
		}
	}()

	if err = dockerClient.ImagePush(fullRepositoryURI, auth); err != nil {
		return err
	}

	return nil
}

func getFullECRImageReference(repositoryURI string, image string) string {
	tag := ""
	if strings.Contains(image, ":") {
		tag = strings.SplitAfter(image, ":")[1]
	}

	if tag != "" {
		return repositoryURI + ":" + tag
	}

	return repositoryURI
}

func encodeRegistryAuthentication(auth ecr.RegistryAuth) (string, error) {
	authConfig := types.AuthConfig{
		Username: auth.Username,
		Password: auth.Password,
	}

	encodedAuthConfig, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}

	token := base64.URLEncoding.EncodeToString(encodedAuthConfig)

	return token, nil
}
