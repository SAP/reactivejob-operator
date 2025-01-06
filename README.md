# Reactive Kubernetes Jobs

[![REUSE status](https://api.reuse.software/badge/github.com/SAP/reactivejob-operator)](https://api.reuse.software/info/github.com/SAP/reactivejob-operator)

## About this project

It is a common approach in Kubernetes to realize certain ad-hoc tasks as Kubernetes jobs.
However it is difficult to do this in a declarative way since Kubernetes job objects are more or less immutable.
That is, after being created, they run to completion once (successfully or with failure).
A natural expectation would be that the job's logic is re-run, whenever the job specification changes.
But this is not how Kubernetes jobs are designed, any such changes to the job's spec will be rejected by Kubernetes.
Workarounds for this problem are to create new jobs with new names again and again, or to delete and recreate the jobs (somehow making use of short job TTLs, or - when working with Helm - realizing jobs as Helm hooks).

This repository proposes a better solution for the above problem, in form of a custom resource type `reactivejobs.batch.cs.sap.com` (kind: `ReactiveJob`),
which basically wraps the builtin job resource (`jobs.batch`, kind `Job`), and re-triggers execution whenever the given job descriptor is changed.

A typical `ReactiveJob` could look as follows:

```yaml
apiVersion: batch.cs.sap.com/v1alpha1
kind: ReactiveJob
metadata:
  name: test
spec:
  jobTemplate:
    metadata:
      annotations:
        digest: abc123
    spec:
      ttlSecondsAfterFinished: 120
      template:
        spec:
          containers:
          - name: main
            image: alpine
            command: ["sleep","10"]
          restartPolicy: Never
```

Whenever the content under `spec.jobTemplate` changes, the controller provided by this repository will create a new `Job`, with a generated name, prefixed by the name of the `ReactiveJob`, such as `test-a7r3s`. The `ReactiveJob`'s status will reflect the status of the most recent dependent `Job`.

Generated jobs will not be deleted by default, but this can of course by achieved by specifying `spec.jobTemplate.spec.ttlSecondsAfterFinished` accordingly.
Note that this will only affect old jobs, the most recent job will never be deleted.

Finally, note that the `ReactiveJob`'s status is compatible with [kstatus](https://github.com/kubernetes-sigs/cli-utils/tree/master/pkg/kstatus), so it can be used e.g. with [flux kustomizations](https://fluxcd.io/docs/components/kustomize/kustomization/).

## Requirements and Setup

The recommended deployment method is to use the [Helm chart](https://github.com/sap/reactivejob-operator-helm):

```bash
helm upgrade -i reactivejob-operator oci://ghcr.io/sap/reactivejob-operator-helm/reactivejob-operator
```

## Documentation
 
The API reference is here: [https://pkg.go.dev/github.com/sap/reactivejob-operator](https://pkg.go.dev/github.com/sap/reactivejob-operator).

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/SAP/reactivejob-operator/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/SAP/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2025 SAP SE or an SAP affiliate company and reactivejob-operator contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/SAP/reactivejob-operator).
