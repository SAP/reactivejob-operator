/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and reactivejob-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/sap/go-generics/slices"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	batchv1alpha1 "github.com/sap/reactivejob-operator/api/v1alpha1"
)

const (
	finalizer = "batch.cs.sap.com/reactivejob-operator"
)

const (
	labelControllerUid             = "batch.cs.sap.com/controller-uid"
	labelControllerName            = "batch.cs.sap.com/controller-name"
	annotationControllerGeneration = "batch.cs.sap.com/controller-generation"
	annotationControllerHash       = "batch.cs.sap.com/controller-hash"
)

type ReactiveJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=batch.cs.sap.com,resources=reactivejobs,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=batch.cs.sap.com,resources=reactivejobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch.cs.sap.com,resources=reactivejobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

func (r *ReactiveJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	log := log.FromContext(ctx)
	log.V(1).Info("running reconcile")

	// Retrieve target ReactiveJob
	reactiveJob := &batchv1alpha1.ReactiveJob{}
	if err := r.Get(ctx, req.NamespacedName, reactiveJob); err != nil {
		if err := client.IgnoreNotFound(err); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "unexpected get error")
		}
		log.Info("not found; ignoring")
		return ctrl.Result{}, nil
	}

	// Call the defaulting webhook logic also here (because defaulting through the webhook might be incomplete in case of generateName usage)
	reactiveJob.Default()

	// Acknowledge observed generation
	reactiveJob.Status.ObservedGeneration = reactiveJob.Generation

	// Always attempt to update the status
	skipStatusUpdate := false
	defer func() {
		if skipStatusUpdate {
			return
		}
		if err != nil {
			reactiveJob.SetState(batchv1alpha1.ReactiveJobStateError, err.Error())
		}
		if updateErr := r.Status().Update(ctx, reactiveJob); updateErr != nil {
			err = utilerrors.NewAggregate([]error{err, updateErr})
			result = ctrl.Result{}
		}
	}()

	// Set a first status (and requeue, because the status update itself will not trigger another reconciliation because of the event filter set)
	if reactiveJob.Status.State == "" {
		reactiveJob.SetState(batchv1alpha1.ReactiveJobStateNew, "First seen")
		return ctrl.Result{Requeue: true}, nil
	}

	// Find dependent job resources
	jobList := &batchv1.JobList{}
	if err := r.Client.List(
		ctx,
		jobList,
		&client.ListOptions{Namespace: req.Namespace},
		client.MatchingLabels{labelControllerUid: string(reactiveJob.UID)},
	); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to list dependent job resources")
	}
	jobs := slices.SortBy(jobList.Items, func(x, y batchv1.Job) bool {
		return mustAtoi(x.Annotations[annotationControllerGeneration]) < mustAtoi(y.Annotations[annotationControllerGeneration])
	})

	// Do the reconciliation
	if reactiveJob.DeletionTimestamp.IsZero() {
		// Create/update case
		if !slices.Contains(reactiveJob.Finalizers, finalizer) {
			controllerutil.AddFinalizer(reactiveJob, finalizer)
			if err := r.Update(ctx, reactiveJob); err != nil {
				return ctrl.Result{}, errors.Wrap(err, "error setting finalizer")
			}
		}

		// TODO: use something better than jsonHash
		jobTemplateHash := jsonHash(reactiveJob.Spec.JobTemplate)
		if len(jobs) == 0 || jobs[0].Annotations[annotationControllerHash] != jobTemplateHash {
			job := &batchv1.Job{
				ObjectMeta: reactiveJob.Spec.JobTemplate.ObjectMeta,
				Spec:       reactiveJob.Spec.JobTemplate.Spec,
			}
			// TODO: should we fail if job.Namespace, job.Name, job.GenerateName are not empty ?
			job.Namespace = req.Namespace
			job.Name = ""
			job.GenerateName = reactiveJob.Name + "-"
			if job.Labels == nil {
				job.Labels = make(map[string]string)
			}
			if job.Annotations == nil {
				job.Annotations = make(map[string]string)
			}
			job.Labels[labelControllerUid] = string(reactiveJob.UID)
			job.Labels[labelControllerName] = reactiveJob.Name
			job.Annotations[annotationControllerGeneration] = strconv.FormatInt(reactiveJob.Generation, 10)
			job.Annotations[annotationControllerHash] = jobTemplateHash
			controllerutil.AddFinalizer(job, finalizer)
			if err := r.Client.Create(ctx, job, &client.CreateOptions{}); err != nil {
				return ctrl.Result{}, errors.Wrap(err, "failed to create dependent job")
			}
			reactiveJob.SetState(batchv1alpha1.ReactiveJobStateProcessing, "job created")
			return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
		}

		for _, job := range jobs[1:] {
			if slices.Contains(job.Finalizers, finalizer) {
				controllerutil.RemoveFinalizer(&job, finalizer)
				if err := r.Client.Update(ctx, &job, &client.UpdateOptions{}); err != nil {
					return ctrl.Result{}, errors.Wrap(err, "error removing finalizer from previous dependent job")
				}
			}
		}

		job := jobs[0]
		reactiveJob.Status.CurrentJobName = job.Name
		state := batchv1alpha1.ReactiveJobStateProcessing
		message := "waiting for job to complete"
		requeueAfter := 10 * time.Second
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobFailed && cond.Status == corev1.ConditionTrue {
				state = batchv1alpha1.ReactiveJobStateError
				message = fmt.Sprintf("%s: %s", cond.Reason, cond.Message)
				requeueAfter = 1 * time.Minute
				break
			} else if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
				state = batchv1alpha1.ReactiveJobStateReady
				message = fmt.Sprintf("%s: %s", cond.Reason, cond.Message)
				requeueAfter = 10 * time.Minute
				break
			}
		}
		reactiveJob.SetState(state, message)
		// TODO: apply some increasing period, depending on the age of the last update
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	} else if len(slices.Remove(reactiveJob.Finalizers, finalizer)) > 0 {
		reactiveJob.SetState(batchv1alpha1.ReactiveJobStateDeletionBlocked, "Deletion blocked due to foreign finalizers")
		// TODO: apply some increasing period, depending on the age of the last update
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	} else {
		// Deletion case
		if len(jobs) == 0 {
			if slices.Contains(reactiveJob.Finalizers, finalizer) {
				controllerutil.RemoveFinalizer(reactiveJob, finalizer)
				if err := r.Update(ctx, reactiveJob); err != nil {
					return ctrl.Result{}, errors.Wrap(err, "error clearing finalizer")
				}
			}
			// skip status update, since the resource will anyway deleted timely by the API server
			// this will suppress unnecessary ugly 409'ish error messages in the logs
			// (occurring in the case that API server would delete the resource in the course of the subsequent reconciliation)
			skipStatusUpdate = true
			return ctrl.Result{}, nil
		} else {
			for _, job := range jobs {
				if slices.Contains(job.Finalizers, finalizer) {
					controllerutil.RemoveFinalizer(&job, finalizer)
					if err := r.Client.Update(ctx, &job, &client.UpdateOptions{}); err != nil {
						return ctrl.Result{}, errors.Wrap(err, "error removing finalizer from previous dependent job")
					}
				}
				if job.DeletionTimestamp.IsZero() {
					if err := r.Client.Delete(ctx, &job, &client.DeleteOptions{PropagationPolicy: &[]metav1.DeletionPropagation{metav1.DeletePropagationForeground}[0]}); err != nil {
						return ctrl.Result{}, errors.Wrap(err, "failed to delete dependent job resources")
					}
				}
			}
			reactiveJob.SetState(batchv1alpha1.ReactiveJobStateDeleting, "waiting for deletion of dependent jobs")
			// TODO: apply some increasing period, depending on the age of the last update
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}
}

func jobEnqueueMapper(ctx context.Context, obj client.Object) []reconcile.Request {
	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Namespace: obj.GetNamespace(),
				Name:      obj.GetLabels()[labelControllerName],
			},
		},
	}
}

func (r *ReactiveJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	reactiveJobPredicate := predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{})
	jobPredicate, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{Key: labelControllerName, Operator: metav1.LabelSelectorOpExists}}})
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1alpha1.ReactiveJob{}, builder.WithPredicates(reactiveJobPredicate)).
		Watches(&batchv1.Job{}, handler.EnqueueRequestsFromMapFunc(jobEnqueueMapper), builder.WithPredicates(jobPredicate)).
		Complete(r)
}
