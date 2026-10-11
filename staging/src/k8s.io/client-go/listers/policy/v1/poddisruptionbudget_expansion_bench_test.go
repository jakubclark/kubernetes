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
	"fmt"
	"testing"

	v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// BenchmarkGetPodPodDisruptionBudgets looks up the one PDB matching a pod among
// n PDBs in its namespace.
func BenchmarkGetPodPodDisruptionBudgets(b *testing.B) {
	for _, n := range []int{100, 1000, 4000, 8000} {
		b.Run(fmt.Sprintf("pdbs=%d", n), func(b *testing.B) {
			indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			for i := 0; i < n; i++ {
				pdb := &policy.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: fmt.Sprintf("pdb-%d", i)},
					Spec:       policy.PodDisruptionBudgetSpec{Selector: &metav1.LabelSelector{MatchLabels: benchMatchLabels(i)}},
				}
				if err := indexer.Add(pdb); err != nil {
					b.Fatal(err)
				}
			}
			lister := NewPodDisruptionBudgetLister(indexer)

			podLabels := benchMatchLabels(n / 2)
			podLabels["pod-template-hash"] = "5d8f9c7b6d"
			pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod", Labels: podLabels}}

			b.ReportAllocs()
			for b.Loop() {
				if _, err := lister.GetPodPodDisruptionBudgets(pod); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// benchMatchLabels returns PDB i's matchLabels: 6 labels, one shared by every
// PDB and one unique per PDB, with value lengths taken from a real namespace
// with ~3.2k PDBs.
func benchMatchLabels(i int) map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by": "op",                    // shared, sorts first
		"deployment-name-hash":         fmt.Sprintf("%032d", i), // unique per PDB
		"deployment-number":            fmt.Sprintf("%02d", i%100),
		"environment":                  fmt.Sprintf("env-%09d", i%389),
		"partition":                    fmt.Sprintf("partition-%021d", i%58),
		"service":                      fmt.Sprintf("service-%013d", i%1380),
	}
}
