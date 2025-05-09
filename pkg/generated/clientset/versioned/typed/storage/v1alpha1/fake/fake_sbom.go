/*
Copyright 2024.

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
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/rancher/sbombastic/api/storage/v1alpha1"
	storagev1alpha1 "github.com/rancher/sbombastic/pkg/generated/applyconfiguration/storage/v1alpha1"
	typedstoragev1alpha1 "github.com/rancher/sbombastic/pkg/generated/clientset/versioned/typed/storage/v1alpha1"
	gentype "k8s.io/client-go/gentype"
)

// fakeSBOMs implements SBOMInterface
type fakeSBOMs struct {
	*gentype.FakeClientWithListAndApply[*v1alpha1.SBOM, *v1alpha1.SBOMList, *storagev1alpha1.SBOMApplyConfiguration]
	Fake *FakeStorageV1alpha1
}

func newFakeSBOMs(fake *FakeStorageV1alpha1, namespace string) typedstoragev1alpha1.SBOMInterface {
	return &fakeSBOMs{
		gentype.NewFakeClientWithListAndApply[*v1alpha1.SBOM, *v1alpha1.SBOMList, *storagev1alpha1.SBOMApplyConfiguration](
			fake.Fake,
			namespace,
			v1alpha1.SchemeGroupVersion.WithResource("sboms"),
			v1alpha1.SchemeGroupVersion.WithKind("SBOM"),
			func() *v1alpha1.SBOM { return &v1alpha1.SBOM{} },
			func() *v1alpha1.SBOMList { return &v1alpha1.SBOMList{} },
			func(dst, src *v1alpha1.SBOMList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.SBOMList) []*v1alpha1.SBOM { return gentype.ToPointerSlice(list.Items) },
			func(list *v1alpha1.SBOMList, items []*v1alpha1.SBOM) { list.Items = gentype.FromPointerSlice(items) },
		),
		fake,
	}
}
