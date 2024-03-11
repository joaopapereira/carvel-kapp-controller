// Copyright 2024 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
package packageinstall_test

import (
	"testing"

	"github.com/k14s/semver/v4"
	"github.com/stretchr/testify/require"
	datapkgingv1alpha1 "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apiserver/apis/datapackaging/v1alpha1"
	fakeapiserver "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apiserver/client/clientset/versioned/fake"
	versions "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions/v1alpha1"

	"github.com/vmware-tanzu/carvel-kapp-controller/pkg/packageinstall"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func Test_PackageFinder_Find(t *testing.T) {
	namespace := "default-ns"
	serviceaccount := "use-local-cluster-sa"
	testCases := []struct {
		name       string
		kcVersion  string
		k8sVersion string
		test       func(t *testing.T, subject packageinstall.Finder, client *fakeapiserver.Clientset)
	}{
		{
			name: "Find Package with given semver config successful",
			test: func(t *testing.T, subject packageinstall.Finder, client *fakeapiserver.Clientset) {
				expected := createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "1.0.0",
					},
				)
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						name:    "other-package",
						ns:      namespace,
						version: "1.0.0",
					},
				)

				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				pkgRes, err := subject.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: "1.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, expected, pkgRes)
			},
		},
		{
			name: "When package does not exists returns Package not found error",
			test: func(t *testing.T, subject packageinstall.Finder, _ *fakeapiserver.Clientset) {
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				pkgRes, err := subject.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: "1.0.0",
				})
				require.ErrorContains(t, err, "Package parent-pkg not found")
				require.Nil(t, pkgRes)
			},
		},
		{
			name: "When package does not exists in the correct namespace returns Package not found error",
			test: func(t *testing.T, subject packageinstall.Finder, client *fakeapiserver.Clientset) {
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      "other-namespace",
						version: "1.0.0",
					},
				)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				pkgRes, err := subject.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: "1.0.0",
				})
				require.ErrorContains(t, err, "Package parent-pkg not found")
				require.Nil(t, pkgRes)
			},
		},
		{
			name: "Find Package with given semver config successful - specific version of the given package is chosen",
			test: func(t *testing.T, subject packageinstall.Finder, client *fakeapiserver.Clientset) {
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "1.0.0",
					},
				)
				expected := createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "2.0.0",
					},
				)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				pkgRes, err := subject.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: "2.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, expected, pkgRes)
			},
		},
		{
			name: "Find Package with given semver config successful - Highest version is with given constraint is chosen when range provided",
			test: func(t *testing.T, subject packageinstall.Finder, client *fakeapiserver.Clientset) {
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "1.0.0",
					},
				)
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "2.0.0",
					},
				)
				expected := createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "3.0.0",
					},
				)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				pkgRes, err := subject.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: ">=1.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, expected, pkgRes)
			},
		},
		{
			name: "Find Package with given semver config successful - Highest version in the range is chosen",
			test: func(t *testing.T, subject packageinstall.Finder, client *fakeapiserver.Clientset) {
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "1.0.0",
					},
				)
				expected := createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "2.0.0",
					},
				)
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "3.0.0",
					},
				)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				pkgRes, err := subject.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: ">=1.0.0 <3.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, expected, pkgRes)
			},
		},
		{
			name: "Find Package with given semver config successful - Highest version in the range is chosen",
			test: func(t *testing.T, subject packageinstall.Finder, client *fakeapiserver.Clientset) {
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "1.0.0",
					},
				)
				expected := createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "2.0.0",
					},
				)
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "3.0.0",
					},
				)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				pkgRes, err := subject.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: ">=1.0.0 <3.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, expected, pkgRes)
			},
		},
		{
			name: "Find Package with given semver config fails - no version in the range is found",
			test: func(t *testing.T, subject packageinstall.Finder, client *fakeapiserver.Clientset) {
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "1.0.0",
					},
				)
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:      namespace,
						version: "2.0.0",
					},
				)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				pkgRes, err := subject.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: ">2.0.0",
				})
				require.ErrorContains(t, err, "Expected to find at least one version, but did not (details: all=2 -> after-prereleases-filter=2 -> after-kapp-controller-version-check=2 -> after-constraints-filter=0)")
				require.Nil(t, pkgRes)
			},
		},
		{
			name:      "Find Package with given semver config successful - Highest version in the range that satisfies kc constraint is chosen",
			kcVersion: "1.5.0",
			test: func(t *testing.T, subject packageinstall.Finder, client *fakeapiserver.Clientset) {
				expected := createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:           namespace,
						version:      "1.0.0",
						kcConstraint: ">=1.5.0",
					},
				)
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:           namespace,
						version:      "2.0.0",
						kcConstraint: ">1.6.0",
					},
				)
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:           namespace,
						version:      "3.0.0",
						kcConstraint: ">1.7.0",
					},
				)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				pkgRes, err := subject.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: "<3.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, expected, pkgRes)
			},
		},
		{
			name:      "Ignore kcversion constraint when PackageInstall have annotation 'packaging.carvel.dev/ignore-kapp-controller-version-selection'",
			kcVersion: "1.25.0",
			test: func(t *testing.T, subject packageinstall.Finder, client *fakeapiserver.Clientset) {
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:           namespace,
						version:      "1.0.0",
						kcConstraint: ">=1.5.0",
					},
				)
				expected := createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:           namespace,
						version:      "2.0.0",
						kcConstraint: ">1.6.0",
					},
				)
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:           namespace,
						version:      "3.0.0",
						kcConstraint: ">1.7.0",
					},
				)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				model.Annotations = map[string]string{"packaging.carvel.dev/ignore-kapp-controller-version-selection'": ""}
				pkgRes, err := subject.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: "<3.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, expected, pkgRes)
			},
		},
		{
			name:       "Find Package with given semver config successful - Highest version in the range that satisfies k8s constraint is chosen",
			k8sVersion: "1.25.0",
			test: func(t *testing.T, subject packageinstall.Finder, client *fakeapiserver.Clientset) {
				expected := createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:            namespace,
						version:       "1.0.0",
						k8sConstraint: ">=1.25.0",
					},
				)
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:            namespace,
						version:       "2.0.0",
						k8sConstraint: ">1.26.0",
					},
				)
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:            namespace,
						version:       "3.0.0",
						k8sConstraint: ">1.27.0",
					},
				)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				pkgRes, err := subject.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: "<3.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, expected, pkgRes)
			},
		},
		{
			name:       "Ignore k8sversion constraint when PackageInstall have annotation 'packaging.carvel.dev/ignore-kubernetes-version-selection'",
			k8sVersion: "1.25.0",
			test: func(t *testing.T, subject packageinstall.Finder, client *fakeapiserver.Clientset) {
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:            namespace,
						version:       "1.0.0",
						k8sConstraint: ">=1.25.0",
					},
				)
				expected := createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:            namespace,
						version:       "2.0.0",
						k8sConstraint: ">1.26.0",
					},
				)
				createPackageWithConstraints(t, client,
					pkgWithConstraints{
						ns:            namespace,
						version:       "3.0.0",
						k8sConstraint: ">1.27.0",
					},
				)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				model.Annotations = map[string]string{"packaging.carvel.dev/ignore-kubernetes-version-selection": ""}
				pkgRes, err := subject.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: "<3.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, expected, pkgRes)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakePkgClient := fakeapiserver.NewSimpleClientset()
			log := logf.Log.WithName("kc")
			compInfo := packageinstall.FakeComponentInfo{}
			if tc.kcVersion != "" {
				compInfo.KCVersion = semver.MustParse(tc.kcVersion)
			}
			if tc.k8sVersion != "" {
				compInfo.K8sVersion = semver.MustParse(tc.k8sVersion)
			}
			finder := packageinstall.NewPackageFinder(log, fakePkgClient, compInfo)
			tc.test(t, finder, fakePkgClient)
		})
	}

}

type pkgWithConstraints struct {
	name          string
	ns            string
	version       string
	dependencies  []*datapkgingv1alpha1.Dependency
	kcConstraint  string
	k8sConstraint string
}

func createPackageWithConstraints(t *testing.T, c *fakeapiserver.Clientset, model pkgWithConstraints) *datapkgingv1alpha1.Package {
	t.Helper()
	name := "parent-pkg"
	if model.name != "" {
		name = model.name
	}
	pkg := generatePackageWithConstraints(name, model.ns, model.version, model.dependencies, model.kcConstraint, model.k8sConstraint)
	require.NoError(t, c.Tracker().Add(pkg))
	return pkg
}
