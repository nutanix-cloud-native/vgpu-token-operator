// Copyright 2025 Nutanix. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

/*
Copyright 2025.

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

package controller

import (
	"testing"

	"github.com/nutanix-cloud-native/vgpu-token-operator/internal/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var TestEnv *helpers.TestEnvironment //nolint:gochecknoglobals //upstream test standard

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func(ctx SpecContext) {
	By("bootstrapping test environment")
	TestEnv = helpers.NewTestEnvironment()
	By("starting the manager")
	go func() {
		defer GinkgoRecover()
		Expect(TestEnv.StartManager()).To(Succeed())
	}()
})

var _ = AfterSuite(func() {
	if TestEnv != nil {
		By("tearing down the test environment")
		Expect(TestEnv.Stop()).To(Succeed())
	}
})
