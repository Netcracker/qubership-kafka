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

package util

import (
	"github.com/sethvargo/go-password/password"
)

type OperatorPasswordGenerator struct {
	generator *password.Generator
}

func NewOperatorPasswordGenerator() (*OperatorPasswordGenerator, error) {
	generator, err := password.NewGenerator(&password.GeneratorInput{Symbols: "_!"})
	if err != nil {
		return nil, err
	}
	return &OperatorPasswordGenerator{generator: generator}, nil
}

func (operatorGenerator OperatorPasswordGenerator) Generate() (string, error) {
	return operatorGenerator.generator.Generate(10, 1, 1, false, false)
}
