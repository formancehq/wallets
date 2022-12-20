package core

import (
	"github.com/imdario/mergo"
)

type Metadata map[string]interface{}

func (m Metadata) Merge(m2 Metadata) Metadata {
	ret := Metadata{}
	if err := mergo.Merge(&ret, m); err != nil {
		panic(err)
	}
	if err := mergo.Merge(&ret, m2); err != nil {
		panic(err)
	}
	return m
}
