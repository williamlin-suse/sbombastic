package handlers

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"path"

	"go.uber.org/zap"

	"github.com/google/go-containerregistry/pkg/name"
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rancher/sbombastic/api/v1alpha1"
	registryclient "github.com/rancher/sbombastic/internal/handlers/registry"
	"github.com/rancher/sbombastic/internal/messaging"
)

// CreateCatalogHandler is a handler for creating a catalog of images in a registry.
type CreateCatalogHandler struct {
	registryClientFactory registryclient.ClientFactory
	k8sClient             client.Client
	logger                *zap.Logger
}

func NewCreateCatalogHandler(registryClientFactory registryclient.ClientFactory, k8sClient client.Client, logger *zap.Logger) *CreateCatalogHandler {
	return &CreateCatalogHandler{
		registryClientFactory: registryClientFactory,
		k8sClient:             k8sClient,
		logger:                logger,
	}
}

func (h *CreateCatalogHandler) Handle(message messaging.Message) error {
	createCatalogMessage, ok := message.(*messaging.CreateCatalog)
	if !ok {
		return fmt.Errorf("expected *messaging.CreateCatalog, got %T", message)
	}

	h.logger.Debug("Catalog creation requested",
		zap.String("registry", createCatalogMessage.RegistryName),
		zap.String("namespace", createCatalogMessage.RegistryNamespace),
	)

	registry := &v1alpha1.Registry{}
	err := h.k8sClient.Get(context.Background(), client.ObjectKey{
		Name:      createCatalogMessage.RegistryName,
		Namespace: createCatalogMessage.RegistryNamespace,
	}, registry)
	if err != nil {
		return fmt.Errorf("cannot get registry %s/%s: %w", createCatalogMessage.RegistryNamespace, createCatalogMessage.RegistryName, err)
	}

	h.logger.Debug("Registry found",
		zap.Any("registry", registry),
	)

	transport := h.transportFromRegistry(registry)
	registryClient := h.registryClientFactory(transport)
	ctx := context.Background()

	repositories, err := h.discoverRepositories(ctx, registryClient, registry)
	if err != nil {
		return fmt.Errorf("cannot discover repositories: %w", err)
	}

	var imageNames []string
	for _, repository := range repositories {
		repoImages, err := h.discoverImages(ctx, registryClient, repository)
		if err != nil {
			h.logger.Error(
				"cannot discover images",
				zap.String("repository", repository),
				zap.Error(err),
			)
			continue
		}
		imageNames = append(imageNames, repoImages...)
	}

	for _, imageName := range imageNames {
		ref, err := name.ParseReference(imageName)
		if err != nil {
			h.logger.Error(
				"cannot parse image name",
				zap.String("image", imageName),
				zap.Error(err),
			)
			continue
		}

		images, err := h.refToImages(registryClient, ref)
		if err != nil {
			h.logger.Error(
				"cannot convert reference to Image",
				zap.String("image", ref.Name()),
				zap.Error(err),
			)
			continue
		}
		for _, image := range images {
			// TODO: ignore creation of images that already exist
			if err := h.k8sClient.Create(ctx, &image); err != nil {
				return fmt.Errorf("cannot create image %s: %w", image.Name, err)
			}
		}
	}

	// TODO: remove images that are not in the registry anymore

	return nil
}

func (h *CreateCatalogHandler) NewMessage() messaging.Message {
	return &messaging.CreateCatalog{}
}

// discoverRepositories discovers all the repositories in a registry.
// Returns the list of fully qualified repository names (e.g. registryclientexample.com/repo)
func (h *CreateCatalogHandler) discoverRepositories(ctx context.Context, registryClient registryclient.Client, registry *v1alpha1.Registry) ([]string, error) {
	reg, err := name.NewRegistry(registry.Spec.URL)
	if err != nil {
		return nil, fmt.Errorf("cannot parse registry %s %s: %w", registry.Name, registry.Namespace, err)
	}

	// If the registry doesn't have any repositories defined, it means we need to catalog all of them.
	// In this case, we need to discover all the repositories in the registry.
	if len(registry.Spec.Repositories) == 0 {
		allRepositories, err := registryClient.Catalog(ctx, reg)
		if err != nil {
			return []string{}, fmt.Errorf("cannot discover repositories: %w", err)
		}

		return allRepositories, nil
	}

	repositories := []string{}
	for _, repository := range registry.Spec.Repositories {
		repositories = append(repositories, path.Join(reg.Name(), repository))
	}

	return repositories, nil
}

// discoverImages discovers all the images defined inside of a repository.
// Returns the list of fully qualified image names (e.g. registryclientexample.com/repo:tag)
func (h *CreateCatalogHandler) discoverImages(ctx context.Context, registryClient registryclient.Client, repository string) ([]string, error) {
	repo, err := name.NewRepository(repository)
	if err != nil {
		return []string{}, fmt.Errorf("cannot parse repository name %q: %w", repository, err)
	}

	contents, err := registryClient.ListRepositoryContents(ctx, repo)
	if err != nil {
		return []string{}, fmt.Errorf("cannot list repository contents: %w", err)
	}

	return contents, nil
}

func (h *CreateCatalogHandler) refToImages(registryClient registryclient.Client, ref name.Reference) ([]v1alpha1.Image, error) {
	platforms, err := h.refToPlatforms(registryClient, ref)
	if err != nil {
		return []v1alpha1.Image{}, fmt.Errorf("cannot get platforms for %s: %w", ref, err)
	}
	if platforms == nil {
		// add a `nil` platform to the list of platforms, this will be used to get the default platform
		platforms = append(platforms, nil)
	}

	images := []v1alpha1.Image{}

	for _, platform := range platforms {
		imageDetails, err := registryClient.GetImageDetails(ref, platform)
		if err != nil {
			platformStr := "default"
			if platform != nil {
				platformStr = platform.String()
			}

			h.logger.Error(
				"cannot get image details",
				zap.String("image", ref.Name()),
				zap.String("platform", platformStr),
				zap.Error(err))
			continue
		}

		image, err := imageDetailsToImage(ref, imageDetails)
		if err != nil {
			h.logger.Error("cannot convert image details to image", zap.Error(err))
			continue
		}

		images = append(images, image)
	}

	return images, nil
}

// refToPlatforms returns the list of platforms for the given image reference.
// If the image is not multi-architecture, it returns an empty list.
func (h *CreateCatalogHandler) refToPlatforms(registryClient registryclient.Client, ref name.Reference) ([]*cranev1.Platform, error) {
	imgIndex, err := registryClient.GetImageIndex(ref)
	if err != nil {
		h.logger.Debug(
			"image doesn't seem to be multi-architecture",
			zap.String("image", ref.Name()),
			zap.Error(err))
		return []*cranev1.Platform(nil), nil
	}

	manifest, err := imgIndex.IndexManifest()
	if err != nil {
		return []*cranev1.Platform(nil), fmt.Errorf("cannot read index manifest of %s: %w", ref, err)
	}

	platforms := make([]*cranev1.Platform, len(manifest.Manifests))
	for i, manifest := range manifest.Manifests {
		platforms[i] = manifest.Platform
	}

	return platforms, nil
}

// transportFromRegistry creates a new http.RoundTripper from the options specified in the Registry spec.
func (h *CreateCatalogHandler) transportFromRegistry(registry *v1alpha1.Registry) http.RoundTripper {
	transport := remote.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: registry.Spec.Insecure, //nolint:gosec // this a user provided option
	}

	if len(registry.Spec.CABundle) > 0 {
		rootCAs, err := x509.SystemCertPool()
		if err != nil {
			h.logger.Error("cannot load system cert pool, using empty pool", zap.Error(err))
			rootCAs = x509.NewCertPool()
		}

		ok := rootCAs.AppendCertsFromPEM([]byte(registry.Spec.CABundle))
		if ok {
			transport.TLSClientConfig.RootCAs = rootCAs
		} else {
			h.logger.Info("cannot load the given CA bundle")
		}
	}

	return transport
}

func imageDetailsToImage(ref name.Reference, details registryclient.ImageDetails) (v1alpha1.Image, error) {
	imageLayers := []v1alpha1.ImageLayer{}

	// There can be more history entries than layers, as some history entries are empty layers
	// For example, a command like "ENV VAR=1" will create a new history entry but no new layer
	layerCounter := 0
	for _, history := range details.History {
		if history.EmptyLayer {
			continue
		}

		if len(details.Layers) < layerCounter {
			return v1alpha1.Image{}, fmt.Errorf("layer %d not found - got only %d layers", layerCounter, len(details.Layers))
		}
		layer := details.Layers[layerCounter]
		digest, err := layer.Digest()
		if err != nil {
			return v1alpha1.Image{}, fmt.Errorf("cannot read layer digest: %w", err)
		}
		diffID, err := layer.DiffID()
		if err != nil {
			return v1alpha1.Image{}, fmt.Errorf("cannot read layer diffID: %w", err)
		}

		imageLayers = append(imageLayers, v1alpha1.ImageLayer{
			Command: base64.StdEncoding.EncodeToString([]byte(history.CreatedBy)),
			Digest:  digest.String(),
			DiffID:  diffID.String(),
		})

		layerCounter++
	}

	image := v1alpha1.Image{
		ObjectMeta: metav1.ObjectMeta{
			Name: computeImageUID(ref, details.Digest.String()),
			Labels: map[string]string{
				v1alpha1.ImageRegistryLabel:   ref.Context().RegistryStr(),
				v1alpha1.ImageRepositoryLabel: ref.Context().RepositoryStr(),
				v1alpha1.ImageTagLabel:        ref.Identifier(),
				v1alpha1.ImagePlatformLabel:   details.Platform.String(),
				v1alpha1.ImageDigestLabel:     details.Digest.String(),
			},
		},
		Spec: v1alpha1.ImageSpec{
			Layers: imageLayers,
		},
	}

	return image, nil
}

// computeImageUID returns the sha256 of “<image-name>@sha256:<digest>`
func computeImageUID(ref name.Reference, digest string) string {
	sha := sha256.New()
	sha.Write([]byte(fmt.Sprintf("%s:%s@%s", ref.Context().Name(), ref.Identifier(), digest)))
	return hex.EncodeToString(sha.Sum(nil))
}
