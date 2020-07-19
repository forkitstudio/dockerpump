package docker_client

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"io"
	"log"
	"os"
)

type DockerImage struct {
	Repository string
	Tag        string
}

type PumpError struct {
	Stage string `json:"stage"`
	Cause string `json:"cause"`
}

func (pe *PumpError) Error() string {
	return fmt.Sprintf("stage: %v, cause: %v", pe.Stage, pe.Cause)
}

func NewPumpError(stage string, err error) error {
	return &PumpError{Stage: stage, Cause: err.Error()}
}

func Health() (types.Ping, error) {
	var ping types.Ping
	context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("Error while initializing client, %v", err)
		return ping, NewPumpError("ping", err)
	}

	ping, err = cli.Ping(context.Background())
	if err != nil {
		log.Printf("ping execution error, %v", err)
		return ping, &PumpError{Stage: "ping", Cause: err.Error()}
	}

	log.Printf("APIVersion: %v\n", ping.APIVersion)
	log.Printf("OSType: %v\n", ping.OSType)
	log.Printf("Experimental: %v\n", ping.Experimental)
	log.Printf("BuilderVersion: %v\n", ping.BuilderVersion)

	return ping, nil
}

func CopyImage(sourceSrv, targetSrv string, image DockerImage, cleanupStore bool) error {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("Error while initializing client, %v", err)
		return &PumpError{Stage: "init", Cause: err.Error()}
	}

	sourceRepo := normalizeRepo(sourceSrv, image.Repository)
	targetRepo := normalizeRepo(targetSrv, image.Repository)

	err = mirrorRepository(ctx, cli, sourceRepo, targetRepo, image.Tag, cleanupStore)
	if err != nil {
		_, ok := err.(*PumpError)
		if ok {
			return NewPumpError("unknown", err)
		}
		return err
	}
	return nil
}

func mirrorRepository(ctx context.Context, cli *client.Client, sourceRepository, targetRepository string, tag string, cleanupStore bool) error {
	sourceRepositoryFull := repositoryFull(sourceRepository, tag)
	targetRepositoryFull := repositoryFull(targetRepository, tag)

	out, err := cli.ImagePull(ctx, sourceRepositoryFull, types.ImagePullOptions{})
	if err != nil {
		log.Printf("Error while pulling image, %v", err)
		return &PumpError{Stage: "pull", Cause: err.Error()}
	}
	defer out.Close()
	io.Copy(os.Stdout, out)
	log.Printf("Image pulled: %v\n", sourceRepositoryFull)

	err = cli.ImageTag(ctx, sourceRepositoryFull, targetRepositoryFull)
	if err != nil {
		log.Printf("Error while tagging image, %v", err)
		return &PumpError{Stage: "tag", Cause: err.Error()}
	}
	log.Printf("Image tagged: %v -> %v\n", sourceRepositoryFull, targetRepositoryFull)

	out, err = cli.ImagePush(ctx, targetRepositoryFull, types.ImagePushOptions{
		All:          true,
		RegistryAuth: "123"})
	if err != nil {
		log.Printf("Error while pushing image, %v", err)
		return &PumpError{Stage: "push", Cause: err.Error()}
	}
	defer out.Close()
	//io.Copy(os.Stdout, out)

	buffIOReader := bufio.NewReader(out)
	type ErrorMessage struct {
		Error string
	}
	var errorMessage ErrorMessage
	// Workaround: Error is nil if server timed out.
	for {
		streamBytes, err := buffIOReader.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		log.Println(string(streamBytes))
		_ = json.Unmarshal(streamBytes, &errorMessage)
		if errorMessage.Error != "" {
			return &PumpError{Stage: "push", Cause: errorMessage.Error}
		}
	}

	log.Printf("Image pushed: %v\n", targetRepositoryFull)

	if cleanupStore {
		imageID, _err := getImageId(ctx, cli, sourceRepositoryFull)
		if _err != nil {
			log.Printf("Error while looking up for specific image, %v", _err)
			return &PumpError{Stage: "lookupImage", Cause: _err.Error()}
		}
		_, _err = cli.ImageRemove(ctx, imageID, types.ImageRemoveOptions{Force: true})
		if _err != nil {
			log.Printf("Error while removing image, %v", _err)
			return &PumpError{Stage: "remove", Cause: _err.Error()}
		}
		log.Printf("Image removed from local registry: %v\n", imageID)
	}
	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	return nil
}

func getImageId(ctx context.Context, cli *client.Client, sourceRepository string) (string, error) {
	listFilters := filters.NewArgs()
	listFilters.Add("reference", sourceRepository)
	images, err := cli.ImageList(ctx, types.ImageListOptions{Filters: listFilters})
	if err != nil {
		panic(err)
	}

	for _, image := range images {
		if stringInSlice(sourceRepository, image.RepoTags) {
			return image.ID, nil
		}
	}
	return "", errors.New(fmt.Sprintf("Image not found %v in local storage", sourceRepository))
}

func normalizeRepo(server string, repository string) string {
	if len(server) < 1 {
		return repository
	}
	return server + "/" + repository
}

func repositoryFull(repository string, tag string) string {
	if len(tag) < 1 {
		return repository
	}
	return repository + ":" + tag
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
