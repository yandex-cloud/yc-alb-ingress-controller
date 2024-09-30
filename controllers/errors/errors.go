package errors

import (
	"errors"
	"reflect"
	"time"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	ctrl "sigs.k8s.io/controller-runtime"

	ycerrors "github.com/yandex-cloud/yc-alb-ingress-controller/pkg/errors"
)

const (
	FailureRequeueInterval = 30
)

const (
	DONE    = "done"
	REQUEUE = "requeue"
	FAIL    = "fail"
)

type Object interface {
	meta.Object
	runtime.Object
}

func HandleError(err error, log logr.Logger) (ctrl.Result, error) {
	var outcome string
	var st *status.Status
	defer func() { logResult(log, outcome, err, st) }()

	outcome = errorOutcome(err)
	switch outcome {
	case DONE:
		return ctrl.Result{}, nil
	case REQUEUE:
		return ctrl.Result{RequeueAfter: FailureRequeueInterval * time.Second}, nil
	}
	return ctrl.Result{}, err
}

func HandleErrorWithObject(err error, obj Object, recorder record.EventRecorder) {
	if isNil(obj) {
		return
	}
	outcome := errorOutcome(err)
	switch outcome {
	case DONE:
		recorder.Eventf(obj, core.EventTypeNormal, "ReconciliationComplete", "Reconciliation complete for %s", obj.GetName())
	case FAIL:
		recorder.Eventf(obj, core.EventTypeWarning, "ReconciliationFailed", "Reconciliation failed for %s: %s", obj.GetName(), err.Error())
	}
}

func isNil(obj interface{}) bool {
	return obj == nil ||
		(reflect.ValueOf(obj).Kind() == reflect.Ptr && reflect.ValueOf(obj).IsNil())
}

func errorOutcome(err error) string {
	if err == nil {
		return DONE
	}
	st := grpcStatus(err)
	if errors.As(err, &ycerrors.ResourceNotReadyError{}) ||
		errors.As(err, &ycerrors.OperationIncompleteError{}) ||
		errors.As(err, &ycerrors.YCResourceNotReadyError{}) ||
		st != nil && st.Code() == codes.FailedPrecondition {
		return REQUEUE
	}
	return FAIL
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
