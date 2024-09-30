package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	albv1alpha1 "github.com/yandex-cloud/yc-alb-ingress-controller/api/v1alpha1"
)

func TestServiceLoader_Load(t *testing.T) {
	svcBasic := core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "basic-svc",
			Namespace: "basic-ns",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
	}

	svcWithFinalizer := core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "final-svc",
			Namespace: "basic-ns",
			Finalizers: []string{
				Finalizer,
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
	}

	svcToDelete := core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deleted-svc",
			Namespace: "basic-ns",
			DeletionTimestamp: &metav1.Time{
				Time: time.Date(2023, time.January, 11, 11, 52, 14, 0, time.Local),
			},
			Finalizers: []string{
				Finalizer,
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
	}

	basicIngress := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "basic-ing",
			Namespace:   "basic-ns",
			Annotations: map[string]string{AlbTag: "default"},
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: "basic-svc",
										},
									},
								},
								{
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: "deleted-svc",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ingWithDefaultBackend := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "basic-ing",
			Namespace:   "basic-ns",
			Annotations: map[string]string{AlbTag: "default"},
		},
		Spec: networking.IngressSpec{
			DefaultBackend: &networking.IngressBackend{
				Service: &networking.IngressServiceBackend{
					Name: "basic-svc",
				},
			},
			Rules: []networking.IngressRule{},
		},
	}

	testData := []struct {
		desc    string
		objects []client.Object
		svc     types.NamespacedName
		exp     ServiceToReconcile
		wantErr bool
	}{
		{
			desc: "basic",
			svc: types.NamespacedName{
				Name:      "basic-svc",
				Namespace: "basic-ns",
			},
			objects: []client.Object{&svcBasic, &basicIngress},
			exp: ServiceToReconcile{
				ToReconcile: &svcBasic,
			},
		},
		{
			desc: "with-finalizer-no-more-managed-to-delete",
			svc: types.NamespacedName{
				Name:      "final-svc",
				Namespace: "basic-ns",
			},
			objects: []client.Object{&svcWithFinalizer, &basicIngress},
			exp: ServiceToReconcile{
				ToDelete: &svcWithFinalizer,
			},
		},
		{
			desc: "with-deletion-ts-to-delete",
			svc: types.NamespacedName{
				Name:      "deleted-svc",
				Namespace: "basic-ns",
			},
			objects: []client.Object{&svcToDelete, &basicIngress},
			exp: ServiceToReconcile{
				ToDelete: &svcToDelete,
			},
		},
		{
			desc: "default",
			svc: types.NamespacedName{
				Name:      "basic-svc",
				Namespace: "basic-ns",
			},
			objects: []client.Object{&svcBasic, &ingWithDefaultBackend},
			exp: ServiceToReconcile{
				ToReconcile: &svcBasic,
			},
		},
	}

	for _, entry := range testData {
		t.Run(entry.desc, func(t *testing.T) {
			i := entry.objects

			ctx := context.Background()

			err := albv1alpha1.AddToScheme(scheme.Scheme)
			assert.NoError(t, err)

			cli := fake.NewClientBuilder().WithObjects(i...).WithScheme(scheme.Scheme).Build()
			loader := DefaultServiceLoader{Client: cli}

			svc, err := loader.Load(ctx, entry.svc)
			if entry.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			if entry.exp.ToReconcile != nil {
				assert.Equal(t, *entry.exp.ToReconcile, *svc.ToReconcile)
			} else {
				assert.Nil(t, entry.exp.ToReconcile)
			}

			if entry.exp.ToDelete != nil {
				assert.Equal(t, *entry.exp.ToDelete, *svc.ToDelete)
			} else {
				assert.Nil(t, entry.exp.ToDelete)
			}
		})
	}
}
