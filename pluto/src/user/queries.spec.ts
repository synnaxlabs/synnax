// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestUser,
  TEST_CREDENTIALS,
} from "@synnaxlabs/client/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import {
  afterEach,
  assert,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";

import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil/Synnax";
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
      const testUser = await client.users.create({ name: "Test User" });

      const { result } = renderHook(() => User.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(testUser.key);
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(client.users.retrieve(testUser.key)).rejects.toThrow();
    });

    it("should delete multiple users", async () => {
      const user1 = await client.users.create({ name: "User One" });
      const user2 = await client.users.create({ name: "User Two" });

      const { result } = renderHook(() => User.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync([user1.key, user2.key]);
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(client.users.retrieve(user1.key)).rejects.toThrow();
      await expect(client.users.retrieve(user2.key)).rejects.toThrow();
    });
  });

  describe("useRename", () => {
    it("should change a user's name", async () => {
      const testUser = await client.users.create({ name: "Test User" });

      const { result } = renderHook(() => User.useRename(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: testUser.key, name: "Renamed User" });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const retrieved = await client.users.retrieve(testUser.key);
      expect(retrieved.name).toBe("Renamed User");
    });

    it("should apply the rename optimistically", async () => {
      const testUser = await client.users.create({ name: "Test User" });
      const afterOptimistic = vi.fn();
      const { result } = renderHook(() => User.useRename({ afterOptimistic }), {
        wrapper,
      });
      await act(async () => {
        await result.current.updateAsync({ key: testUser.key, name: "Renamed User" });
      });
      expect(afterOptimistic).toHaveBeenCalledOnce();
    });
  });

  describe("useGroupID", () => {
    it("should retrieve the Users group ID", async () => {
      const { result } = await renderHookSuspended(() => User.useGroupID({}), {
        wrapper,
      });

      await waitFor(() => expect(result.current).not.toBeNull());
      const res = await client.ontology.retrieve(result.current as ontology.ID);
      expect(res.name).toBe("Users");
    });
  });

  describe("use", () => {
    it("should retrieve a user by key", async () => {
      const testUser = await client.users.create({ name: "Retrieve Test" });

      const { result } = await renderHookSuspended(
        () => User.use({ key: testUser.key }),
        {
          wrapper,
        },
      );

      await waitFor(() => {
        expect(result.current).not.toBeNull();
      });

      expect(result.current?.key).toEqual(testUser.key);
      expect(result.current?.name).toEqual(testUser.name);
    });

    it("should retrieve the current authenticated user when no key provided", async () => {
      const { result } = await renderHookSuspended(() => User.use({}), {
        wrapper,
      });

      await waitFor(() => {
        expect(result.current).not.toBeNull();
      });

      expect(result.current).not.toBeNull();
      const { username } = TEST_CREDENTIALS;
      const signedIn = await client.users.retrieve({ username });
      expect(result.current?.key).toEqual(signedIn.key);
    });

    it("should update the current user on a rename from another client", async () => {
      const { user: u, username } = await createTestUser(client);
      const roles = await client.access.roles.retrieve({});
      const viewer = roles.find((r) => r.name === "Viewer");
      assert(viewer != null);
      await client.access.roles.assign({ user: u.key, role: viewer.key });
      const userClient = createTestClient({ username, password: "test" });
      const { result } = renderHook(() => User.useResult({}), {
        wrapper: createSynnaxWrapper({ client: userClient }),
      });
      // Changes made while the stream is still opening are lost, so open it first.
      await userClient.connect();
      await waitFor(() => expect(result.current.data?.key).toEqual(u.key));
      await client.users.rename(u.key, "Abigail Adams");
      await waitFor(() => expect(result.current.data?.name).toEqual("Abigail Adams"));
    });

    it("should cache retrieved users", async () => {
      const testUser = await client.users.create({ name: "Cached User" });

      const { result: result1 } = await renderHookSuspended(
        () => User.use({ key: testUser.key }),
        { wrapper },
      );
      await waitFor(() => expect(result1.current).not.toBeNull());

      const { result: result2 } = await renderHookSuspended(
        () => User.use({ key: testUser.key }),
        { wrapper },
      );
      await waitFor(() => expect(result2.current).not.toBeNull());
      expect(result2.current).toEqual(result1.current);
    });
  });

  describe("useForm", () => {
    describe("create mode", () => {
      it("should initialize with default values for new user", async () => {
        const { result } = renderHook(() => User.useForm({ query: null }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        const formData = result.current.form.value();
        expect(formData.key).toBeUndefined();
        expect(formData.username).toBe("");
        expect(formData.name).toBe("");
        expect(formData.password).toBe("");
      });

      it("should create a new user on save", async () => {
        const { result } = renderHook(() => User.useForm({ query: null }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        const username = `test-form-${Date.now()}`;
        act(() => {
          result.current.form.set("username", username);
          result.current.form.set("name", "Form Test");
          result.current.form.set("password", "password123");
        });

        await act(async () => {
          result.current.save();
        });

        await waitFor(() => {
          expect(result.current.variant).toBe("success");
        });
        await waitFor(async () => {
          const created = await client.users.retrieve({ username });
          expect(created.name).toBe("Form Test");
        });
      });

      it("should remove the new user when its username is taken", async () => {
        const { username } = await createTestUser(client);
        const { result } = renderHook(() => User.useForm({ query: null }), {
          wrapper,
        });
        await waitFor(() => expect(result.current.variant).toBe("success"));
        const name = `taken-${Date.now()}`;
        act(() => {
          result.current.form.set("username", username);
          result.current.form.set("name", name);
          result.current.form.set("password", "password123");
        });
        await act(async () => {
          result.current.save();
        });
        await waitFor(() => expect(result.current.variant).toBe("error"));
        const users = await client.users.retrieve({});
        expect(users.filter((u) => u.name === name)).toHaveLength(0);
      });

      it("should validate required fields", async () => {
        const { result } = renderHook(() => User.useForm({ query: null }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        await act(async () => {
          result.current.save();
        });

        const usernameField = result.current.form.get("username");
        const passwordField = result.current.form.get("password");
        const nameField = result.current.form.get("name");

        expect(usernameField.status.variant).toBe("error");
        expect(passwordField.status.variant).toBe("error");
        expect(nameField.status.variant).toBe("error");
      });
    });

    describe("validation", () => {
      it("should validate username field", async () => {
        const { result } = renderHook(() => User.useForm({ query: null }), {
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
        expect(usernameField.status.message).toContain("username is required");
      });

      it("should validate all required fields", async () => {
        const { result } = renderHook(() => User.useForm({ query: null }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        const isValid = result.current.form.validate();
        expect(isValid).toBe(false);

        expect(result.current.form.get("username").status.variant).toBe("error");
        expect(result.current.form.get("password").status.variant).toBe("error");
        expect(result.current.form.get("name").status.variant).toBe("error");
      });
    });
  });
});
