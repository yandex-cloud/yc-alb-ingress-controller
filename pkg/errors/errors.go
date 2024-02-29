package errors2

import (
	"fmt"
)

type OperationIncompleteError struct {
	ID string
}

func (e OperationIncompleteError) Error() string {
	return fmt.Sprintf("operation %s is still in progress", e.ID)
}

// ResourceNotReadyError should be thrown when a reconciliation chain requests a resource that will be reconciled with
// another reconciliation chain, e.g. when Ingress reconciliation requires a target group, but the Node reconciliation
// which creates it has not yet been run
type ResourceNotReadyError struct {
	ResourceType, Name string
}

func (e ResourceNotReadyError) Error() string {
	return fmt.Sprintf("resource %s (%s) not ready", e.ResourceType, e.Name)
}

// YCResourceNotReadyError same as ResourceNotReadyError but for YC
type YCResourceNotReadyError struct {
	ResourceType, Name string
}

func (e YCResourceNotReadyError) Error() string {
	return fmt.Sprintf("resource %s (%s) not ready", e.ResourceType, e.Name)
}
