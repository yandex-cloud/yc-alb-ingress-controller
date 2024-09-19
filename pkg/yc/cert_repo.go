package yc

import (
	"context"
	"errors"
	"fmt"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/certificatemanager/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
)

const CertLabel = "alb-controller"

type Certificate struct {
	Chain string
	Key   string
	ID    string
	Name  string
}

type CertRepo interface {
	LoadCertificate(ctx context.Context, name string) (*certificatemanager.Certificate, error)
	LoadCertificates(ctx context.Context) (map[string]*certificatemanager.Certificate, error)
	LoadCertificateData(context.Context, string) (*certificatemanager.GetCertificateContentResponse, error)

	CreateCertificate(context.Context, Certificate) error
	UpdateCertificate(context.Context, Certificate) error
	DeleteCertificate(context.Context, string) error
}

type certRepo struct {
	sdk      *ycsdk.SDK
	folderID string
}

func NewCertRepo(sdk *ycsdk.SDK, folderID string) CertRepo {
	return &certRepo{
		sdk:      sdk,
		folderID: folderID,
	}
}

// LoadCertificate loads cert from cloud certificate manager. If there is no certificate with specified name, nil is returned
func (r *certRepo) LoadCertificate(ctx context.Context, name string) (*certificatemanager.Certificate, error) {
	certs, err := r.LoadCertificates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificates: %w", err)
	}

	return certs[name], nil
}

func (r *certRepo) LoadCertificates(ctx context.Context) (map[string]*certificatemanager.Certificate, error) {
	certs, err := r.sdk.Certificates().Certificate().List(ctx, &certificatemanager.ListCertificatesRequest{
		FolderId: r.folderID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %w", err)
	}

	result := make(map[string]*certificatemanager.Certificate)
	for _, cert := range certs.Certificates {
		if cert.Labels["yc-alb-ingress-controller"] != CertLabel {
			continue
		}

		if _, ok := result[cert.Name]; ok {
			return nil, errors.New("there is more than one certificate with specified name")
		}

		result[cert.Name] = cert
	}

	return result, nil
}

func (r *certRepo) LoadCertificateData(ctx context.Context, id string) (*certificatemanager.GetCertificateContentResponse, error) {
	data, err := r.sdk.CertificatesData().CertificateContent().Get(ctx, &certificatemanager.GetCertificateContentRequest{
		CertificateId: id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate data: %w", err)
	}

	return data, nil
}

func (r *certRepo) CreateCertificate(ctx context.Context, cert Certificate) error {
	_, err := r.sdk.Certificates().Certificate().Create(ctx, &certificatemanager.CreateCertificateRequest{
		Chain:      cert.Chain,
		PrivateKey: cert.Key,
		FolderId:   r.folderID,
		Name:       cert.Name,
		Labels: map[string]string{
			"yc-alb-ingress-controller": CertLabel,
		},
	})
	return err
}

func (r *certRepo) UpdateCertificate(ctx context.Context, cert Certificate) error {
	_, err := r.sdk.Certificates().Certificate().Update(ctx, &certificatemanager.UpdateCertificateRequest{
		Chain:         cert.Chain,
		CertificateId: cert.ID,
		PrivateKey:    cert.Key,
		Labels: map[string]string{
			"yc-alb-ingress-controller": CertLabel,
		},
	})
	return err
}

func (r *certRepo) DeleteCertificate(ctx context.Context, id string) error {
	_, err := r.sdk.Certificates().Certificate().Delete(ctx, &certificatemanager.DeleteCertificateRequest{
		CertificateId: id,
	})

	return err
}
