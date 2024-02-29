package k8s

import (
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func NewSecretsStore(secretsEventChan chan<- event.GenericEvent, keyFunc cache.KeyFunc) *SecretsStore {
	return &SecretsStore{
		eventChan: secretsEventChan,
		store:     cache.NewStore(keyFunc),
	}
}

type SecretsStore struct {
	store     cache.Store
	eventChan chan<- event.GenericEvent
}

func (s *SecretsStore) Add(i interface{}) error {
	if err := s.store.Add(i); err != nil {
		return err
	}

	obj := i.(client.Object)
	s.eventChan <- event.GenericEvent{
		Object: obj,
	}
	return nil
}

func (s *SecretsStore) Update(i interface{}) error {
	if err := s.store.Update(i); err != nil {
		return err
	}

	obj := i.(client.Object)
	s.eventChan <- event.GenericEvent{
		Object: obj,
	}

	return nil
}

func (s *SecretsStore) Delete(i interface{}) error {
	if err := s.store.Delete(i); err != nil {
		return err
	}

	obj := i.(client.Object)
	s.eventChan <- event.GenericEvent{
		Object: obj,
	}

	return nil
}

func (s *SecretsStore) Replace(list []interface{}, resourceVersion string) error {
	return s.store.Replace(list, resourceVersion)
}

func (s *SecretsStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return s.store.Get(obj)
}

func (s *SecretsStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	return s.store.GetByKey(key)
}

func (s *SecretsStore) List() []interface{} {
	return s.store.List()
}

func (s *SecretsStore) ListKeys() []string {
	return s.store.ListKeys()
}

func (s *SecretsStore) Resync() error {
	return s.store.Resync()
}
