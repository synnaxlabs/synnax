// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type ontology } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { User } from "@/user";

describe("User queries", () => {
  let controller: AbortController;
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  beforeEach(() => {
    controller = new AbortController();
  });

  afterEach(() => {
    controller.abort();
  });

  describe("useDelete", () => {
    it("should delete a single user", async () => {
      const testUser = await client.users.create({
        username: `test-user-${Date.now()}`,
        firstName: "Test",
        lastName: "User",
        password: "password123",
      });

      const { result } = renderHook(() => User.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(testUser.key);
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(client.users.retrieve({ key: testUser.key })).rejects.toThrow();
    });

    it("should delete multiple users", async () => {
      const user1 = await client.users.create({
        username: `user1-${Date.now()}`,
        firstName: "User",
        lastName: "One",
        password: "password123",
      });
      const user2 = await client.users.create({
        username: `user2-${Date.now()}`,
        firstName: "User",
        lastName: "Two",
        password: "password123",
      });

      const { result } = renderHook(() => User.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync([user1.key, user2.key]);
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(client.users.retrieve({ key: user1.key })).rejects.toThrow();
      await expect(client.users.retrieve({ key: user2.key })).rejects.toThrow();
    });
  });

  describe("useRename", () => {
    it("should change a user's username", async () => {
      const testUser = await client.users.create({
        username: `original-${Date.now()}`,
        firstName: "Test",
        lastName: "User",
        password: "password123",
      });

      const newUsername = `renamed-${Date.now()}`;
      const { result } = renderHook(() => User.useRename(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          key: testUser.key,
          username: newUsername,
        });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const retrieved = await client.users.retrieve({ key: testUser.key });
      expect(retrieved.username).toBe(newUsername);
    });
  });

  describe("useRetrieveGroupID", () => {
    it("should retrieve the Users group ID", async () => {
      const { result } = renderHook(() => User.useRetrieveGroupID({}), { wrapper });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(result.current.data).toBeDefined();
      const res = await client.ontology.retrieve(result.current.data as ontology.ID);
      expect(res.name).toBe("Users");
    });
  });

  describe("useRetrieve", () => {
    it("should retrieve a user by key", async () => {
      const testUser = await client.users.create({
        username: `retrieve-test-${Date.now()}`,
        firstName: "Retrieve",
        lastName: "Test",
        password: "password123",
      });

      const { result } = renderHook(() => User.useRetrieve({ key: testUser.key }), {
        wrapper,
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(result.current.data?.key).toEqual(testUser.key);
      expect(result.current.data?.username).toEqual(testUser.username);
      expect(result.current.data?.firstName).toEqual(testUser.firstName);
      expect(result.current.data?.lastName).toEqual(testUser.lastName);
    });

    it("should retrieve the current authenticated user when no key provided", async () => {
      const { result } = renderHook(() => User.useRetrieve({}), { wrapper });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(result.current.data).toBeDefined();
      expect(result.current.data?.key).toEqual(client.auth?.user?.key);
      expect(result.current.data?.username).toEqual(client.auth?.user?.username);
    });

    it("should cache retrieved users", async () => {
      const testUser = await client.users.create({
        username: `cached-user-${Date.now()}`,
        firstName: "Cached",
        lastName: "User",
        password: "password123",
      });

      const { result: result1 } = renderHook(
        () => User.useRetrieve({ key: testUser.key }),
        { wrapper },
      );
      await waitFor(() => expect(result1.current.variant).toEqual("success"));

      const { result: result2 } = renderHook(
        () => User.useRetrieve({ key: testUser.key }),
        { wrapper },
      );
      await waitFor(() => expect(result2.current.variant).toEqual("success"));
      expect(result2.current.data).toEqual(result1.current.data);
    });
  });

  describe("useForm", () => {
    describe("create mode", () => {
      it("should initialize with default values for new user", async () => {
        const { result } = renderHook(() => User.useForm({ query: {} }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        const formData = result.current.form.value();
        expect(formData.key).toBeUndefined();
        expect(formData.username).toBe("");
        expect(formData.firstName).toBe("");
        expect(formData.lastName).toBe("");
        expect(formData.password).toBe("");
      });

      it("should create a new user on save", async () => {
        const { result } = renderHook(() => User.useForm({ query: {} }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        const timestamp = Date.now();
        act(() => {
          result.current.form.set("username", `test-form-${timestamp}`);
          result.current.form.set("firstName", "Form");
          result.current.form.set("lastName", "Test");
          result.current.form.set("password", "password123");
        });

        await act(async () => {
          result.current.save();
        });

        await waitFor(() => {
          expect(result.current.variant).toBe("success");
        });
      });

      it("should validate required fields", async () => {
        const { result } = renderHook(() => User.useForm({ query: {} }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        await act(async () => {
          result.current.save();
        });

        const usernameField = result.current.form.get("username");
        const passwordField = result.current.form.get("password");
        const firstNameField = result.current.form.get("firstName");
        const lastNameField = result.current.form.get("lastName");

        expect(usernameField.status.variant).toBe("error");
        expect(passwordField.status.variant).toBe("error");
        expect(firstNameField.status.variant).toBe("error");
        expect(lastNameField.status.variant).toBe("error");
      });
    });

    describe("validation", () => {
      it("should validate username field", async () => {
        const { result } = renderHook(() => User.useForm({ query: {} }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        act(() => {
          result.current.form.set("username", "");
        });

        const isValid = result.current.form.validate("username");
        expect(isValid).toBe(false);

        const usernameField = result.current.form.get("username");
        expect(usernameField.status.variant).toBe("error");
        expect(usernameField.status.message).toContain("Username is required");
      });

      it("should validate all required fields", async () => {
        const { result } = renderHook(() => User.useForm({ query: {} }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        const isValid = result.current.form.validate();
        expect(isValid).toBe(false);

        expect(result.current.form.get("username").status.variant).toBe("error");
        expect(result.current.form.get("password").status.variant).toBe("error");
        expect(result.current.form.get("firstName").status.variant).toBe("error");
        expect(result.current.form.get("lastName").status.variant).toBe("error");
      });
    });
  });
});
