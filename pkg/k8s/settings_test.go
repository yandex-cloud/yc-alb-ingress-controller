package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	albv1alpha1 "github.com/yandex-cloud/yc-alb-ingress-controller/api/v1alpha1"
	networking "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGroupSettingsLoader_Load(t *testing.T) {
	NoSettings := IngressGroup{
		Items: []networking.Ingress{
			ingWithName("ing1"),
			ingWithName("ing2"),
			ingWithName("ing3"),
		},
	}

	EmptySettings := IngressGroup{
		Items: []networking.Ingress{
			ingWithSettingsAnnotation("ing1", ""),
			ingWithSettingsAnnotation("ing2", ""),
			ingWithSettingsAnnotation("ing3", ""),
		},
	}

	OnlyOneSettings := IngressGroup{
		Items: []networking.Ingress{
			ingWithSettingsAnnotation("with-settings", "default-settings"),
			ingWithName("without-settings"),
			ingWithName("without-settings1"),
			ingWithName("without-settings2"),
		},
	}

	MoreThanOneSettingsValid := IngressGroup{
		Items: []networking.Ingress{
			ingWithSettingsAnnotation("with-settings1", "default-settings"),
			ingWithSettingsAnnotation("with-settings2", "default-settings"),
			ingWithSettingsAnnotation("with-settings3", "default-settings"),
			ingWithName("without-settings"),
		},
	}

	MoreThanOneSettingsInvalid := IngressGroup{
		Items: []networking.Ingress{
			ingWithSettingsAnnotation("with-default-settings", "default-settings"),
			ingWithSettingsAnnotation("with-nondefault-settings2", "nondefault-settings"),
			ingWithName("without-settings"),
		},
	}

	DefaultSettings := albv1alpha1.IngressGroupSettings{
		ObjectMeta: v1.ObjectMeta{
			Name: "default-settings",
		},
	}

	testData := []struct {
		desc    string
		objects []client.Object
		g       IngressGroup
		exp     *albv1alpha1.IngressGroupSettings
		wantErr bool
	}{
		{
			desc:    "without-settings",
			objects: []client.Object{},
			g:       NoSettings,
			exp:     nil,
		},
		{
			desc:    "with-empty-settings",
			objects: []client.Object{},
			g:       EmptySettings,
			wantErr: true,
		},
		{
			desc:    "only-one-settings",
			objects: []client.Object{&DefaultSettings},
			g:       OnlyOneSettings,
			exp:     &DefaultSettings,
		},
		{
			desc:    "more-than-one-settings-valid",
			objects: []client.Object{&DefaultSettings},
			g:       MoreThanOneSettingsValid,
			exp:     &DefaultSettings,
		},
		{
			desc:    "more-than-one-settings-invalid",
			objects: []client.Object{&DefaultSettings},
			g:       MoreThanOneSettingsInvalid,
			wantErr: true,
		},
	}

	for _, entry := range testData {
		t.Run(entry.desc, func(t *testing.T) {
			ctx := context.Background()

			err := albv1alpha1.AddToScheme(scheme.Scheme)
			assert.NoError(t, err)

			cli := fake.NewClientBuilder().WithObjects(entry.objects...).WithScheme(scheme.Scheme).Build()
			loader := GroupSettingsLoader{Client: cli}

			act, err := loader.Load(ctx, &entry.g)
			if entry.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			if entry.exp == nil {
				assert.Nil(t, act)
			} else {
				assert.Equal(t, entry.exp.Name, act.Name)
			}
		})
	}
}

func ingWithSettingsAnnotation(name, annotation string) networking.Ingress {
	ing := ingWithName(name)
	ing.Annotations[GroupSettings] = annotation
	return ing
}
