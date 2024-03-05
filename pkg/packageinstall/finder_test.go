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
		name string
		test func(t *testing.T)
	}{
		{
			name: "Find Package with given semver config successful",
			test: func(t *testing.T) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				fakePkgClient := fakeapiserver.NewSimpleClientset(pkg)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				log := logf.Log.WithName("kc")
				finder := packageinstall.NewPackageFinder(log, fakePkgClient, packageinstall.FakeComponentInfo{KCVersion: semver.MustParse("0.42.31337")})
				pkgRes, err := finder.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: "1.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, pkg, pkgRes)
			},
		},
		{
			name: "Find Package with given semver config fails - Package not found",
			test: func(t *testing.T) {
				fakePkgClient := fakeapiserver.NewSimpleClientset()
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				log := logf.Log.WithName("kc")
				finder := packageinstall.NewPackageFinder(log, fakePkgClient, packageinstall.FakeComponentInfo{KCVersion: semver.MustParse("0.42.31337")})
				pkgRes, err := finder.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: "1.0.0",
				})
				require.ErrorContains(t, err, "Package parent-pkg not found")
				require.Nil(t, pkgRes)
			},
		},
		{
			name: "Find Package with given semver config successful - specific version of the given package is choosen",
			test: func(t *testing.T) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				pkg2 := generatePackageWithConstraints(
					"parent-pkg", namespace, "2.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				fakePkgClient := fakeapiserver.NewSimpleClientset(pkg, pkg2)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				log := logf.Log.WithName("kc")
				finder := packageinstall.NewPackageFinder(log, fakePkgClient, packageinstall.FakeComponentInfo{KCVersion: semver.MustParse("0.42.31337")})
				pkgRes, err := finder.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: "2.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, pkg2, pkgRes)
			},
		},
		{
			name: "Find Package with given semver config successful - Highest version is with given constraint is choosen",
			test: func(t *testing.T) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				pkg2 := generatePackageWithConstraints(
					"parent-pkg", namespace, "2.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				pkg3 := generatePackageWithConstraints(
					"parent-pkg", namespace, "3.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				fakePkgClient := fakeapiserver.NewSimpleClientset(pkg, pkg2, pkg3)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				log := logf.Log.WithName("kc")
				finder := packageinstall.NewPackageFinder(log, fakePkgClient, packageinstall.FakeComponentInfo{KCVersion: semver.MustParse("0.42.31337")})
				pkgRes, err := finder.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: ">=1.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, pkg3, pkgRes)
			},
		},
		{
			name: "Find Package with given semver config successful - Highest version in the range is choosen",
			test: func(t *testing.T) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				pkg2 := generatePackageWithConstraints(
					"parent-pkg", namespace, "2.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				pkg3 := generatePackageWithConstraints(
					"parent-pkg", namespace, "3.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				fakePkgClient := fakeapiserver.NewSimpleClientset(pkg, pkg2, pkg3)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				log := logf.Log.WithName("kc")
				finder := packageinstall.NewPackageFinder(log, fakePkgClient, packageinstall.FakeComponentInfo{KCVersion: semver.MustParse("0.42.31337")})
				pkgRes, err := finder.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: ">=1.0.0 <3.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, pkg2, pkgRes)
			},
		},
		{
			name: "Find Package with given semver config successful - Highest version in the range is choosen",
			test: func(t *testing.T) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				pkg2 := generatePackageWithConstraints(
					"parent-pkg", namespace, "2.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				pkg3 := generatePackageWithConstraints(
					"parent-pkg", namespace, "3.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				fakePkgClient := fakeapiserver.NewSimpleClientset(pkg, pkg2, pkg3)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				log := logf.Log.WithName("kc")
				finder := packageinstall.NewPackageFinder(log, fakePkgClient, packageinstall.FakeComponentInfo{KCVersion: semver.MustParse("0.42.31337")})
				pkgRes, err := finder.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: ">=1.0.0 <3.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, pkg2, pkgRes)
			},
		},
		{
			name: "Find Package with given semver config fails - no version in the range is found",
			test: func(t *testing.T) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				pkg2 := generatePackageWithConstraints(
					"parent-pkg", namespace, "2.0.0", []datapkgingv1alpha1.Dependency{},
					"", "",
				)
				fakePkgClient := fakeapiserver.NewSimpleClientset(pkg, pkg2)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				log := logf.Log.WithName("kc")
				finder := packageinstall.NewPackageFinder(log, fakePkgClient, packageinstall.FakeComponentInfo{KCVersion: semver.MustParse("0.42.31337")})
				pkgRes, err := finder.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: ">2.0.0",
				})
				require.ErrorContains(t, err, "Expected to find at least one version, but did not (details: all=2 -> after-prereleases-filter=2 -> after-kapp-controller-version-check=2 -> after-constraints-filter=0)")
				require.Nil(t, pkgRes)
			},
		},
		{
			name: "Find Package with given semver config successful - Highest version in the range that satifies kc constraint is choosen",
			test: func(t *testing.T) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{},
					">=1.5.0", "",
				)
				pkg2 := generatePackageWithConstraints(
					"parent-pkg", namespace, "2.0.0", []datapkgingv1alpha1.Dependency{},
					">1.6.0", "",
				)
				pkg3 := generatePackageWithConstraints(
					"parent-pkg", namespace, "3.0.0", []datapkgingv1alpha1.Dependency{},
					">1.7.0", "",
				)
				fakePkgClient := fakeapiserver.NewSimpleClientset(pkg, pkg2, pkg3)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				log := logf.Log.WithName("kc")
				finder := packageinstall.NewPackageFinder(log, fakePkgClient, packageinstall.FakeComponentInfo{KCVersion: semver.MustParse("1.5.0")})
				pkgRes, err := finder.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: "<3.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, pkg, pkgRes)
			},
		},
		{
			name: "Find Package with given semver config successful - Highest version in the range that satifies k8s constraint is choosen",
			test: func(t *testing.T) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{},
					"", ">=1.25.0",
				)
				pkg2 := generatePackageWithConstraints(
					"parent-pkg", namespace, "2.0.0", []datapkgingv1alpha1.Dependency{},
					"", ">1.26.0",
				)
				pkg3 := generatePackageWithConstraints(
					"parent-pkg", namespace, "3.0.0", []datapkgingv1alpha1.Dependency{},
					"", ">1.27.0",
				)
				fakePkgClient := fakeapiserver.NewSimpleClientset(pkg, pkg2, pkg3)
				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				log := logf.Log.WithName("kc")
				finder := packageinstall.NewPackageFinder(log, fakePkgClient, packageinstall.FakeComponentInfo{K8sVersion: semver.MustParse("1.25.0")})
				pkgRes, err := finder.Find(model, "parent-pkg", &versions.VersionSelectionSemver{
					Constraints: "<3.0.0",
				})
				require.NoError(t, err)
				require.Equal(t, pkg, pkgRes)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.test(t)
		})
	}

}
