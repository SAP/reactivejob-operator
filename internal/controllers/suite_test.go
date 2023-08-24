/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and reactivejob-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package controllers_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	batchv1alpha1 "github.com/sap/reactivejob-operator/api/v1alpha1"
	"github.com/sap/reactivejob-operator/internal/controllers"
	// +kubebuilder:scaffold:imports
)

func TestOperator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator")
}

var testEnv *envtest.Environment
var cfg *rest.Config
var cli client.Client
var ctx context.Context
var cancel context.CancelFunc
var threads sync.WaitGroup
var tmpdir string
var namespace string

var _ = BeforeSuite(func() {
	var err error

	By("initializing")
	log.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())
	tmpdir, err = os.MkdirTemp("", "")
	Expect(err).NotTo(HaveOccurred())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{"../../crds"},
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			ValidatingWebhooks: []*admissionv1.ValidatingWebhookConfiguration{
				buildValidatingWebhookConfiguration(),
			},
		},
	}
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	webhookInstallOptions := &testEnv.WebhookInstallOptions

	err = clientcmd.WriteToFile(*kubeConfigFromRestConfig(cfg), fmt.Sprintf("%s/kubeconfig", tmpdir))
	Expect(err).NotTo(HaveOccurred())
	fmt.Printf("A temporary kubeconfig for the envtest environment can be found here: %s/kubeconfig\n", tmpdir)

	By("populating scheme")
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(batchv1alpha1.AddToScheme(scheme))

	By("initializing client")
	cli, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	By("creating manager")
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&batchv1alpha1.ReactiveJob{},
					&batchv1.Job{},
				},
			},
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		HealthProbeBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	err = (&controllers.ReactiveJobReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	err = (&batchv1alpha1.ReactiveJob{}).SetupWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	By("starting dummy controller-manager")
	threads.Add(1)
	go func() {
		defer threads.Done()
		defer GinkgoRecover()
		// since there is no controller-manager in envtest, we have to fake jobs into a terminal state
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				jobList := &batchv1.JobList{}
				err := cli.List(context.Background(), jobList)
				Expect(err).NotTo(HaveOccurred())
				for _, job := range jobList.Items {
					if job.DeletionTimestamp.IsZero() {
						status := &job.Status
						oldStatus := status.DeepCopy()
						switch job.Annotations["testing.cs.sap.com/job-result"] {
						case "success":
							status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}
						case "failure":
							status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue}}
						}
						if reflect.DeepEqual(status, oldStatus) {
							continue
						}
						err = cli.Status().Update(context.Background(), &job)
						if apierrors.IsNotFound(err) || apierrors.IsConflict(err) {
							err = nil
						}
						Expect(err).NotTo(HaveOccurred())
					} else {
						if controllerutil.RemoveFinalizer(&job, metav1.FinalizerDeleteDependents) {
							err = cli.Update(context.Background(), &job)
							if apierrors.IsNotFound(err) || apierrors.IsConflict(err) {
								err = nil
							}
							Expect(err).NotTo(HaveOccurred())
						}
					}
				}
			}
		}
	}()

	By("starting manager")
	threads.Add(1)
	go func() {
		defer threads.Done()
		defer GinkgoRecover()
		err := mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	By("waiting for operator to become ready")
	Eventually(func() error { return mgr.GetWebhookServer().StartedChecker()(nil) }, "10s", "100ms").Should(Succeed())

	By("create testing namespace")
	namespace, err = createNamespace()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	threads.Wait()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
	err = os.RemoveAll(tmpdir)
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Create reactive jobs", func() {
	var reactiveJob *batchv1alpha1.ReactiveJob

	BeforeEach(func() {
		reactiveJob = &batchv1alpha1.ReactiveJob{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
			},
			Spec: batchv1alpha1.ReactiveJobSpec{
				JobTemplate: batchv1.JobTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"testing.cs.sap.com/test-id": uuid.New().String(),
						},
					},
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "run",
										Image: "alpine",
									},
								},
								RestartPolicy: corev1.RestartPolicyNever,
							},
						},
					},
				},
			},
		}
	})

	It("should create the job as specified (expecting it to succeed)", func() {
		reactiveJob.GenerateName = "test-"
		reactiveJob.Spec.JobTemplate.Annotations["testing.cs.sap.com/job-result"] = "success"
		err := cli.Create(ctx, reactiveJob)
		Expect(err).NotTo(HaveOccurred())
		waitForReactiveJobReady(reactiveJob)
		ensureJobSuccessful(reactiveJob)
	})

	It("should create the job as specified (expecting it to fail)", func() {
		reactiveJob.GenerateName = "test-"
		reactiveJob.Spec.JobTemplate.Annotations["testing.cs.sap.com/job-result"] = "failure"
		err := cli.Create(ctx, reactiveJob)
		Expect(err).NotTo(HaveOccurred())
		waitForReactiveJobError(reactiveJob)
		ensureJobFailed(reactiveJob)
	})
})

var _ = Describe("Update reactive jobs", func() {
	var reactiveJob *batchv1alpha1.ReactiveJob

	BeforeEach(func() {
		reactiveJob = &batchv1alpha1.ReactiveJob{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    namespace,
				GenerateName: "test-",
			},
			Spec: batchv1alpha1.ReactiveJobSpec{
				JobTemplate: batchv1.JobTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"testing.cs.sap.com/test-id":    uuid.New().String(),
							"testing.cs.sap.com/job-result": "success",
						},
					},
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "run",
										Image: "alpine",
									},
								},
								RestartPolicy: corev1.RestartPolicyNever,
							},
						},
					},
				},
			},
		}

		err := cli.Create(ctx, reactiveJob)
		Expect(err).NotTo(HaveOccurred())
		waitForReactiveJobReady(reactiveJob)
	})

	It("should re-execute the job as specified", func() {
		reactiveJob.Spec.JobTemplate.Annotations["testing.cs.sap.com/test-id"] = uuid.New().String()
		err := cli.Update(ctx, reactiveJob)
		Expect(err).NotTo(HaveOccurred())
		waitForReactiveJobReady(reactiveJob)
		ensureJobSuccessful(reactiveJob)
	})
})

var _ = Describe("Delete reactive jobs", func() {
	var reactiveJob *batchv1alpha1.ReactiveJob

	BeforeEach(func() {
		reactiveJob = &batchv1alpha1.ReactiveJob{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    namespace,
				GenerateName: "test-",
			},
			Spec: batchv1alpha1.ReactiveJobSpec{
				JobTemplate: batchv1.JobTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"testing.cs.sap.com/test-id":    uuid.New().String(),
							"testing.cs.sap.com/job-result": "success",
						},
					},
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "run",
										Image: "alpine",
									},
								},
								RestartPolicy: corev1.RestartPolicyNever,
							},
						},
					},
				},
			},
		}

		err := cli.Create(ctx, reactiveJob)
		Expect(err).NotTo(HaveOccurred())
		waitForReactiveJobReady(reactiveJob)
	})

	It("should delete the job", func() {
		err := cli.Delete(ctx, reactiveJob)
		Expect(err).NotTo(HaveOccurred())
		waitForReactiveJobGone(reactiveJob)
		ensureJobsGone(reactiveJob)
	})
})

func waitForReactiveJobReady(reactiveJob *batchv1alpha1.ReactiveJob) {
	Eventually(func() error {
		if err := cli.Get(ctx, types.NamespacedName{Namespace: reactiveJob.Namespace, Name: reactiveJob.Name}, reactiveJob); err != nil {
			return err
		}
		if reactiveJob.Status.ObservedGeneration != reactiveJob.Generation || reactiveJob.Status.State != batchv1alpha1.ReactiveJobStateReady {
			return fmt.Errorf("again")
		}
		return nil
	}, "10s", "100ms").Should(Succeed())
}

func waitForReactiveJobError(reactiveJob *batchv1alpha1.ReactiveJob) {
	Eventually(func() error {
		if err := cli.Get(ctx, types.NamespacedName{Namespace: reactiveJob.Namespace, Name: reactiveJob.Name}, reactiveJob); err != nil {
			return err
		}
		if reactiveJob.Status.ObservedGeneration != reactiveJob.Generation || reactiveJob.Status.State != batchv1alpha1.ReactiveJobStateError {
			return fmt.Errorf("again")
		}
		return nil
	}, "10s", "100ms").Should(Succeed())
}

func waitForReactiveJobGone(reactiveJob *batchv1alpha1.ReactiveJob) {
	Eventually(func() error {
		err := cli.Get(ctx, types.NamespacedName{Namespace: reactiveJob.Namespace, Name: reactiveJob.Name}, reactiveJob)
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		if err == nil {
			return fmt.Errorf("again")
		}
		return nil
	}, "10s", "100ms").Should(Succeed())
}

func ensureJobSuccessful(reactiveJob *batchv1alpha1.ReactiveJob) {
	job := &batchv1.Job{}
	err := cli.Get(ctx, types.NamespacedName{Namespace: namespace, Name: reactiveJob.Status.CurrentJobName}, job)
	Expect(err).NotTo(HaveOccurred())

	for key, value := range reactiveJob.Spec.JobTemplate.Annotations {
		Expect(job.Annotations).To(HaveKeyWithValue(key, value))
	}

	complete := false
	for _, cond := range job.Status.Conditions {
		if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
			complete = true
		}
	}
	Expect(complete).To(BeTrue())
}

func ensureJobFailed(reactiveJob *batchv1alpha1.ReactiveJob) {
	job := &batchv1.Job{}
	err := cli.Get(ctx, types.NamespacedName{Namespace: namespace, Name: reactiveJob.Status.CurrentJobName}, job)
	Expect(err).NotTo(HaveOccurred())

	for key, value := range reactiveJob.Spec.JobTemplate.Annotations {
		Expect(job.Annotations).To(HaveKeyWithValue(key, value))
	}

	failed := false
	for _, cond := range job.Status.Conditions {
		if cond.Type == batchv1.JobFailed && cond.Status == corev1.ConditionTrue {
			failed = true
		}
	}
	Expect(failed).To(BeTrue())
}

func ensureJobsGone(reactiveJob *batchv1alpha1.ReactiveJob) {
	Eventually(func() error {
		jobList := &batchv1.JobList{}
		err := cli.List(ctx, jobList, client.MatchingLabels{"batch.cs.sap.com/controller-uid": string(reactiveJob.UID)})
		if err != nil {
			return err
		}
		if len(jobList.Items) > 0 {
			return fmt.Errorf("again")
		}
		return nil
	}, "10s", "100ms").Should(Succeed())
}

// create namespace with a generated unique name
func createNamespace() (string, error) {
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "test-"}}
	if err := cli.Create(ctx, namespace); err != nil {
		return "", err
	}
	return namespace.Name, nil
}

// assemble validatingwebhookconfiguration descriptor
func buildValidatingWebhookConfiguration() *admissionv1.ValidatingWebhookConfiguration {
	return &admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "validate-reactivejob",
		},
		Webhooks: []admissionv1.ValidatingWebhook{{
			Name:                    "validate-reactivejob.test.local",
			AdmissionReviewVersions: []string{"v1"},
			ClientConfig: admissionv1.WebhookClientConfig{
				Service: &admissionv1.ServiceReference{
					Path: &[]string{fmt.Sprintf("/validate-%s-%s-%s", strings.ReplaceAll(batchv1alpha1.GroupVersion.Group, ".", "-"), batchv1alpha1.GroupVersion.Version, "reactivejob")}[0],
				},
			},
			Rules: []admissionv1.RuleWithOperations{{
				Operations: []admissionv1.OperationType{
					admissionv1.Create,
					admissionv1.Update,
					admissionv1.Delete,
				},
				Rule: admissionv1.Rule{
					APIGroups:   []string{batchv1alpha1.GroupVersion.Group},
					APIVersions: []string{batchv1alpha1.GroupVersion.Version},
					Resources:   []string{"reactivejobs"},
				},
			}},
			SideEffects: &[]admissionv1.SideEffectClass{admissionv1.SideEffectClassNone}[0],
		}},
	}
}

// convert rest.Config into kubeconfig
func kubeConfigFromRestConfig(restConfig *rest.Config) *clientcmdapi.Config {
	apiConfig := clientcmdapi.NewConfig()

	apiConfig.Clusters["envtest"] = clientcmdapi.NewCluster()
	cluster := apiConfig.Clusters["envtest"]
	cluster.Server = restConfig.Host
	cluster.CertificateAuthorityData = restConfig.CAData

	apiConfig.AuthInfos["envtest"] = clientcmdapi.NewAuthInfo()
	authInfo := apiConfig.AuthInfos["envtest"]
	authInfo.ClientKeyData = restConfig.KeyData
	authInfo.ClientCertificateData = restConfig.CertData

	apiConfig.Contexts["envtest"] = clientcmdapi.NewContext()
	context := apiConfig.Contexts["envtest"]
	context.Cluster = "envtest"
	context.AuthInfo = "envtest"

	apiConfig.CurrentContext = "envtest"

	return apiConfig
}
