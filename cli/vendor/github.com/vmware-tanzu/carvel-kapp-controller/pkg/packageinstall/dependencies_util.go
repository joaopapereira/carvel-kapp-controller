// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package packageinstall

import (
	"math/rand"
	"time"
)

// generateRandomToken generates a random token of specified length
func generateRandomToken(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	token := make([]byte, length)
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	for i := range token {
		token[i] = charset[r.Intn(len(charset))]
	}
	return string(token)
}
