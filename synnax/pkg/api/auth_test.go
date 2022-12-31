// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api"
	apierrors "github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/api/mock"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/password"
)

var testCreds = auth.InsecureCredentials{
	Username: "test",
	Password: "test",
}

var _ = Describe("AuthService", Ordered, func() {
	var (
		builder *mock.Builder
		prov    api.Provider
		svc     *api.AuthService
	)
	BeforeAll(func() {
		builder = mock.New()
		prov = builder.New()
		svc = api.NewAuthServer(prov)
	})
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Describe("New", func() {
		It("Should register a new user", func() {
			tr, err := svc.Register(ctx, api.RegistrationRequest{InsecureCredentials: testCreds})
			Expect(err).To(MatchError(apierrors.Nil))
			Expect(tr.Token).ToNot(BeEmpty())
			Expect(tr.User.Key).ToNot(BeEmpty())
		})
		It("Should return an error if the user already exists", func() {
			_, err := svc.Register(ctx, api.RegistrationRequest{InsecureCredentials: testCreds})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(apierrors.General(errors.New("[auth] - username already registered"))))
		})
		It("Should return an error if the user does not provide a field", func() {
			_, err := svc.Register(ctx, api.RegistrationRequest{})
			Expect(err).To(Equal(apierrors.Validation(apierrors.Fields{
				{
					Field:   "username",
					Message: "required",
				},
				{
					Field:   "password",
					Message: "required",
				},
			})))
		})
	})
	Describe("Login", func() {
		It("Should authenticate the user successfully", func() {
			tr, err := svc.Login(ctx, testCreds)
			Expect(err).To(MatchError(apierrors.Nil))
			Expect(tr.Token).ToNot(BeEmpty())
		})
		It("Should return an InvalidCredentials error if the creds are invalid", func() {
			tr, err := svc.Login(ctx, auth.InsecureCredentials{Username: "test", Password: "blabla"})
			Expect(err).To(MatchError(apierrors.Auth(auth.InvalidCredentials)))
			Expect(tr.Token).To(BeEmpty())
		})
		It("Should return an InvalidCredentials error if the user can't be found", func() {
			tr, err := svc.Login(ctx, auth.InsecureCredentials{Username: "jeff", Password: "blabla"})
			Expect(err).To(MatchError(apierrors.Auth(auth.InvalidCredentials)))
			Expect(tr.Token).To(BeEmpty())
		})
		It("Should return a Validation error if the username field is empty", func() {
			tr, err := svc.Login(ctx, auth.InsecureCredentials{Password: "test"})
			Expect(err.Type).To(Equal(apierrors.TypeValidation))
			Expect(err.Err).To(Equal(apierrors.Fields{{Field: "username", Message: "required"}}))
			Expect(tr.Token).To(BeEmpty())
		})
		It("Should return a Validation error it the caller provides an empty username or password field", func() {
			tr, err := svc.Login(ctx, auth.InsecureCredentials{})
			Expect(err.Type).To(Equal(apierrors.TypeValidation))
			Expect(err.Err).To(Equal(apierrors.Fields{
				{Field: "username", Message: "required"},
				{Field: "password", Message: "required"},
			}))
			Expect(tr.Token).To(BeEmpty())
		})
	})
	Describe("ChangePassword", func() {
		It("Should change the users password successfully", func() {
			var pass password.Raw = "newPass"
			Expect(svc.ChangePassword(ctx, api.ChangePasswordRequest{
				InsecureCredentials: testCreds,
				NewPassword:         pass,
			})).To(MatchError(apierrors.Nil))
			testCreds.Password = pass
		})
		It("Should return an error if the caller provides no new password", func() {
			Expect(svc.ChangePassword(ctx, api.ChangePasswordRequest{
				InsecureCredentials: testCreds,
			})).To(Equal(apierrors.Validation(apierrors.Fields{{
				Field: "new_password", Message: "required",
			}})))
		})
	})
	Describe("ChangeUsername", func() {
		It("Should change the users username successfully", func() {
			Expect(svc.ChangeUsername(ctx, api.ChangeUsernameRequest{
				InsecureCredentials: testCreds,
				NewUsername:         "newUser",
			})).To(MatchError(apierrors.Nil))
			testCreds.Username = "newUser"
		})
		It("Should return an error if the caller provides no new username", func() {
			Expect(svc.ChangeUsername(ctx, api.ChangeUsernameRequest{
				InsecureCredentials: testCreds,
			})).To(Equal(apierrors.Validation(apierrors.Fields{{
				Field: "new_username", Message: "required",
			}})))
		})
	})
})
