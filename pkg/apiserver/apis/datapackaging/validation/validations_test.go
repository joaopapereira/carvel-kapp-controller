// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-kapp-controller/pkg/apiserver/apis/datapackaging"
	"github.com/vmware-tanzu/carvel-kapp-controller/pkg/apiserver/apis/datapackaging/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidatePackageMetadataNameInvalid(t *testing.T) {
	invalidName := "bummer-boy"
	// Name could be invalid for many reasons so just assert we have
	// an error relating to name and not specific error string
	expectedErr := field.Error{
		Type:  field.ErrorTypeInvalid,
		Field: "metadata.name",
	}

	errList := validation.ValidatePackageMetadataName(invalidName, field.NewPath("metadata").Child("name"))

	if len(errList) == 0 {
		t.Fatalf("Expected validation to error when given invalid name")
	}

	if !contains(errList, expectedErr) {
		t.Fatalf("Expected invalid field error for metadata.name, but got: %v", errList.ToAggregate())
	}
}

func TestValidatePackageMetadataNameValid(t *testing.T) {
	validName := "package.carvel.dev"

	errList := validation.ValidatePackageMetadataName(validName, field.NewPath("metadata").Child("name"))

	if len(errList) != 0 {
		t.Fatalf("Expected no error for valid name, but got: %v", errList.ToAggregate())
	}
}

func TestValidatePackageNameInvalid(t *testing.T) {
	invalidName := "pkg.3.0"
	pkgName := "pkg"
	pkgVersion := "2.0"
	expectedErr := field.Error{
		Type:  field.ErrorTypeInvalid,
		Field: "metadata.name",
	}

	errList := validation.ValidatePackageName(invalidName, pkgName, pkgVersion, field.NewPath("metadata", "name"))

	if len(errList) == 0 {
		t.Fatalf("Expected error when PackageVersion name is invalid")
	}

	if !contains(errList, expectedErr) {
		t.Fatalf("Expected invalid field error for metadata.name, but got: %v", errList.ToAggregate())
	}
}

func TestValidatePackageNameValid(t *testing.T) {
	validName := "pkg.2.0"
	pkgName := "pkg"
	pkgVersion := "2.0"

	errList := validation.ValidatePackageName(validName, pkgName, pkgVersion, field.NewPath("metadata", "name"))

	if len(errList) != 0 {
		t.Fatalf("Expected no error when PackageVersion name is valid, but got: %v", errList.ToAggregate())
	}
}

func TestValidatePackageSpecPackageVersionInvalidEmpty(t *testing.T) {
	invalidVersion := ""
	expectedErr := field.Error{
		Type:  field.ErrorTypeInvalid,
		Field: "spec.version",
	}

	errList := validation.ValidatePackageSpecVersion(invalidVersion, field.NewPath("spec", "version"))

	if len(errList) == 0 {
		t.Fatalf("Expected error when spec.version is invalid")
	}

	if !contains(errList, expectedErr) {
		t.Fatalf("Expected invalid field error for spec.version, but got: %v", errList.ToAggregate())
	}
}

func TestValidatePackageSpecPackageVersionInvalidNonSemver(t *testing.T) {
	invalidVersion := "invalid.1.0"
	expectedErr := field.Error{
		Type:  field.ErrorTypeInvalid,
		Field: "spec.version",
	}

	errList := validation.ValidatePackageSpecVersion(invalidVersion, field.NewPath("spec", "version"))

	if len(errList) == 0 {
		t.Fatalf("Expected error when spec.version is invalid")
	}

	if !contains(errList, expectedErr) {
		t.Fatalf("Expected invalid field error for spec.version, but got: %v", errList.ToAggregate())
	}
}

func TestValidatePackageSpecPackageVersionValid(t *testing.T) {
	validVersion := "1.0.0"

	errList := validation.ValidatePackageSpecVersion(validVersion, field.NewPath("spec", "version"))

	if len(errList) != 0 {
		t.Fatalf("Expected no error when spec.version is valid, but got %v", errList.ToAggregate().Error())
	}
}

func TestValidatePackageSpecPackageNameInvalid(t *testing.T) {
	invalidName := ""
	expectedErr := field.Error{
		Type:  field.ErrorTypeRequired,
		Field: "spec.packageName",
	}

	errList := validation.ValidatePackageSpecPackageName(invalidName, field.NewPath("spec", "packageName"))

	if len(errList) == 0 {
		t.Fatalf("Expected error when spec.packageName is invalid")
	}

	if !contains(errList, expectedErr) {
		t.Fatalf("Expected invalid field error for spec.packageName, but got: %v", errList.ToAggregate())
	}
}

func TestValidatePackageSpecPackageNameValid(t *testing.T) {
	validName := "package.carvel.dev"

	errList := validation.ValidatePackageSpecPackageName(validName, field.NewPath("spec", "packageName"))

	if len(errList) != 0 {
		t.Fatalf("Expected no error when spec.packageName is valid")
	}
}

func TestValidatePackageVersionConstraints(t *testing.T) {
	errList := validation.ValidatePackageVersionConstraints(">=1.21.0", field.NewPath("spec", "kubernetesVersionSelection", "constraints"))
	assert.Empty(t, errList)

	errList = validation.ValidatePackageVersionConstraints("my cat's breath smells like cat food", field.NewPath("spec", "kubernetesVersionSelection", "constraints"))
	assert.Equal(t, 1, len(errList))
}

// Searches for Error in ErrorList by Type + Field, but not details
func contains(errList field.ErrorList, expectedErr field.Error) bool {
	for _, err := range errList {
		if err.Type == expectedErr.Type && err.Field == expectedErr.Field {
			return true
		}
	}
	return false
}

func TestValidatePackageDependencies(t *testing.T) {

	testCases := []struct {
		name                  string
		expectedErrorList     []string
		dependencies          []*datapackaging.Dependency
		expectedErrListLength int
		testExec              func(t *testing.T)
	}{
		{
			name:                  "Dependency name cannot be empty",
			dependencies:          []*datapackaging.Dependency{{}},
			expectedErrListLength: 1,
			expectedErrorList:     []string{"spec.dependencies[0].name: Required value: cannot be empty"},
		},
		{
			name:                  "Dependency name cannot be empty - for multiple dependencies",
			dependencies:          []*datapackaging.Dependency{{}, {}},
			expectedErrListLength: 2,
			expectedErrorList: []string{
				"spec.dependencies[0].name: Required value: cannot be empty",
				"spec.dependencies[1].name: Required value: cannot be empty",
			},
		},
		{
			name:                  "Dependency name should be unique",
			dependencies:          []*datapackaging.Dependency{{Name: "dep-1"}, {Name: "dep-1"}},
			expectedErrListLength: 1,
			expectedErrorList:     []string{"spec.dependencies[1].name: Invalid value: \"dep-1\": should be unique"},
		},
		{
			name: "Dependency.*.Package.RefName cannot be empty",
			dependencies: []*datapackaging.Dependency{
				{
					Name:    "dep-1",
					Package: &datapackaging.PackageRef{},
				},
			},
			expectedErrListLength: 1,
			expectedErrorList:     []string{"spec.dependencies[0].package.refName: Required value: cannot be empty"},
		},
		{
			name: "ValidatePackageDependencies - Success",
			dependencies: []*datapackaging.Dependency{
				{
					Name: "dep-1",
					Package: &datapackaging.PackageRef{
						RefName: "test-pkg",
					},
				},
			},
		},
		{
			name:         "ValidatePackageDependencies, Package key can be ignored - Success",
			dependencies: []*datapackaging.Dependency{{Name: "dep-1"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errList := validation.ValidatePackageDependencies(tc.dependencies)
			require.Len(t, errList, tc.expectedErrListLength)
			for _, err := range errList {
				require.Contains(t, tc.expectedErrorList, err.Error())
			}
		})
	}

}
