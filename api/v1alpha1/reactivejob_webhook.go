/*
SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and reactivejob-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var reactivejoblog = logf.Log.WithName("reactivejob-resource")
var reactivejobclient client.Client

func (r *ReactiveJob) SetupWebhookWithManager(mgr ctrl.Manager) error {
	reactivejobclient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-batch-cs-sap-com-v1alpha1-reactivejob,mutating=true,failurePolicy=fail,sideEffects=None,groups=batch.cs.sap.com,resources=reactivejobs,verbs=create;update,versions=v1alpha1,name=mreactivejob.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ReactiveJob{}

func (r *ReactiveJob) Default() {
	reactivejoblog.Info("default", "name", r.Name)
}

// +kubebuilder:webhook:path=/validate-batch-cs-sap-com-v1alpha1-reactivejob,mutating=false,failurePolicy=fail,sideEffects=None,groups=batch.cs.sap.com,resources=reactivejobs,verbs=create;update,versions=v1alpha1,name=vreactivejob.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ReactiveJob{}

func (r *ReactiveJob) ValidateCreate() (admission.Warnings, error) {
	reactivejoblog.Info("validate create", "name", r.Name)
	return nil, r.validate()
}

func (r *ReactiveJob) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	reactivejoblog.Info("validate update", "name", r.Name)
	return nil, r.validate()
}

func (r *ReactiveJob) ValidateDelete() (admission.Warnings, error) {
	reactivejoblog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *ReactiveJob) validate() error {
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
	err := client.NewDryRunClient(reactivejobclient).Create(ctx, job)
	if err != nil {
		return fmt.Errorf("specified job template is inavlid; dry-run output was: %s", err)
	}
	return nil
}
