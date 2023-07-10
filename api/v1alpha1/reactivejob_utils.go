/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and reactivejob-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Set state (and the 'Ready' condition) of a ReactiveJob
func (reactiveJob *ReactiveJob) SetState(state ReactiveJobState, message string) {
	conditionStatus := corev1.ConditionUnknown

	switch state {
	case ReactiveJobStateReady:
		conditionStatus = corev1.ConditionTrue
	case ReactiveJobStateError:
		conditionStatus = corev1.ConditionFalse
	}

	setCondition(&reactiveJob.Status.Conditions, ReactiveJobConditionTypeReady, conditionStatus, string(state), message)
	reactiveJob.Status.State = state
}

func getCondition(conditions []ReactiveJobCondition, conditionType ReactiveJobConditionType) *ReactiveJobCondition {
	for i := 0; i < len(conditions); i++ {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

func setCondition(conditions *[]ReactiveJobCondition, conditionType ReactiveJobConditionType, conditionStatus corev1.ConditionStatus, conditionReason string, conditionMessage string) {
	now := metav1.Now()

	cond := getCondition(*conditions, conditionType)

	if cond == nil {
		*conditions = append(*conditions, ReactiveJobCondition{Type: conditionType, LastTransitionTime: &now})
		cond = &(*conditions)[len(*conditions)-1]
	} else if cond.Status != conditionStatus {
		cond.LastTransitionTime = &now
	}
	cond.LastUpdateTime = &now
	cond.Status = conditionStatus
	cond.Reason = conditionReason
	cond.Message = conditionMessage
}
