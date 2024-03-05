// Copyright 2024 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
package packageinstall_test

import (
	"testing"

	"github.com/k14s/semver/v4"
	"github.com/stretchr/testify/require"
	pkgingv1alpha1 "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apis/packaging/v1alpha1"

	datapkgingv1alpha1 "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apiserver/apis/datapackaging/v1alpha1"
	fakeapiserver "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apiserver/client/clientset/versioned/fake"
	fakekappctrl "github.com/vmware-tanzu/carvel-kapp-controller/pkg/client/clientset/versioned/fake"

	"github.com/vmware-tanzu/carvel-kapp-controller/pkg/packageinstall"
	versions "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func Test_PackageDependencyHandler_Resolve(t *testing.T) {
	namespace := "default-ns"
	testCases := []struct {
		name     string
		testExec func(t *testing.T, subject *packageinstall.PackageDependencyHandler, fakePkgClient *fakeapiserver.Clientset, fakeAppClient *fakekappctrl.Clientset)
	}{
		{
			name: "Dependency Resolution Successful - single dependency package",
			testExec: func(t *testing.T, subject *packageinstall.PackageDependencyHandler, fakePkgClient *fakeapiserver.Clientset, fakeAppClient *fakekappctrl.Clientset) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{
						buildDependency("dependency-pkg", "1.0.0"),
					}, "", "",
				)
				dependencyPkg := generatePackageWithConstraints("dependency-pkg", namespace, "1.0.0", nil, "", "")
				require.NoError(t, fakePkgClient.Tracker().Add(pkg))
				require.NoError(t, fakePkgClient.Tracker().Add(dependencyPkg))

				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", "use-local-cluster-sa", true, true)
				require.NoError(t, fakeAppClient.Tracker().Add(model))

				pkgList, err := subject.Resolve(model, pkg)
				require.NoError(t, err)
				require.Len(t, pkgList, 1)
				require.Equal(t, dependencyPkg, pkgList[0])
			},
		},
		{
			name: "Dependency Resolution Successful - with multiple dependency packages",
			testExec: func(t *testing.T, subject *packageinstall.PackageDependencyHandler, fakePkgClient *fakeapiserver.Clientset, fakeAppClient *fakekappctrl.Clientset) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{
						buildDependency("dependency-pkg", "1.0.0"),
						buildDependency("dependency-pkg2", "1.0.0"),
					}, "", "",
				)
				dependencyPkg := generatePackageWithConstraints("dependency-pkg", namespace, "1.0.0", nil, "", "")
				dependencyPkg2 := generatePackageWithConstraints("dependency-pkg2", namespace, "1.0.0", nil, "", "")
				require.NoError(t, fakePkgClient.Tracker().Add(pkg))
				require.NoError(t, fakePkgClient.Tracker().Add(dependencyPkg))
				require.NoError(t, fakePkgClient.Tracker().Add(dependencyPkg2))

				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", "use-local-cluster-sa", true, true)
				require.NoError(t, fakeAppClient.Tracker().Add(model))

				pkgList, err := subject.Resolve(model, pkg)
				require.NoError(t, err)
				require.Len(t, pkgList, 2)
				require.Equal(t, []*datapkgingv1alpha1.Package{dependencyPkg, dependencyPkg2}, pkgList)
			},
		},
		{
			name: "Dependency package not found",
			testExec: func(t *testing.T, subject *packageinstall.PackageDependencyHandler, fakePkgClient *fakeapiserver.Clientset, fakeAppClient *fakekappctrl.Clientset) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{
						buildDependency("dependency-pkg", "1.0.0"),
					}, "", "",
				)
				require.NoError(t, fakePkgClient.Tracker().Add(pkg))

				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", "use-local-cluster-sa", true, true)
				require.NoError(t, fakeAppClient.Tracker().Add(model))

				pkgList, err := subject.Resolve(model, pkg)
				expectedError := "Failed to resolve the following dependencies:\n dependency-pkg/1.0.0 : Expected to find at least one version, but did not (details: all=0 -> after-prereleases-filter=0 -> after-kapp-controller-version-check=0 -> after-constraints-filter=0)"
				require.ErrorContains(t, err, expectedError)
				require.Equal(t, 0, len(pkgList))
			},
		},
		{
			name: "Multiple dependency packages not found",
			testExec: func(t *testing.T, subject *packageinstall.PackageDependencyHandler, fakePkgClient *fakeapiserver.Clientset, fakeAppClient *fakekappctrl.Clientset) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{
						buildDependency("dependency-pkg", "1.0.0"),
						buildDependency("dependency-pkg-2", "1.0.0"),
					}, "", "",
				)
				require.NoError(t, fakePkgClient.Tracker().Add(pkg))

				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", "use-local-cluster-sa", true, true)
				require.NoError(t, fakeAppClient.Tracker().Add(model))

				pkgList, err := subject.Resolve(model, pkg)
				expectedError := "Failed to resolve the following dependencies:\n dependency-pkg/1.0.0 : Expected to find at least one version, but did not (details: all=0 -> after-prereleases-filter=0 -> after-kapp-controller-version-check=0 -> after-constraints-filter=0)\ndependency-pkg-2/1.0.0 : Expected to find at least one version, but did not (details: all=0 -> after-prereleases-filter=0 -> after-kapp-controller-version-check=0 -> after-constraints-filter=0)"
				require.ErrorContains(t, err, expectedError)
				require.Equal(t, 0, len(pkgList))
			},
		},
		{
			name: "One of the depencies not found",
			testExec: func(t *testing.T, subject *packageinstall.PackageDependencyHandler, fakePkgClient *fakeapiserver.Clientset, fakeAppClient *fakekappctrl.Clientset) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{
						buildDependency("dependency-pkg", "1.0.0"),
						buildDependency("dependency-pkg2", "1.0.0"),
					}, "", "",
				)
				dependencyPkg := generatePackageWithConstraints("dependency-pkg", namespace, "1.0.0", nil, "", "")
				require.NoError(t, fakePkgClient.Tracker().Add(pkg))
				require.NoError(t, fakePkgClient.Tracker().Add(dependencyPkg))

				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", "use-local-cluster-sa", true, true)
				require.NoError(t, fakeAppClient.Tracker().Add(model))

				pkgList, err := subject.Resolve(model, pkg)
				expectedError := "Failed to resolve the following dependencies:\n dependency-pkg2/1.0.0 : Expected to find at least one version"
				require.ErrorContains(t, err, expectedError)
				require.Equal(t, 0, len(pkgList))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakePkgClient := fakeapiserver.NewSimpleClientset()
			log := logf.Log.WithName("kc")
			fakekctrl := fakekappctrl.NewSimpleClientset()
			pdh := packageinstall.NewPackageDependencyHandler(
				packageinstall.NewPackageFinder(log, fakePkgClient, packageinstall.FakeComponentInfo{KCVersion: semver.MustParse("0.42.31337")}),
				fakekctrl,
			)
			tc.testExec(t, pdh, fakePkgClient, fakekctrl)
		})
	}
}

func Test_PackageDependencyHandler_Reconcile(t *testing.T) {
	namespace := "default-ns"
	serviceaccount := "use-local-cluster-sa"
	testCases := []struct {
		name     string
		test     func(t *testing.T)
		testExec func(t *testing.T, subject *packageinstall.PackageDependencyHandler, fakePkgClient *fakeapiserver.Clientset, fakeAppClient *fakekappctrl.Clientset)
	}{
		{
			name: "Dependency Reconciliation Successful - PackageInstall created for dependency package",
			testExec: func(t *testing.T, subject *packageinstall.PackageDependencyHandler, fakePkgClient *fakeapiserver.Clientset, fakeAppClient *fakekappctrl.Clientset) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{
						buildDependency("dependency-pkg", "1.0.0"),
					}, "", "",
				)
				dependencyPkg := generatePackageWithConstraints("dependency-pkg", namespace, "1.0.0", nil, "", "")
				require.NoError(t, fakePkgClient.Tracker().Add(pkg))
				require.NoError(t, fakePkgClient.Tracker().Add(dependencyPkg))

				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				require.NoError(t, fakeAppClient.Tracker().Add(model))

				err := subject.Reconcile(model, []*datapkgingv1alpha1.Package{dependencyPkg})
				require.NoError(t, err)

				objs, _ := getPackageInstalls(fakeAppClient, namespace)
				pkgiList := objs.(*pkgingv1alpha1.PackageInstallList)
				var pkgi *pkgingv1alpha1.PackageInstall
				for _, pi := range pkgiList.Items {
					if pi.Spec.PackageRef.VersionSelection.Constraints == "1.0.0" && pi.Spec.PackageRef.RefName == "dependency-pkg" {
						pkgi = &pi
						break
					}
				}
				require.NotNil(t, pkgi)
				require.Equal(t, pkgi.ObjectMeta.Annotations[packageinstall.OwnerAnnKey], "PackageInstall/parent-pkgi")
				require.Equal(t, pkgi.ObjectMeta.Namespace, namespace)
				require.Equal(t, pkgi.Spec.ServiceAccountName, serviceaccount)
			},
		},
		{
			name: "Dependency Reconciliation Successful - PackageInstall created for multiple dependency packages",
			testExec: func(t *testing.T, subject *packageinstall.PackageDependencyHandler, fakePkgClient *fakeapiserver.Clientset, fakeAppClient *fakekappctrl.Clientset) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{
						buildDependency("dependency-pkg", "1.0.0"),
						buildDependency("dependency-pkg2", "1.0.0"),
					}, "", "",
				)
				dependencyPkg := generatePackageWithConstraints("dependency-pkg", namespace, "1.0.0", nil, "", "")
				dependencyPkg2 := generatePackageWithConstraints("dependency-pkg2", namespace, "1.0.0", nil, "", "")
				require.NoError(t, fakePkgClient.Tracker().Add(pkg))
				require.NoError(t, fakePkgClient.Tracker().Add(dependencyPkg))
				require.NoError(t, fakePkgClient.Tracker().Add(dependencyPkg2))

				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				require.NoError(t, fakeAppClient.Tracker().Add(model))

				err := subject.Reconcile(model, []*datapkgingv1alpha1.Package{dependencyPkg, dependencyPkg2})
				require.NoError(t, err)

				objs, _ := getPackageInstalls(fakeAppClient, namespace)
				pkgiList := objs.(*pkgingv1alpha1.PackageInstallList)
				var pkgi, pkgi2 = pkgingv1alpha1.PackageInstall{}, pkgingv1alpha1.PackageInstall{}
				for _, pi := range pkgiList.Items {
					if pi.Spec.PackageRef.VersionSelection.Constraints == "1.0.0" && pi.Spec.PackageRef.RefName == "dependency-pkg" {
						pkgi = pi
					} else if pi.Spec.PackageRef.VersionSelection.Constraints == "1.0.0" && pi.Spec.PackageRef.RefName == "dependency-pkg2" {
						pkgi2 = pi
					}
				}

				// verifying dependency pkgi 1
				require.NotNil(t, pkgi)
				require.Equal(t, pkgi.ObjectMeta.Annotations[packageinstall.OwnerAnnKey], "PackageInstall/parent-pkgi")
				require.Equal(t, pkgi.ObjectMeta.Namespace, namespace)
				require.Equal(t, pkgi.Spec.ServiceAccountName, serviceaccount)

				// verifying dependency pkgi 2
				require.NotNil(t, pkgi2)
				require.Equal(t, pkgi2.ObjectMeta.Annotations[packageinstall.OwnerAnnKey], "PackageInstall/parent-pkgi")
				require.Equal(t, pkgi2.ObjectMeta.Namespace, namespace)
				require.Equal(t, pkgi2.Spec.ServiceAccountName, serviceaccount)

			},
		},
		{
			name: "Dependency Reconciliation Successful - PackageInstall already exist for dependency package",
			testExec: func(t *testing.T, subject *packageinstall.PackageDependencyHandler, fakePkgClient *fakeapiserver.Clientset, fakeAppClient *fakekappctrl.Clientset) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{
						buildDependency("dependency-pkg", "1.0.0"),
					}, "", "",
				)
				dependencyPkg := generatePackageWithConstraints("dependency-pkg", namespace, "1.0.0", nil, "", "")
				require.NoError(t, fakePkgClient.Tracker().Add(pkg))
				require.NoError(t, fakePkgClient.Tracker().Add(dependencyPkg))

				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				depmodel := buildPackageInstall("dep-pkgi", namespace, "dependency-pkg", "1.0.0", serviceaccount, true, true)
				require.NoError(t, fakeAppClient.Tracker().Add(model))
				require.NoError(t, fakeAppClient.Tracker().Add(depmodel))

				err := subject.Reconcile(model, []*datapkgingv1alpha1.Package{dependencyPkg})
				require.NoError(t, err)

				objs, _ := getPackageInstalls(fakeAppClient, namespace)
				pkgiList := objs.(*pkgingv1alpha1.PackageInstallList)
				var pkgi *pkgingv1alpha1.PackageInstall
				for _, pi := range pkgiList.Items {
					if pi.Spec.PackageRef.VersionSelection.Constraints == "1.0.0" && pi.Spec.PackageRef.RefName == "dependency-pkg" {
						pkgi = &pi
						break
					}
				}
				require.NotNil(t, pkgi)
				require.Equal(t, "", pkgi.ObjectMeta.Annotations[packageinstall.OwnerAnnKey])
				require.Equal(t, pkgi.ObjectMeta.Namespace, namespace)
				require.Equal(t, pkgi.Spec.ServiceAccountName, serviceaccount)
			},
		},
		{
			name: "Dependency Reconciliation Successful - PackageInstall already exist for one of the dependency packages",
			testExec: func(t *testing.T, subject *packageinstall.PackageDependencyHandler, fakePkgClient *fakeapiserver.Clientset, fakeAppClient *fakekappctrl.Clientset) {
				pkg := generatePackageWithConstraints(
					"parent-pkg", namespace, "1.0.0", []datapkgingv1alpha1.Dependency{
						buildDependency("dependency-pkg", "1.0.0"),
						buildDependency("dependency-pkg2", "1.0.0"),
					}, "", "",
				)
				dependencyPkg := generatePackageWithConstraints("dependency-pkg", namespace, "1.0.0", nil, "", "")
				dependencyPkg2 := generatePackageWithConstraints("dependency-pkg2", namespace, "1.0.0", nil, "", "")
				require.NoError(t, fakePkgClient.Tracker().Add(pkg))
				require.NoError(t, fakePkgClient.Tracker().Add(dependencyPkg))
				require.NoError(t, fakePkgClient.Tracker().Add(dependencyPkg2))

				model := buildPackageInstall("parent-pkgi", namespace, "parent-pkg", "1.0.0", serviceaccount, true, true)
				depmodel := buildPackageInstall("dep-pkgi", namespace, "dependency-pkg", "1.0.0", serviceaccount, true, true)
				depmodel.Status.Version = "1.0.0"
				require.NoError(t, fakeAppClient.Tracker().Add(model))
				require.NoError(t, fakeAppClient.Tracker().Add(depmodel))

				err := subject.Reconcile(model, []*datapkgingv1alpha1.Package{dependencyPkg, dependencyPkg2})
				require.NoError(t, err)

				objs, _ := getPackageInstalls(fakeAppClient, namespace)
				pkgiList := objs.(*pkgingv1alpha1.PackageInstallList)
				var pkgi, pkgi2 = pkgingv1alpha1.PackageInstall{}, pkgingv1alpha1.PackageInstall{}
				for _, pi := range pkgiList.Items {
					if pi.Spec.PackageRef.VersionSelection.Constraints == "1.0.0" && pi.Spec.PackageRef.RefName == "dependency-pkg" {
						pkgi = pi
					} else if pi.Spec.PackageRef.VersionSelection.Constraints == "1.0.0" && pi.Spec.PackageRef.RefName == "dependency-pkg2" {
						pkgi2 = pi
					}
				}

				// verifying dependency pkgi 1
				require.NotNil(t, pkgi)
				require.Equal(t, pkgi.ObjectMeta.Annotations[packageinstall.OwnerAnnKey], "")
				require.Equal(t, pkgi.ObjectMeta.Namespace, namespace)
				require.Equal(t, pkgi.Spec.ServiceAccountName, serviceaccount)

				// verifying dependency pkgi 2
				require.NotNil(t, pkgi2)
				require.Equal(t, pkgi2.ObjectMeta.Annotations[packageinstall.OwnerAnnKey], "PackageInstall/parent-pkgi")
				require.Equal(t, pkgi2.ObjectMeta.Namespace, namespace)
				require.Equal(t, pkgi2.Spec.ServiceAccountName, serviceaccount)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakePkgClient := fakeapiserver.NewSimpleClientset()
			log := logf.Log.WithName("kc")
			fakekctrl := fakekappctrl.NewSimpleClientset()
			pdh := packageinstall.NewPackageDependencyHandler(
				packageinstall.NewPackageFinder(log, fakePkgClient, packageinstall.FakeComponentInfo{KCVersion: semver.MustParse("0.42.31337")}),
				fakekctrl,
			)
			tc.testExec(t, pdh, fakePkgClient, fakekctrl)
		})
	}

}

func buildDependency(refName string, version string) datapkgingv1alpha1.Dependency {
	return datapkgingv1alpha1.Dependency{
		Package: &datapkgingv1alpha1.PackageRef{
			RefName: refName,
			VersionSelection: &versions.VersionSelectionSemver{
				Constraints: version,
				Prereleases: &versions.VersionSelectionSemverPrereleases{},
			},
		},
	}
}

func buildPackageInstall(name, namespace, refName, version, saName string, addDependencies, installDependencies bool) *pkgingv1alpha1.PackageInstall {
	pkgInstall := &pkgingv1alpha1.PackageInstall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: pkgingv1alpha1.PackageInstallSpec{
			PackageRef: &pkgingv1alpha1.PackageRef{
				RefName: refName,
				VersionSelection: &versions.VersionSelectionSemver{
					Constraints: version,
				},
			},
			ServiceAccountName: saName,
		},
	}

	if addDependencies {
		pkgInstall.Spec.Dependencies = pkgingv1alpha1.Dependencies{
			Install: installDependencies,
		}
	}
	return pkgInstall
}

func generatePackageWithConstraints(name, ns, version string, dependencies []datapkgingv1alpha1.Dependency, kcConstraint, k8sConstraint string) *datapkgingv1alpha1.Package {
	return &datapkgingv1alpha1.Package{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "." + version,
			Namespace: ns,
		},
		Spec: datapkgingv1alpha1.PackageSpec{
			RefName:      name,
			Version:      version,
			Dependencies: dependencies,
			KappControllerVersionSelection: &datapkgingv1alpha1.VersionSelection{
				Constraints: kcConstraint,
			},
			KubernetesVersionSelection: &datapkgingv1alpha1.VersionSelection{
				Constraints: k8sConstraint,
			},
		},
	}
}

func getPackageInstalls(fakekctrl *fakekappctrl.Clientset, namespace string) (runtime.Object, error) {
	gvr := schema.GroupVersionResource{"packaging.carvel.dev", "v1alpha1", "packageinstalls"}
	gvk := schema.GroupVersionKind{Group: "packaging.carvel.dev", Version: "v1alpha1", Kind: "PackageInstall"}
	return fakekctrl.Tracker().List(gvr, gvk, namespace)
}
