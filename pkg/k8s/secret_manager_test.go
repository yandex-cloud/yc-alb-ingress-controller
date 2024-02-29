package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func Test_defaultSecretsManager_MonitorSecrets(t *testing.T) {
	ing1 := v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ing-1",
			Namespace: "ns-1",
		},
		Spec: v1.IngressSpec{
			TLS: []v1.IngressTLS{
				{
					SecretName: "secret-1",
				},
			},
		},
	}

	ing2 := v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ing-2",
			Namespace: "ns-2",
		},
		Spec: v1.IngressSpec{
			TLS: []v1.IngressTLS{
				{
					SecretName: "secret-2",
				},
			},
		},
	}

	ing3 := v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ing-3",
			Namespace: "ns-3",
		},
		Spec: v1.IngressSpec{
			TLS: []v1.IngressTLS{
				{
					SecretName: "secret-3",
				},
				{
					SecretName: "secret-4",
				},
			},
		},
	}

	ing4 := v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ing-4",
			Namespace: "ns-4",
		},
		Spec: v1.IngressSpec{
			TLS: []v1.IngressTLS{
				{
					SecretName: "secret-5",
				},
			},
		},
	}

	tests := []struct {
		testName           string
		monitorSecretsCall []IngressGroup
		wantSecrets        []types.NamespacedName
	}{
		{
			testName: "No secrets",
		},
		{
			testName: "Single group",
			monitorSecretsCall: []IngressGroup{
				{
					Items: []v1.Ingress{
						ing1,
					},
				},
			},
			wantSecrets: []types.NamespacedName{
				{Name: "secret-1", Namespace: "ns-1"},
			},
		},
		{
			testName: "Single group, multiple secrets",
			monitorSecretsCall: []IngressGroup{
				{
					Items: []v1.Ingress{
						ing1, ing2, ing3,
					},
				},
			},
			wantSecrets: []types.NamespacedName{
				{Name: "secret-1", Namespace: "ns-1"},
				{Name: "secret-2", Namespace: "ns-2"},
				{Name: "secret-3", Namespace: "ns-3"},
				{Name: "secret-4", Namespace: "ns-3"},
			},
		},
		{
			testName: "Multiple group, overlapping secrets",
			monitorSecretsCall: []IngressGroup{
				{
					Tag: "group-1",
					Items: []v1.Ingress{
						ing1, ing2, ing3,
					},
				},
				{
					Tag:   "group-2",
					Items: []v1.Ingress{ing2, ing3, ing4},
				},
			},
			wantSecrets: []types.NamespacedName{
				{Name: "secret-1", Namespace: "ns-1"},
				{Name: "secret-2", Namespace: "ns-2"},
				{Name: "secret-3", Namespace: "ns-3"},
				{Name: "secret-4", Namespace: "ns-3"},
				{Name: "secret-5", Namespace: "ns-4"},
			},
		},
		{
			testName: "Multiple group, with deletion",
			monitorSecretsCall: []IngressGroup{
				{
					Tag: "group-1",
					Items: []v1.Ingress{
						ing1, ing2, ing3,
					},
				},
				{
					Tag:   "group-2",
					Items: []v1.Ingress{ing2, ing3, ing4},
				},
				{
					Tag: "group-1",
					Items: []v1.Ingress{
						ing2, ing3,
					},
				},
			},
			wantSecrets: []types.NamespacedName{
				{Name: "secret-2", Namespace: "ns-2"},
				{Name: "secret-3", Namespace: "ns-3"},
				{Name: "secret-4", Namespace: "ns-3"},
				{Name: "secret-5", Namespace: "ns-4"},
			},
		},
		{
			testName: "Multiple group, delete all",
			monitorSecretsCall: []IngressGroup{
				{
					Tag: "group-1",
					Items: []v1.Ingress{
						ing1, ing2, ing3,
					},
				},
				{
					Tag:   "group-2",
					Items: []v1.Ingress{ing2, ing3, ing4},
				},
				{
					Tag:   "group-1",
					Items: []v1.Ingress{},
				},
				{
					Tag:   "group-2",
					Items: []v1.Ingress{},
				},
			},
			wantSecrets: []types.NamespacedName{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			secretsEventChan := make(chan event.GenericEvent, 100)
			fakeClient := fake.NewSimpleClientset()
			secretsManager := &secretManager{
				secretFollowers:  make(map[types.NamespacedName]*secretFollower),
				clientSet:        fakeClient,
				secretsEventChan: secretsEventChan,
			}

			for _, group := range tt.monitorSecretsCall {
				secretsManager.ManageGroup(context.Background(), &group)
			}
			assert.Equal(t, len(tt.wantSecrets), len(secretsManager.secretFollowers))
			for _, want := range tt.wantSecrets {
				_, exists := secretsManager.secretFollowers[want]
				assert.True(t, exists)
			}
		})
	}
}
