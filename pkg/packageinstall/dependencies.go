// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package packageinstall

import (
	"context"
	"fmt"
	"strings"

	packagingv1alpha1 "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apis/packaging/v1alpha1"
	pkgingv1alpha1 "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apis/packaging/v1alpha1"
	"github.com/vmware-tanzu/carvel-kapp-controller/pkg/apiserver/apis/datapackaging/v1alpha1"
	kcclient "github.com/vmware-tanzu/carvel-kapp-controller/pkg/client/clientset/versioned"
	versionsv1alpha1 "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// OwnerAnnKey is key for Kapp controller ownsership for dependency package installs
	OwnerAnnKey                    = "kapp-controller.carvel.dev/owner"
	ownershipAnnotationValuePrefix = "PackageInstall/"
)

// PackageDependencyHandler helps in resolve and reconcile dependency packages for any PackageInstall
type PackageDependencyHandler struct {
	finder   *PackageFinder
	kcclient kcclient.Interface
}

var _ DependencyHandler = &PackageDependencyHandler{}

// NewPackageDependencyHandler creates a new object for PackageDependencyHandler
func NewPackageDependencyHandler(finder *PackageFinder, kcclient kcclient.Interface) *PackageDependencyHandler {
	return &PackageDependencyHandler{
		finder:   finder,
		kcclient: kcclient,
	}
}

// Resolve returns the list of dependency packages yet to be reconciled for the given PackageInstall
func (pdh *PackageDependencyHandler) Resolve(pkgi *pkgingv1alpha1.PackageInstall, pkg *v1alpha1.Package) ([]*v1alpha1.Package, error) {
	var dependencies []*v1alpha1.Package
	var missingDependencies []string

	overridesMap, err := pdh.PackageVersionOverrides(pkgi, pkg.Spec.Dependencies)
	if err != nil {
		return nil, err
	}

	// Check if packages exist in the cluster
	for _, dep := range pkg.Spec.Dependencies {
		switch {
		case dep.Package != nil:
			version := dep.Package.VersionSelection
			if newVersion, ok := overridesMap[dep.Name]; ok {
				version = newVersion
			}

			pkg, err := pdh.finder.Find(pkgi, dep.Package.RefName, version)
			if err != nil {
				errorMsg := fmt.Sprintf("%s : %+v", dep.Package.RefName+"/"+version.Constraints, err)
				missingDependencies = append(missingDependencies, errorMsg)
				continue
			}
			dependencies = append(dependencies, pkg)
		}
	}

	if len(missingDependencies) > 0 {
		return nil, fmt.Errorf("Failed to resolve the following dependencies:\n " + strings.Join(missingDependencies, "\n"))
	}
	return dependencies, nil
}

// Reconcile installs dependency Packages for a PackageInstall if it is not already installed
func (pdh *PackageDependencyHandler) Reconcile(pkgi *pkgingv1alpha1.PackageInstall, dependencyList []*v1alpha1.Package) error {
	pkgiList, err := pdh.kcclient.PackagingV1alpha1().PackageInstalls(pkgi.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, dep := range dependencyList {
		pkgiFound := false
		for _, pkgi := range pkgiList.Items {
			if dep.Spec.RefName == pkgi.Spec.PackageRef.RefName && pkgi.Status.Version == dep.Spec.Version {
				pkgiFound = true
			}
		}

		if !pkgiFound {
			pkgiName := "dep-pkgi-" + generateRandomToken(6)
			pkgi := &packagingv1alpha1.PackageInstall{
				TypeMeta: pkgi.TypeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name:      pkgiName,
					Namespace: pkgi.Namespace,
					Annotations: map[string]string{
						OwnerAnnKey: ownershipAnnotationValuePrefix + pkgi.Name,
					},
				},
				Spec: packagingv1alpha1.PackageInstallSpec{
					ServiceAccountName: pkgi.Spec.ServiceAccountName,
					PackageRef: &packagingv1alpha1.PackageRef{
						RefName: dep.Spec.RefName,
						VersionSelection: &versionsv1alpha1.VersionSelectionSemver{
							Constraints: dep.Spec.Version,
						},
					},
					DefaultNamespace: pkgi.Spec.DefaultNamespace,
				},
			}

			_, err := pdh.kcclient.PackagingV1alpha1().PackageInstalls(pkgi.Namespace).Create(context.TODO(), pkgi, metav1.CreateOptions{})
			if errors.IsAlreadyExists(err) {
				pkgi.ObjectMeta.Name = "dep-pkgi-" + generateRandomToken(6)
				_, err = pdh.kcclient.PackagingV1alpha1().PackageInstalls(pkgi.Namespace).Create(context.TODO(), pkgi, metav1.CreateOptions{})
			}
			if err != nil {
				return fmt.Errorf("unable to create the packageinstall for the package %s: %w", dep.Name, err)
			}
		}
	}
	return nil
}

// PackageVersionOverrides overrides the dependencies based on PackageInstall
func (pdh *PackageDependencyHandler) PackageVersionOverrides(pkgi *pkgingv1alpha1.PackageInstall, dependencies []*v1alpha1.Dependency) (map[string]*versionsv1alpha1.VersionSelectionSemver, error) {
	// store the dependency packages that can be overridden in a map
	depPackages := make(map[string]string)
	for _, dep := range dependencies {
		if dep.Package != nil {
			depPackages[dep.Name] = dep.Package.RefName
		}
	}

	// store the package overrides to the map
	overridesMap := make(map[string]*versionsv1alpha1.VersionSelectionSemver)
	invalidOverrides := []string{}
	for _, override := range pkgi.Spec.Dependencies.Override {
		if pkgRefName, ok := depPackages[override.Name]; ok && override.Package.RefName == pkgRefName {
			overridesMap[override.Name] = override.Package.VersionSelection
		} else {
			invalidOverrides = append(invalidOverrides, override.Name+"/"+override.Package.RefName)
		}
	}

	if len(invalidOverrides) > 0 {
		return nil, fmt.Errorf("The following dependency overrides '" + strings.Join(invalidOverrides, ", ") +
			"' are not defined as dependencies in the Package " + pkgi.Spec.PackageRef.RefName)
	}
	return overridesMap, nil
}
