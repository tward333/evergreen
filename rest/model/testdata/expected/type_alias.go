// Code generated by rest/model/codegen.go. DO NOT EDIT.

package model

import "github.com/evergreen-ci/evergreen/rest/model"

type APIStructWithAliased struct {
	Foo *string `json:"foo"`
	Bar *string `json:"bar"`
}

func APIStructWithAliasedBuildFromService(t model.StructWithAliased) *APIStructWithAliased {
	m := APIStructWithAliased{}
	m.Foo = StringStringPtr(t.Foo)
	m.Bar = StringStringPtr(t.Bar)
	return &m
}

func APIStructWithAliasedToService(m APIStructWithAliased) *model.StructWithAliased {
	out := &model.StructWithAliased{}
	out.Foo = StringPtrString(m.Foo)
	out.Bar = StringPtrString(m.Bar)
	return out
}