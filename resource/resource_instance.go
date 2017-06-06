package resource

import (
	"encoding/json"

	"code.cloudfoundry.org/lager"
	"github.com/concourse/atc"
	"github.com/concourse/atc/db"
	"github.com/concourse/atc/worker"
	"github.com/concourse/baggageclaim"
)

//go:generate counterfeiter . ResourceInstance

type ResourceInstance interface {
	ResourceUser() db.ResourceUser
	ContainerOwner() db.ContainerOwner

	FindInitializedOn(lager.Logger, worker.Client) (worker.Volume, bool, error)
	CreateOn(lager.Logger, worker.Client) (worker.Volume, error)

	ResourceCacheIdentifier() worker.ResourceCacheIdentifier
}

type resourceInstance struct {
	resourceTypeName       ResourceType
	version                atc.Version
	source                 atc.Source
	params                 atc.Params
	resourceUser           db.ResourceUser
	containerOwner         db.ContainerOwner
	resourceTypes          atc.VersionedResourceTypes
	dbResourceCacheFactory db.ResourceCacheFactory
}

func NewResourceInstance(
	resourceTypeName ResourceType,
	version atc.Version,
	source atc.Source,
	params atc.Params,
	resourceUser db.ResourceUser,
	containerOwner db.ContainerOwner,
	resourceTypes atc.VersionedResourceTypes,
	dbResourceCacheFactory db.ResourceCacheFactory,
) ResourceInstance {
	return &resourceInstance{
		resourceTypeName:       resourceTypeName,
		version:                version,
		source:                 source,
		params:                 params,
		resourceUser:           resourceUser,
		containerOwner:         containerOwner,
		resourceTypes:          resourceTypes,
		dbResourceCacheFactory: dbResourceCacheFactory,
	}
}

func (instance resourceInstance) ResourceUser() db.ResourceUser {
	return instance.resourceUser
}

func (instance resourceInstance) ContainerOwner() db.ContainerOwner {
	return instance.containerOwner
}

func (instance resourceInstance) CreateOn(logger lager.Logger, workerClient worker.Client) (worker.Volume, error) {
	resourceCache, err := instance.dbResourceCacheFactory.FindOrCreateResourceCache(
		logger,
		instance.resourceUser,
		string(instance.resourceTypeName),
		instance.version,
		instance.source,
		instance.params,
		instance.resourceTypes,
	)
	if err != nil {
		return nil, err
	}

	return workerClient.CreateVolumeForResourceCache(
		logger,
		worker.VolumeSpec{
			Strategy: baggageclaim.EmptyStrategy{},
		},
		resourceCache,
	)
}

func (instance resourceInstance) FindInitializedOn(logger lager.Logger, workerClient worker.Client) (worker.Volume, bool, error) {
	resourceCache, err := instance.dbResourceCacheFactory.FindOrCreateResourceCache(
		logger,
		instance.resourceUser,
		string(instance.resourceTypeName),
		instance.version,
		instance.source,
		instance.params,
		instance.resourceTypes,
	)
	if err != nil {
		logger.Error("failed-to-find-or-initialized-volume-resource-cache-for-build", err)
		return nil, false, err
	}

	return workerClient.FindInitializedVolumeForResourceCache(
		logger,
		resourceCache,
	)
}

func (instance resourceInstance) ResourceCacheIdentifier() worker.ResourceCacheIdentifier {
	return worker.ResourceCacheIdentifier{
		ResourceVersion: instance.version,
		ResourceHash:    GenerateResourceHash(instance.source, string(instance.resourceTypeName)),
	}
}

func GenerateResourceHash(source atc.Source, resourceType string) string {
	sourceJSON, _ := json.Marshal(source)
	return resourceType + string(sourceJSON)
}
