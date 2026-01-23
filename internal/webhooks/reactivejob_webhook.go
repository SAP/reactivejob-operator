/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and reactivejob-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package webhooks

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	batchv1 "k8s.io/api/batch/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	batchv1alpha1 "github.com/sap/reactivejob-operator/api/v1alpha1"
)

// +kubebuilder:webhook:path=/validate-batch-cs-sap-com-v1alpha1-reactivejob,mutating=false,failurePolicy=fail,sideEffects=None,groups=batch.cs.sap.com,resources=reactivejobs,verbs=create;update,versions=v1alpha1,name=vreactivejob.kb.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/mutate-batch-cs-sap-com-v1alpha1-reactivejob,mutating=true,failurePolicy=fail,sideEffects=None,groups=batch.cs.sap.com,resources=reactivejobs,verbs=create;update,versions=v1alpha1,name=mreactivejob.kb.io,admissionReviewVersions=v1

type ReactiveJobWebhook struct {
	Client client.Client
	Log    logr.Logger
}

var _ admission.Validator[*batchv1alpha1.ReactiveJob] = &ReactiveJobWebhook{}
var _ admission.Defaulter[*batchv1alpha1.ReactiveJob] = &ReactiveJobWebhook{}

func (w *ReactiveJobWebhook) ValidateCreate(ctx context.Context, r *batchv1alpha1.ReactiveJob) (admission.Warnings, error) {
	w.Log.Info("validate create", "name", r.Name)
	return nil, w.validate(r)
}

func (w *ReactiveJobWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj *batchv1alpha1.ReactiveJob) (admission.Warnings, error) {
	w.Log.Info("validate update", "name", newObj.Name)
	return nil, w.validate(newObj)
}

func (w *ReactiveJobWebhook) ValidateDelete(ctx context.Context, r *batchv1alpha1.ReactiveJob) (admission.Warnings, error) {
	w.Log.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (w *ReactiveJobWebhook) Default(ctx context.Context, r *batchv1alpha1.ReactiveJob) error {
	w.Log.Info("default", "name", r.Name)
	return nil
}

func (w *ReactiveJobWebhook) validate(r *batchv1alpha1.ReactiveJob) error {
	if !r.DeletionTimestamp.IsZero() {
		// Skip validation when object is being deleted e.g. while deleting the namespace,
		// the validation would prevent the deletion because of finalizer changes.
		return nil
	}
	if r.Spec.JobTemplate.Namespace != "" {
		return fmt.Errorf("specified job template is inavlid: namespace must not be specified")
	}
	if r.Spec.JobTemplate.Name != "" {
		return fmt.Errorf("specified job template is inavlid: name must not be specified")
	}
	if r.Spec.JobTemplate.GenerateName != "" {
		return fmt.Errorf("specified job template is inavlid: generateName must not be specified")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	job := &batchv1.Job{
		ObjectMeta: r.Spec.JobTemplate.ObjectMeta,
		Spec:       r.Spec.JobTemplate.Spec,
	}
	job.Namespace = r.Namespace
	job.Name = r.Name + "-xxxxx"
	err := client.NewDryRunClient(w.Client).Create(ctx, job)
	if err != nil {
		return fmt.Errorf("specified job template is inavlid; dry-run output was: %s", err)
	}
	return nil
}

func (w *ReactiveJobWebhook) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &batchv1alpha1.ReactiveJob{}).
		WithValidator(w).
		WithDefaulter(w).
		Complete()
}
