// Copyright 2024 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
package kappcontroller

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vmware-tanzu/carvel-kapp-controller/test/e2e"
)

func Test_PackageInstall_Dependencies_Installed_Successfully(t *testing.T) {
	env := e2e.BuildEnv(t)
	logger := e2e.Logger{}
	kubectl := e2e.Kubectl{T: t, Namespace: env.Namespace, L: logger}

	cleanUp := func() {

		kubectl.RunWithOpts([]string{"delete", "packageinstalls/" + getPackageInstallByPackageName(&kubectl, "pkg.child.carvel.dev")}, e2e.RunOpts{AllowError: true})
		kubectl.RunWithOpts([]string{"delete", "packageinstalls/" + getPackageInstallByPackageName(&kubectl, "pkg.child2.carvel.dev")}, e2e.RunOpts{AllowError: true})
		kubectl.RunWithOpts([]string{"delete", "packageinstalls/" + "parent-pkgi"}, e2e.RunOpts{AllowError: true})
	}

	defer cleanUp()

	packagesYaml := fmt.Sprintf(`
---
apiVersion: data.packaging.carvel.dev/v1alpha1
kind: Package
metadata:
  name: pkg.parent.carvel.dev.1.0.0
spec:
  refName: pkg.parent.carvel.dev
  version: 1.0.0
  dependencies:
  - package:
      refName: pkg.child.carvel.dev
      version:
        constraints: 1.0.0
  - package:
      refName: pkg.child2.carvel.dev
      version:
        constraints: 1.0.0
  template:
    spec:
      fetch:
      - inline:
          paths:
            file.yml: |
              apiVersion: v1
              kind: ConfigMap
              metadata:
                name: configmap-of-parent
      template:
      - ytt:
          paths:
          - file.yml
      deploy:
      - kapp: {}
---
apiVersion: data.packaging.carvel.dev/v1alpha1
kind: Package
metadata:
  name: pkg.child.carvel.dev.1.0.0
spec:
  refName: pkg.child.carvel.dev
  version: 1.0.0
  template:
    spec:
      fetch:
      - inline:
          paths:
            file.yml: |
              apiVersion: v1
              kind: ConfigMap
              metadata:
                name: configmap-of-child1
      template:
      - ytt:
          paths:
          - file.yml
      deploy:
      - kapp: {}
---
apiVersion: data.packaging.carvel.dev/v1alpha1
kind: Package
metadata:
  name: pkg.child2.carvel.dev.1.0.0
spec:
  refName: pkg.child2.carvel.dev
  version: 1.0.0
  template:
    spec:
      fetch:
      - inline:
          paths:
            file.yml: |
              apiVersion: v1
              kind: ConfigMap
              metadata:
                name: configmap-of-child2
      template:
      - ytt:
          paths:
          - file.yml
      deploy:
      - kapp: {}
`)

	packageInstallYaml := fmt.Sprintf(`
---
apiVersion: packaging.carvel.dev/v1alpha1
kind: PackageInstall
metadata:
  name: parent-pkgi
spec:
  serviceAccountName: kappctrl-e2e-ns-sa
  packageRef:
    refName: pkg.parent.carvel.dev
    versionSelection:
      constraints: 1.0.0
  dependencies:
    install: true
`)

	logger.Section("Create Packages", func() {
		kubectl.RunWithOpts([]string{"apply", "-f", "-"}, e2e.RunOpts{StdinReader: strings.NewReader(packagesYaml)})
	})

	logger.Section("Create PackageInstall and verify dependencies installed", func() {
		kubectl.RunWithOpts([]string{"apply", "-f", "-"}, e2e.RunOpts{StdinReader: strings.NewReader(packageInstallYaml)})
		kubectl.Run([]string{"wait", "--for=condition=ReconcileSucceeded", "packageinstalls/" + getPackageInstallByPackageName(&kubectl, "pkg.child.carvel.dev"), "--timeout", "1m"})
		kubectl.Run([]string{"wait", "--for=condition=ReconcileSucceeded", "packageinstalls/" + getPackageInstallByPackageName(&kubectl, "pkg.child2.carvel.dev"), "--timeout", "1m"})
		kubectl.Run([]string{"wait", "--for=condition=ReconcileSucceeded", "packageinstalls/" + "parent-pkgi", "--timeout", "1m"})
	})

}

func getPackageInstallByPackageName(kubectl *e2e.Kubectl, packageName string) string {
	out := kubectl.Run([]string{"get", "pkgi"})
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 1 && fields[1] == packageName {
			return fields[0]
		}
	}
	return ""
}
