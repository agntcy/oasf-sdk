// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validator

type option struct {
	schemaURL string
}

type Option func(*option)

func WithSchemaURL(url string) Option {
	return func(o *option) {
		o.schemaURL = url
	}
}
