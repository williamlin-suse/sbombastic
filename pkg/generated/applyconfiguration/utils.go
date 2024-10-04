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
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package applyconfiguration

import (
	v1alpha1 "github.com/rancher/sbombastic/api/storage/v1alpha1"
	internal "github.com/rancher/sbombastic/pkg/generated/applyconfiguration/internal"
	storagev1alpha1 "github.com/rancher/sbombastic/pkg/generated/applyconfiguration/storage/v1alpha1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	testing "k8s.io/client-go/testing"
)

// ForKind returns an apply configuration type for the given GroupVersionKind, or nil if no
// apply configuration type exists for the given GroupVersionKind.
func ForKind(kind schema.GroupVersionKind) interface{} {
	switch kind {
	// Group=storage.sbombastic.rancher.io, Version=v1alpha1
	case v1alpha1.SchemeGroupVersion.WithKind("ScanResult"):
		return &storagev1alpha1.ScanResultApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ScanResultSpec"):
		return &storagev1alpha1.ScanResultSpecApplyConfiguration{}

	}
	return nil
}

func NewTypeConverter(scheme *runtime.Scheme) *testing.TypeConverter {
	return &testing.TypeConverter{Scheme: scheme, TypeResolver: internal.Parser()}
}
