// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package packageinstall

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	semver "github.com/k14s/semver/v4"
	pkgingv1alpha1 "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apis/packaging/v1alpha1"
	"github.com/vmware-tanzu/carvel-kapp-controller/pkg/apiserver/apis/datapackaging/v1alpha1"
	datapkgingv1alpha1 "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apiserver/apis/datapackaging/v1alpha1"
	pkgclient "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apiserver/client/clientset/versioned"
	"github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions"
	versionsv1alpha1 "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions/v1alpha1"
	verv1alpha1 "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PackageFinder helps to find an available package for any Package Install
type PackageFinder struct {
	pkgclient pkgclient.Interface
	compInfo  ComponentInfo
	log       logr.Logger
}

var _ Finder = &PackageFinder{}

// NewPackageFinder creates new Object for PackageFinder
func NewPackageFinder(log logr.Logger, pkgclient pkgclient.Interface, compInfo ComponentInfo) *PackageFinder {
	return &PackageFinder{log: log, pkgclient: pkgclient, compInfo: compInfo}
}

// Find retrieves the most recent package that matches the semverConfig selector
func (pf *PackageFinder) Find(model *pkgingv1alpha1.PackageInstall, packageRef string, semverConfig *versionsv1alpha1.VersionSelectionSemver) (*v1alpha1.Package, error) {
	// TODO:  this can cause some performance problems,
	// we should try caching this in the future, if the pkgclient doesn't do that for us right now
	pkgList, err := pf.pkgclient.DataV1alpha1().Packages(model.Namespace).List(
		context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(pkgList.Items) == 0 {
		return nil, fmt.Errorf("Package %s not found", packageRef)
	}

	var versionStrs []string
	versionToPkg := map[string]datapkgingv1alpha1.Package{}

	requiresClusterVersion := false
	for _, pkg := range pkgList.Items {
		if pkg.Spec.RefName == packageRef {
			versionStrs = append(versionStrs, pkg.Spec.Version)
			versionToPkg[pkg.Spec.Version] = pkg

			if pkgHasK8sConstraint(&pkg) {
				requiresClusterVersion = true
			}
		}
	}

	// If constraint is a single specified version, then we
	// do not want to force user to manually set prereleases={}
	if len(semverConfig.Constraints) > 0 && semverConfig.Prereleases == nil {
		// Will error if it's not a single version
		singleVer, err := versions.NewSemver(semverConfig.Constraints)
		if err == nil && len(singleVer.Pre) > 0 {
			semverConfig.Prereleases = &verv1alpha1.VersionSelectionSemverPrereleases{}
		}
	}

	var vcc []versions.ConstraintCallback
	// we only need to populate the versionInfo we know that the packages have constraints that will require this info.
	if requiresClusterVersion {
		v, err := pf.compInfo.KubernetesVersion(model.Spec.ServiceAccountName, model.Spec.Cluster, &model.ObjectMeta)
		if err != nil {
			return nil, fmt.Errorf("Unable to get kubernetes version: %s", err)
		}

		k8sConstraint := func(pkgVer string) bool {
			pkg := versionToPkg[pkgVer]
			return pf.clusterVersionConstraintsSatisfied(&pkg, model.Annotations, v)
		}

		vcc = append(vcc, versions.ConstraintCallback{Constraint: k8sConstraint, Name: "kubernetes-version-check"})
	}

	kcConstraint := func(pkgVer string) bool {
		pkg := versionToPkg[pkgVer]
		return pf.kcVersionConstraintsSatisfied(&pkg, model.Annotations)
	}

	vcc = append(vcc, versions.ConstraintCallback{Constraint: kcConstraint, Name: "kapp-controller-version-check"})

	verConfig := verv1alpha1.VersionSelection{Semver: semverConfig}
	selectedVersion, err := versions.HighestConstrainedVersionWithAdditionalConstraints(versionStrs, verConfig, vcc)
	if err != nil {
		return nil, err
	}

	if pkg, found := versionToPkg[selectedVersion]; found {
		return &pkg, nil
	}

	return nil, fmt.Errorf("Could not find package with name '%s' and version '%s'",
		packageRef, selectedVersion)
}

// kcVersionConstraintsSatisfied helps to check if the given package's kapp controller version constraints are satisfied
func (pf *PackageFinder) kcVersionConstraintsSatisfied(pkg *datapkgingv1alpha1.Package, annotations map[string]string) bool {
	if pkg.Spec.KappControllerVersionSelection == nil || pkg.Spec.KappControllerVersionSelection.Constraints == "" {
		return true
	}
	const kappControllerVersionOverrideAnnotation = "packaging.carvel.dev/ignore-kapp-controller-version-selection"

	_, found := annotations[kappControllerVersionOverrideAnnotation]
	if found {
		pf.log.Info("Found kapp-controller version override annotation; not applying version constraints")
		return true
	}

	v, err := pf.compInfo.KappControllerVersion()
	if err != nil {
		return false
	}

	v.Pre = semver.PRVersion{}
	v.Build = semver.BuildMeta{}

	constraints, _ := semver.ParseRange(pkg.Spec.KappControllerVersionSelection.Constraints) // ignore err because validation should have already caught it
	return constraints(v)
}

// clusterVersionConstraintsSatisfied helps to check if the given package's kubernetes cluster version constraints are satisfied
func (pf *PackageFinder) clusterVersionConstraintsSatisfied(pkg *datapkgingv1alpha1.Package, annotations map[string]string, clusterVersion semver.Version) bool {
	if !pkgHasK8sConstraint(pkg) {
		return true
	}
	const kubernetesVersionOverrideAnnotation = "packaging.carvel.dev/ignore-kubernetes-version-selection"

	_, found := annotations[kubernetesVersionOverrideAnnotation]
	if found {
		pf.log.Info("Found kubernetes version override annotation; not applying version constraints")
		return true
	}

	constraintsFunc, _ := semver.ParseRange(pkg.Spec.KubernetesVersionSelection.Constraints) // ignore err because validation should have already caught it
	return constraintsFunc(clusterVersion)
}

// pkgHasK8sConstraint check if the given package has a kubernetes version selection constraint
func pkgHasK8sConstraint(pkg *datapkgingv1alpha1.Package) bool {
	return pkg.Spec.KubernetesVersionSelection != nil && pkg.Spec.KubernetesVersionSelection.Constraints != ""
}
