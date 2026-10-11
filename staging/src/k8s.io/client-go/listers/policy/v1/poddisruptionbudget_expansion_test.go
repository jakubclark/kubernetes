/*
Copyright The Kubernetes Authors.

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
	"testing"

	v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

func TestSelectorMatches(t *testing.T) {
	fooExists := []metav1.LabelSelectorRequirement{{Key: "foo", Operator: metav1.LabelSelectorOpExists}}
	invalidOp := []metav1.LabelSelectorRequirement{{Key: "foo", Operator: "bogus"}}

	tests := []struct {
		name      string
		selector  *metav1.LabelSelector
		podLabels labels.Set
		want      bool
	}{
		{"nil selector", nil, labels.Set{"foo": "bar"}, false},
		{"empty selector", &metav1.LabelSelector{}, labels.Set{"foo": "bar"}, true},
		{"empty selector, unlabeled pod", &metav1.LabelSelector{}, nil, true},
		{"matchLabels match", &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}, labels.Set{"foo": "bar", "x": "y"}, true},
		{"matchLabels wrong value", &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}, labels.Set{"foo": "baz"}, false},
		{"matchLabels missing key", &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}, labels.Set{"x": "y"}, false},
		{"matchLabels empty value", &metav1.LabelSelector{MatchLabels: map[string]string{"foo": ""}}, labels.Set{"foo": ""}, true},
		{"matchLabels empty value, missing key", &metav1.LabelSelector{MatchLabels: map[string]string{"foo": ""}}, nil, false},
		{"matchExpressions match", &metav1.LabelSelector{MatchExpressions: fooExists}, labels.Set{"foo": "bar"}, true},
		{"matchExpressions no match", &metav1.LabelSelector{MatchExpressions: fooExists}, labels.Set{"x": "y"}, false},
		{"matchLabels and matchExpressions match", &metav1.LabelSelector{MatchLabels: map[string]string{"x": "y"}, MatchExpressions: fooExists}, labels.Set{"foo": "bar", "x": "y"}, true},
		{"matchLabels match, matchExpressions don't", &metav1.LabelSelector{MatchLabels: map[string]string{"x": "y"}, MatchExpressions: fooExists}, labels.Set{"x": "y"}, false},
		{"invalid operator", &metav1.LabelSelector{MatchExpressions: invalidOp}, labels.Set{"foo": "bar"}, false},
		{"matchLabels match, invalid operator", &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}, MatchExpressions: invalidOp}, labels.Set{"foo": "bar"}, false},
		{"invalid label value", &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar baz"}}, labels.Set{"foo": "bar baz"}, false},
		{"invalid label key", &metav1.LabelSelector{MatchLabels: map[string]string{"bad key": "x"}}, labels.Set{"bad key": "x"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := selectorMatches(tc.selector, tc.podLabels); got != tc.want {
				t.Errorf("selectorMatches() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGetPodPodDisruptionBudgets(t *testing.T) {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	for _, pdb := range []*policy.PodDisruptionBudget{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "match"}, Spec: policy.PodDisruptionBudgetSpec{Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}}},
		{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "no-match"}, Spec: policy.PodDisruptionBudgetSpec{Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "baz"}}}},
		{ObjectMeta: metav1.ObjectMeta{Namespace: "other", Name: "other-namespace"}, Spec: policy.PodDisruptionBudgetSpec{Selector: &metav1.LabelSelector{}}},
	} {
		if err := indexer.Add(pdb); err != nil {
			t.Fatal(err)
		}
	}
	lister := NewPodDisruptionBudgetLister(indexer)

	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod", Labels: map[string]string{"foo": "bar"}}}
	pdbs, err := lister.GetPodPodDisruptionBudgets(pod)
	if err != nil {
		t.Fatal(err)
	}
	if len(pdbs) != 1 || pdbs[0].Name != "match" {
		t.Errorf("got %d PodDisruptionBudgets, want only %q", len(pdbs), "match")
	}

	pod.Labels = map[string]string{"foo": "qux"}
	if _, err := lister.GetPodPodDisruptionBudgets(pod); err == nil {
		t.Error("want an error when no PodDisruptionBudget matches")
	}
}
