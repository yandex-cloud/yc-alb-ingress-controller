package errors

import (
	"errors"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	ctrl "sigs.k8s.io/controller-runtime"

	ycerrors "github.com/yandex-cloud/alb-ingress/pkg/errors"
)

const (
	FailureRequeueInterval = 30
)

const (
	DONE    = "done"
	REQUEUE = "requeue"
	FAIL    = "fail"
)

func HandleError(err error, log logr.Logger) (ctrl.Result, error) {
	var outcome string
	var st *status.Status
	defer func() { logResult(log, outcome, err, st) }()

	if err == nil {
		outcome = DONE
		return ctrl.Result{}, nil
	}
	st = grpcStatus(err)
	if errors.As(err, &ycerrors.ResourceNotReadyError{}) ||
		errors.As(err, &ycerrors.OperationIncompleteError{}) ||
		errors.As(err, &ycerrors.YCResourceNotReadyError{}) ||
		st != nil && st.Code() == codes.FailedPrecondition {
		outcome = REQUEUE
		return ctrl.Result{RequeueAfter: time.Second * FailureRequeueInterval}, nil
	}

	outcome = FAIL
	return ctrl.Result{}, err
}

func grpcStatus(err error) *status.Status {
	for err != nil {
		if e, ok := err.(interface{ GRPCStatus() *status.Status }); ok {
			return e.GRPCStatus()
		}
		err = errors.Unwrap(err)
	}
	return nil
}

// log.info for FAIL outcome reports GRPC status details and a short error description
// fully unwrapped error message is automatically printed by kubebuilder's internal controller's log.error invocation
func logResult(log logr.Logger, outcome string, err error, st *status.Status) {
	log = log.WithValues("result", outcome)
	if err != nil {
		if details := st.Details(); details != nil {
			log = log.WithValues("reason", st.Message())
			log = log.WithValues("details", details)
		} else {
			log = log.WithValues("reason", err.Error())
		}
	}
	log.Info("reconcile attempt completed")
}
