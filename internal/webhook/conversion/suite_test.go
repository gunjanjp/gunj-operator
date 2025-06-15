/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestMain(m *testing.M) {
	// Setup logging
	logf.SetLogger(zap.New(zap.UseDevMode(true)))
	klog.InitFlags(nil)

	// Run tests
	m.Run()
}

func init() {
	// Set up test environment paths
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "..", "config", "crd", "bases"),
		},
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{
				filepath.Join("..", "..", "..", "..", "config", "webhook"),
			},
		},
	}

	// Store test environment for use in BeforeSuite
	testEnvironment = testEnv
}

var testEnvironment *envtest.Environment
