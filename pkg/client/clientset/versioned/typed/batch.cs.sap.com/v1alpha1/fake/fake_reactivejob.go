/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and reactivejob-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/sap/reactivejob-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeReactiveJobs implements ReactiveJobInterface
type FakeReactiveJobs struct {
	Fake *FakeBatchV1alpha1
	ns   string
}

var reactivejobsResource = schema.GroupVersionResource{Group: "batch.cs.sap.com", Version: "v1alpha1", Resource: "reactivejobs"}

var reactivejobsKind = schema.GroupVersionKind{Group: "batch.cs.sap.com", Version: "v1alpha1", Kind: "ReactiveJob"}

// Get takes name of the reactiveJob, and returns the corresponding reactiveJob object, and an error if there is any.
func (c *FakeReactiveJobs) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ReactiveJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(reactivejobsResource, c.ns, name), &v1alpha1.ReactiveJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ReactiveJob), err
}

// List takes label and field selectors, and returns the list of ReactiveJobs that match those selectors.
func (c *FakeReactiveJobs) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ReactiveJobList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(reactivejobsResource, reactivejobsKind, c.ns, opts), &v1alpha1.ReactiveJobList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ReactiveJobList{ListMeta: obj.(*v1alpha1.ReactiveJobList).ListMeta}
	for _, item := range obj.(*v1alpha1.ReactiveJobList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested reactiveJobs.
func (c *FakeReactiveJobs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(reactivejobsResource, c.ns, opts))

}

// Create takes the representation of a reactiveJob and creates it.  Returns the server's representation of the reactiveJob, and an error, if there is any.
func (c *FakeReactiveJobs) Create(ctx context.Context, reactiveJob *v1alpha1.ReactiveJob, opts v1.CreateOptions) (result *v1alpha1.ReactiveJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(reactivejobsResource, c.ns, reactiveJob), &v1alpha1.ReactiveJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ReactiveJob), err
}

// Update takes the representation of a reactiveJob and updates it. Returns the server's representation of the reactiveJob, and an error, if there is any.
func (c *FakeReactiveJobs) Update(ctx context.Context, reactiveJob *v1alpha1.ReactiveJob, opts v1.UpdateOptions) (result *v1alpha1.ReactiveJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(reactivejobsResource, c.ns, reactiveJob), &v1alpha1.ReactiveJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ReactiveJob), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeReactiveJobs) UpdateStatus(ctx context.Context, reactiveJob *v1alpha1.ReactiveJob, opts v1.UpdateOptions) (*v1alpha1.ReactiveJob, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(reactivejobsResource, "status", c.ns, reactiveJob), &v1alpha1.ReactiveJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ReactiveJob), err
}

// Delete takes name of the reactiveJob and deletes it. Returns an error if one occurs.
func (c *FakeReactiveJobs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(reactivejobsResource, c.ns, name, opts), &v1alpha1.ReactiveJob{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeReactiveJobs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(reactivejobsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.ReactiveJobList{})
	return err
}

// Patch applies the patch and returns the patched reactiveJob.
func (c *FakeReactiveJobs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ReactiveJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(reactivejobsResource, c.ns, name, pt, data, subresources...), &v1alpha1.ReactiveJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ReactiveJob), err
}
