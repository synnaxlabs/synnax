// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package config_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

var errTest = errors.New("config is invalid")

type testConfig struct {
	name           string
	secondaryField bool
}

var _ config.Config[testConfig] = testConfig{}

func (c testConfig) Override(other testConfig) testConfig {
	c.name = other.name
	if other.secondaryField {
		c.secondaryField = true
	}
	return c
}

func (c testConfig) Validate() error {
	if c.name == "error" {
		return errTest
	}
	return nil
}

var (
	defaultConfig   = testConfig{name: "default"}
	successConfig   = testConfig{name: "success"}
	errorConfig     = testConfig{name: "error"}
	secondaryConfig = testConfig{name: "secondary", secondaryField: true}
)

var _ = Describe("Config", func() {
	Describe("New", func() {
		It("Should succeed if the initial configuration is valid", func() {
			Expect(config.New(defaultConfig)).To(Equal(defaultConfig))
		})
		It("Should fail if the initial configuration is invalid", func() {
			Expect(config.New(errorConfig)).Error().To(MatchError(errTest))
		})
		It("Should succeed if the initial configuration is valid and the override is valid", func() {
			Expect(config.New(defaultConfig, successConfig)).To(Equal(successConfig))
		})
		It("Should fail if the initial configuration is valid and the override is invalid", func() {
			Expect(config.New(defaultConfig, errorConfig)).Error().To(MatchError(errTest))
		})
		It("Should succeed if the initial configuration is invalid and the override is valid", func() {
			Expect(config.New(errorConfig, successConfig)).To(Equal(successConfig))
		})
		It("Should fail if the initial configuration is invalid and the override is invalid", func() {
			Expect(config.New(errorConfig, errorConfig)).Error().To(MatchError(errTest))
		})
		It("Should succeed with multiple overrides", func() {
			c := MustSucceed(config.New(defaultConfig, secondaryConfig, successConfig))
			Expect(c.name).To(Equal(successConfig.name))
			Expect(c.secondaryField).To(BeTrue())
		})
		It("Should succeed with multiple overrides even if one is invalid", func() {
			Expect(config.New(defaultConfig, errorConfig, successConfig)).To(Equal(successConfig))
		})
	})
	DescribeTable("Bool", func(b bool) {
		Expect(*config.Bool(b)).To(Equal(b))
	},
		Entry("true", true),
		Entry("false", false),
	)
	Describe("True", func() {
		It("Should return a pointer to a true boolean", func() {
			Expect(*config.True()).To(BeTrue())
		})
	})
	Describe("False", func() {
		It("Should return a pointer to a false boolean", func() {
			Expect(*config.False()).To(BeFalse())
		})
	})
})
