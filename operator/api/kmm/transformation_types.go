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

// +kubebuilder:object:generate=true
package kmm

// Transformation shows transformation configuration
type Transformation struct {
	Transforms []Transform `json:"transforms"`
	Predicates []Predicate `json:"predicates,omitempty"`
}

// Transform shows transform configuration
type Transform struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:validation:MinLength=1
	Type      string            `json:"type"`
	Predicate string            `json:"predicate,omitempty"`
	Negate    *bool             `json:"negate,omitempty"`
	Params    map[string]string `json:"params,omitempty"`
}

// Predicate shows predicate configuration
type Predicate struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:validation:MinLength=1
	Type   string            `json:"type"`
	Params map[string]string `json:"params,omitempty"`
}
