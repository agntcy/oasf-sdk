// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validator

type option struct {
	schemaURL string
	strict    bool
}

type Option func(*option)

func WithSchemaURL(url string) Option {
	return func(o *option) {
		o.schemaURL = url
	}
}

func WithStrict(strict bool) Option {
	return func(o *option) {
		o.strict = strict
	}
}
