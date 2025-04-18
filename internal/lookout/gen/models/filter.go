// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"encoding/json"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// Filter filter
//
// swagger:model filter
type Filter struct {

	// field
	// Required: true
	// Min Length: 1
	Field string `json:"field"`

	// is annotation
	IsAnnotation bool `json:"isAnnotation,omitempty"`

	// match
	// Required: true
	// Enum: ["exact","anyOf","startsWith","contains","greaterThan","lessThan","greaterThanOrEqualTo","lessThanOrEqualTo","exists"]
	Match string `json:"match"`

	// value
	// Required: true
	Value interface{} `json:"value"`
}

// Validate validates this filter
func (m *Filter) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateField(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateMatch(formats); err != nil {
		res = append(res, err)
	}

	if err := m.validateValue(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Filter) validateField(formats strfmt.Registry) error {

	if err := validate.RequiredString("field", "body", m.Field); err != nil {
		return err
	}

	if err := validate.MinLength("field", "body", m.Field, 1); err != nil {
		return err
	}

	return nil
}

var filterTypeMatchPropEnum []interface{}

func init() {
	var res []string
	if err := json.Unmarshal([]byte(`["exact","anyOf","startsWith","contains","greaterThan","lessThan","greaterThanOrEqualTo","lessThanOrEqualTo","exists"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		filterTypeMatchPropEnum = append(filterTypeMatchPropEnum, v)
	}
}

const (

	// FilterMatchExact captures enum value "exact"
	FilterMatchExact string = "exact"

	// FilterMatchAnyOf captures enum value "anyOf"
	FilterMatchAnyOf string = "anyOf"

	// FilterMatchStartsWith captures enum value "startsWith"
	FilterMatchStartsWith string = "startsWith"

	// FilterMatchContains captures enum value "contains"
	FilterMatchContains string = "contains"

	// FilterMatchGreaterThan captures enum value "greaterThan"
	FilterMatchGreaterThan string = "greaterThan"

	// FilterMatchLessThan captures enum value "lessThan"
	FilterMatchLessThan string = "lessThan"

	// FilterMatchGreaterThanOrEqualTo captures enum value "greaterThanOrEqualTo"
	FilterMatchGreaterThanOrEqualTo string = "greaterThanOrEqualTo"

	// FilterMatchLessThanOrEqualTo captures enum value "lessThanOrEqualTo"
	FilterMatchLessThanOrEqualTo string = "lessThanOrEqualTo"

	// FilterMatchExists captures enum value "exists"
	FilterMatchExists string = "exists"
)

// prop value enum
func (m *Filter) validateMatchEnum(path, location string, value string) error {
	if err := validate.EnumCase(path, location, value, filterTypeMatchPropEnum, true); err != nil {
		return err
	}
	return nil
}

func (m *Filter) validateMatch(formats strfmt.Registry) error {

	if err := validate.RequiredString("match", "body", m.Match); err != nil {
		return err
	}

	// value enum
	if err := m.validateMatchEnum("match", "body", m.Match); err != nil {
		return err
	}

	return nil
}

func (m *Filter) validateValue(formats strfmt.Registry) error {

	if m.Value == nil {
		return errors.Required("value", "body", nil)
	}

	return nil
}

// ContextValidate validates this filter based on context it is used
func (m *Filter) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *Filter) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Filter) UnmarshalBinary(b []byte) error {
	var res Filter
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
