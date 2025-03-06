// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestClient struct {
	ConfigMapContent string
	SecretData       map[string]string
}

func (tc TestClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	switch o := obj.(type) {
	case *v1.ConfigMap:
		m := make(map[string]string)
		m["config"] = tc.ConfigMapContent
		o.Data = m
	case *v1.Secret:
		o.Data = make(map[string][]byte)
		for key, value := range tc.SecretData {
			o.Data[key] = []byte(value)
		}
	default:
		return nil
	}
	return nil
}

func (tc TestClient) List(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
	return nil
}

func (tc TestClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return nil
}

func (tc TestClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return nil
}

func (tc TestClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return nil
}

func (tc TestClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

func (tc TestClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}

func (tc TestClient) Status() client.StatusWriter {
	return nil
}

func (tc TestClient) Scheme() *runtime.Scheme {
	return nil
}

func (tc TestClient) RESTMapper() meta.RESTMapper {
	return nil
}
