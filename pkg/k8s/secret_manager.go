package k8s

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

const CertIDPrefix = "yc-certmgr-cert-id-"

type SecretManager interface {
	ManageGroup(ctx context.Context, group *IngressGroup)
}

type secretManager struct {
	secretFollowers map[types.NamespacedName]*secretFollower

	clientSet        kubernetes.Interface
	secretsEventChan chan<- event.GenericEvent
}

type secretFollower struct {
	groups map[string]struct{}

	rt        *cache.Reflector
	closeChan chan struct{}
}

func NewSecretManager(clientSet kubernetes.Interface, secretEventChan chan<- event.GenericEvent) SecretManager {
	return &secretManager{
		secretFollowers:  make(map[types.NamespacedName]*secretFollower),
		clientSet:        clientSet,
		secretsEventChan: secretEventChan,
	}
}

func (m *secretManager) newSecretFollower(secret types.NamespacedName) *secretFollower {
	fieldSelector := fields.Set{"metadata.name": secret.Name}.AsSelector().String()
	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector
		return m.clientSet.CoreV1().Secrets(secret.Namespace).List(context.TODO(), options)
	}

	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.FieldSelector = fieldSelector
		return m.clientSet.CoreV1().Secrets(secret.Namespace).Watch(context.TODO(), options)
	}
	store := NewSecretsStore(m.secretsEventChan, cache.MetaNamespaceKeyFunc)
	rt := cache.NewNamedReflector(
		fmt.Sprintf("secret-%s/%s", secret.Namespace, secret.Name),
		&cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc},
		&corev1.Secret{},
		store,
		0,
	)

	sf := &secretFollower{
		groups:    make(map[string]struct{}),
		rt:        rt,
		closeChan: make(chan struct{}),
	}

	return sf
}

func (m *secretManager) ManageGroup(_ context.Context, group *IngressGroup) {
	secrets := ParseSecrets(group.Items)

	for secret := range secrets {
		if strings.HasPrefix(secret.Name, CertIDPrefix) {
			continue
		}

		sf, ok := m.secretFollowers[secret]
		if !ok {
			m.secretFollowers[secret] = m.newSecretFollower(secret)
			sf = m.secretFollowers[secret]
			go sf.rt.Run(sf.closeChan)

			m.secretsEventChan <- event.GenericEvent{
				Object: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: secret.Namespace,
						Name:      secret.Name,
					},
				},
			}
		}

		sf.groups[group.Tag] = struct{}{}
	}

	for secret, sf := range m.secretFollowers {
		_, ok := secrets[secret]
		if ok {
			continue
		}

		_, ok = sf.groups[group.Tag]
		if ok {
			delete(sf.groups, group.Tag)
		}

		if len(sf.groups) == 0 {
			close(sf.closeChan)
			m.secretsEventChan <- event.GenericEvent{
				Object: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: secret.Namespace,
						Name:      secret.Name,
					},
				},
			}
			delete(m.secretFollowers, secret)
		}
	}
}

func ParseSecrets(ings []networking.Ingress) map[types.NamespacedName]struct{} {
	result := make(map[types.NamespacedName]struct{})

	for _, item := range ings {
		for _, tls := range item.Spec.TLS {
			result[types.NamespacedName{
				Name:      tls.SecretName,
				Namespace: item.Namespace,
			}] = struct{}{}
		}
	}

	return result
}
