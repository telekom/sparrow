// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

// OpenapiFromPerfData takes in check perfdata and returns an openapi3.SchemaRef of a result wrapping the perfData
// this is a workaround, since the openapi3gen.NewSchemaRefForValue function does not work with any types
func OpenapiFromPerfData[T any](data T) (*openapi3.SchemaRef, error) {
	checkSchema, err := openapi3gen.NewSchemaRefForValue(Result{}, openapi3.Schemas{})
	if err != nil {
		return nil, err
	}
	perfdataSchema, err := openapi3gen.NewSchemaRefForValue(data, openapi3.Schemas{}, openapi3gen.UseAllExportedFields())
	if err != nil {
		return nil, err
	}

	checkSchema.Value.Properties["data"] = perfdataSchema
	return checkSchema, nil
}
