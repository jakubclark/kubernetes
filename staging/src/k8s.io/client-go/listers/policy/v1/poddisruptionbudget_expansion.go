/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"fmt"

	"k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// PodDisruptionBudgetListerExpansion allows custom methods to be added to
// PodDisruptionBudgetLister.
type PodDisruptionBudgetListerExpansion interface {
	GetPodPodDisruptionBudgets(pod *v1.Pod) ([]*policy.PodDisruptionBudget, error)
}

// PodDisruptionBudgetNamespaceListerExpansion allows custom methods to be added to
// PodDisruptionBudgetNamespaceLister.
type PodDisruptionBudgetNamespaceListerExpansion interface{}

// GetPodPodDisruptionBudgets returns a list of PodDisruptionBudgets matching a pod.
func (s *podDisruptionBudgetLister) GetPodPodDisruptionBudgets(pod *v1.Pod) ([]*policy.PodDisruptionBudget, error) {
	list, err := s.PodDisruptionBudgets(pod.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	podLabels := labels.Set(pod.Labels)
	var pdbList []*policy.PodDisruptionBudget
	for i := range list {
		pdb := list[i]
		if !selectorMatches(pdb.Spec.Selector, podLabels) {
			continue
		}
		pdbList = append(pdbList, pdb)
	}

	if len(pdbList) == 0 {
		return nil, fmt.Errorf("could not find PodDisruptionBudget for pod %s in namespace %s with labels: %v", pod.Name, pod.Namespace, pod.Labels)
	}

	return pdbList, nil
}

// selectorMatches reports whether a PodDisruptionBudget selector matches the
// pod labels. A nil or invalid selector matches nothing, and unlike v1beta1,
// an empty selector matches everything.
//
// matchLabels are compared first, without parsing or allocating. Only selectors
// that pass are parsed and validated.
func selectorMatches(selector *metav1.LabelSelector, podLabels labels.Set) bool {
	if selector == nil {
		return false
	}
	// Prefilter only: matchLabels aren't validated yet, LabelSelectorAsSelector does that below.
	if !labels.ValidatedSetSelector(selector.MatchLabels).Matches(podLabels) {
		return false
	}
	s, err := metav1.LabelSelectorAsSelector(selector)
	return err == nil && s.Matches(podLabels)
}
