package controllers

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"github.com/yandex-cloud/alb-ingress/pkg/k8s"
	"k8s.io/client-go/tools/record"
	"strings"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/certificatemanager/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	errors2 "github.com/yandex-cloud/alb-ingress/controllers/errors"
	"github.com/yandex-cloud/alb-ingress/pkg/metadata"
	"github.com/yandex-cloud/alb-ingress/pkg/yc"
)

type SecretController struct {
	cli   client.Client
	names *metadata.Names

	repo     yc.CertRepo
	recorder record.EventRecorder
}

func NewSecretController(cli client.Client, certRepo yc.CertRepo, names *metadata.Names) *SecretController {
	return &SecretController{
		cli:   cli,
		names: names,

		repo: certRepo,
	}
}

func (sc *SecretController) SetupWithManager(mgr ctrl.Manager, secretEventChan chan event.GenericEvent) error {
	secretMapFn := func(a client.Object) []reconcile.Request {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Name:      a.GetName(),
				Namespace: a.GetNamespace(),
			},
		},
		}
	}

	c, err := controller.New("secret", mgr, controller.Options{
		MaxConcurrentReconciles: 1,
		Reconciler:              sc,
	})
	if err != nil {
		return err
	}

	sc.recorder = mgr.GetEventRecorderFor(k8s.ControllerName)

	return c.Watch(&source.Channel{Source: secretEventChan}, handler.EnqueueRequestsFromMapFunc(secretMapFn))
}

func (sc *SecretController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rLog := log.FromContext(ctx).WithValues("name", req.NamespacedName, "kind", "Secret")
	rLog.Info("Secret event detected")
	secret, err := sc.doReconcile(ctx, req)
	errors2.HandleErrorWithObject(err, secret, sc.recorder)
	return errors2.HandleError(err, rLog)
}

func (sc *SecretController) doReconcile(ctx context.Context, req reconcile.Request) (*v1.Secret, error) {
	certs, err := sc.repo.LoadCertificates(ctx)
	if err != nil {
		return nil, err
	}

	certName := sc.names.Certificate(req.NamespacedName)
	cert := certs[certName]

	var secret v1.Secret
	err = sc.cli.Get(ctx, req.NamespacedName, &secret)
	if errors.IsNotFound(err) || secret.DeletionTimestamp != nil {
		if cert == nil {
			return nil, nil
		}

		return nil, sc.repo.DeleteCertificate(ctx, cert.Id)
	}
	if err != nil {
		return &secret, err
	}

	secretKey, err := convertKeyIfNeeded(secret.Data["tls.key"])
	if err != nil {
		return &secret, err
	}

	if cert == nil {
		return &secret, sc.repo.CreateCertificate(ctx, yc.Certificate{
			Name:  certName,
			Key:   secretKey,
			Chain: string(secret.Data["tls.crt"]),
		})
	}

	certData, err := sc.repo.LoadCertificateData(ctx, cert.Id)
	if err != nil {
		return &secret, err
	}

	if certNeedsUpdate(secret, certData) {
		return &secret, sc.repo.UpdateCertificate(ctx, yc.Certificate{
			ID:    cert.Id,
			Name:  cert.Name,
			Key:   secretKey,
			Chain: string(secret.Data["tls.crt"]),
		})
	}

	return &secret, nil
}

func certNeedsUpdate(secret v1.Secret, data *certificatemanager.GetCertificateContentResponse) bool {
	return string(secret.Data["tls.crt"]) != strings.Join(data.CertificateChain, "")
}

func convertKeyIfNeeded(bs []byte) (string, error) {
	block, _ := pem.Decode(bs)
	if block.Type == "PRIVATE KEY" {
		return string(bs), nil
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	encryptedKey, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}

	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: encryptedKey,
	})), nil
}
