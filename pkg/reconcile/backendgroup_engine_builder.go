package reconcile

import (
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"

	albv1alpha1 "github.com/yandex-cloud/alb-ingress/api/v1alpha1"
	"github.com/yandex-cloud/alb-ingress/pkg/builders"
)

type DefaultBackendGroupEngineBuilder struct {
	factory               *builders.Factory
	newBackendGroupEngine func(data *builders.Data) *BackendGroupEngine
}

func NewBackendGroupEngineBuilder(factory *builders.Factory, newEngineFn func(data *builders.Data) *BackendGroupEngine) *DefaultBackendGroupEngineBuilder {
	return &DefaultBackendGroupEngineBuilder{
		factory:               factory,
		newBackendGroupEngine: newEngineFn,
	}
}

func (d *DefaultBackendGroupEngineBuilder) Build(crd *albv1alpha1.HttpBackendGroup) (*BackendGroupEngine, error) {
	b := builders.Data{}
	var err error
	// TODO: using builders.BackendGroups for only one BG doesn't seem quite right
	b.BackendGroups, err = d.buildBackendGroups(crd)
	if err != nil {
		return nil, err
	}
	return d.newBackendGroupEngine(&b), nil
}

func (d *DefaultBackendGroupEngineBuilder) buildBackendGroups(crd *albv1alpha1.HttpBackendGroup) (*builders.BackendGroups, error) {
	b := d.factory.BackendGroupForCRDBuilder()
	bg, err := b.BuildBgFromCR(crd)
	if err != nil {
		return nil, err
	}
	return &builders.BackendGroups{
		BackendGroups:      []*apploadbalancer.BackendGroup{bg},
		BackendGroupByName: map[string]*apploadbalancer.BackendGroup{bg.Name: bg},
	}, nil
}
