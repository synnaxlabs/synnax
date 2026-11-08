// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access, NotFoundError } from "@synnaxlabs/client";
import {
  createEmptyRole,
  createTestClient,
  TEST_CREDENTIALS,
} from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Role } from "@/access/role";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const path = Role.changeRoleFormPath;

const client = createTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useList", () => {
    it("should return a list of role keys", async () => {
      const role1 = await client.access.roles.create({
        name: "role1",
        description: "First role",
      });
      const role2 = await client.access.roles.create({
        name: "role2",
        description: "Second role",
      });

      const { result } = renderHook(() => Role.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(role1.key);
      expect(result.current.data).toContain(role2.key);
    });

    it("should get individual roles using getItem", async () => {
      const testRole = await client.access.roles.create({
        name: "testRole",
        description: "Test role description",
      });

      const { result } = renderHook(() => Role.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedRole = result.current.getItem(testRole.key);
      expect(retrievedRole?.key).toEqual(testRole.key);
      expect(retrievedRole?.name).toEqual("testRole");
      expect(retrievedRole?.description).toEqual("Test role description");
    });

    it("should handle pagination with limit and offset", async () => {
      for (let i = 0; i < 5; i++)
        await client.access.roles.create({
          name: `paginationRole${i}`,
          description: `Description ${i}`,
        });

      const { result } = renderHook(() => Role.useList(), { wrapper });
      act(() => {
        result.current.retrieve({ limit: 2, offset: 1 });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
    });

    it("should return all roles when no pagination params provided", async () => {
      const role1 = await client.access.roles.create({
        name: "allRoles1",
        description: "First",
      });
      const role2 = await client.access.roles.create({
        name: "allRoles2",
        description: "Second",
      });

      const { result } = renderHook(() => Role.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(role1.key);
      expect(result.current.data).toContain(role2.key);
    });

    it("should update the list when a role is created", async () => {
      const { result } = renderHook(() => Role.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const initialLength = result.current.data.length;

      const newRole = await client.access.roles.create({
        name: "newRole",
        description: "Newly created role",
      });

      await waitFor(() => {
        expect(result.current.data.length).toBeGreaterThanOrEqual(initialLength + 1);
        expect(result.current.data).toContain(newRole.key);
      });
    });

    it("should remove role from list when deleted", async () => {
      const testRole = await client.access.roles.create({
        name: "toDeleteRole",
        description: "Will be deleted",
      });

      const { result } = renderHook(() => Role.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(testRole.key);

      await client.access.roles.delete(testRole.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(testRole.key);
      });
    });

    it("should handle multiple role updates simultaneously", async () => {
      const role1 = await client.access.roles.create({
        name: "multiUpdate1",
        description: "First",
      });
      const role2 = await client.access.roles.create({
        name: "multiUpdate2",
        description: "Second",
      });

      const { result } = renderHook(() => Role.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      await Promise.all([
        client.access.roles.create({ ...role1, name: "updated1" }),
        client.access.roles.create({ ...role2, name: "updated2" }),
      ]);

      await waitFor(() => {
        expect(result.current.getItem(role1.key)?.name).toEqual("updated1");
        expect(result.current.getItem(role2.key)?.name).toEqual("updated2");
      });
    });
  });

  describe("use", () => {
    it("should retrieve a single role by key", async () => {
      const testRole = await client.access.roles.create({
        name: "singleRole",
        description: "Single role description",
      });

      const { result } = await renderHookSuspended(
        () => Role.use({ key: testRole.key }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current).not.toBeNull());

      expect(result.current?.key).toEqual(testRole.key);
      expect(result.current?.name).toEqual("singleRole");
      expect(result.current?.description).toEqual("Single role description");
    });

    it("should handle retrieve with valid role key", async () => {
      const role = await client.access.roles.create({
        name: "validRole",
        description: "Valid description",
      });

      const { result } = await renderHookSuspended(() => Role.use({ key: role.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current).not.toBeNull());

      expect(result.current).not.toBeNull();
      expect(result.current?.key).toEqual(role.key);
      expect(result.current?.description).toEqual("Valid description");
    });
  });

  describe("useResultForUser", () => {
    const createUser = async (prefix: string, role: access.role.Key) =>
      await client.users.create({
        username: `${prefix}-${id.create()}`,
        name: "Test User",
        password: "password123",
        role,
      });

    it("should return the role the user holds", async () => {
      const role = await createEmptyRole(client);
      const u = await createUser("for-user", role.key);

      const { result } = renderHook(() => Role.useResultForUser({ user: u.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.data?.key).toEqual(role.key));
      expect(result.current.data?.name).toEqual(role.name);
    });

    it("should pick up a role given by another client", async () => {
      const remote = createTestClient();
      const [from, to] = await Promise.all([
        createEmptyRole(client),
        createEmptyRole(client),
      ]);
      const u = await createUser("late-assign", from.key);

      const { result } = renderHook(() => Role.useResultForUser({ user: u.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.data?.key).toEqual(from.key));
      await remote.users.setRole(u.key, to.key);
      await waitFor(() => expect(result.current.data?.key).toEqual(to.key));
    });

    it("should pick up an edit to the held role by another client", async () => {
      const remote = createTestClient();
      const role = await client.access.roles.create({
        name: `edited-${id.create()}`,
        description: "Before",
      });
      const u = await createUser("role-edit", role.key);

      const { result } = renderHook(() => Role.useResultForUser({ user: u.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.data?.description).toEqual("Before"));
      await remote.access.roles.create({ ...role, description: "After" });
      await waitFor(() => expect(result.current.data?.description).toEqual("After"));
    });

    it("should not read at all when the query is null", async () => {
      const { result } = renderHook(() => Role.useResultForUser(null), { wrapper });
      expect(result.current.data).toBeUndefined();
    });
  });

  describe("useRename", () => {
    it("should correctly rename a role", async () => {
      const role = await client.access.roles.create({
        name: "testRole",
        description: "Test description",
      });

      const { result } = await renderHookSuspended(
        () => ({
          retrieve: Role.use({ key: role.key }),
          rename: Role.useRename(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.rename.update({ key: role.key, name: "newName" });
      });
      await waitFor(() => expect(result.current.retrieve?.name).toEqual("newName"));
    });

    it("should apply the rename optimistically", async () => {
      const role = await client.access.roles.create({
        name: "testRole",
        description: "Test description",
      });
      const afterOptimistic = vi.fn();
      const { result } = renderHook(() => Role.useRename({ afterOptimistic }), {
        wrapper,
      });
      await act(async () => {
        await result.current.updateAsync({ key: role.key, name: "newName" });
      });
      expect(afterOptimistic).toHaveBeenCalledOnce();
    });
  });

  describe("useDelete", () => {
    it("should correctly delete a role", async () => {
      const role = await client.access.roles.create({
        name: "testRole",
        description: "Test description",
      });

      const { result } = renderHook(() => Role.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(role.key);
      });
      await waitFor(async () => {
        await expect(client.access.roles.retrieve(role.key)).rejects.toThrow(
          NotFoundError,
        );
      });
    });

    it("should delete multiple roles", async () => {
      const role1 = await client.access.roles.create({
        name: "deleteRole1",
        description: "First",
      });
      const role2 = await client.access.roles.create({
        name: "deleteRole2",
        description: "Second",
      });

      const { result } = renderHook(() => Role.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync([role1.key, role2.key]);
      });
      await waitFor(async () => {
        await expect(client.access.roles.retrieve(role1.key)).rejects.toThrow(
          NotFoundError,
        );
        await expect(client.access.roles.retrieve(role2.key)).rejects.toThrow(
          NotFoundError,
        );
      });
    });

    it("should report an error when a user holds the role", async () => {
      const role = await createEmptyRole(client);
      const u = await client.users.create({
        username: `holder-${id.create()}`,
        password: "password123",
        role: role.key,
      });

      const { result } = renderHook(() => Role.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(role.key);
      });
      expect(result.current.variant).toEqual("error");
      expect(result.current.status.message).toEqual("Failed to delete role");
      expect(result.current.status.description).toEqual(
        `cannot delete role ${role.name} because user ${u.name} ` +
          "uses it as its role: validation error",
      );
      expect((await client.access.roles.retrieve(role.key)).key).toEqual(role.key);
    });
  });

  describe("useChangeRoleForm", () => {
    it("should retrieve the current role for a user", async () => {
      const role = await createEmptyRole(client);
      const testUser = await client.users.create({
        username: `retrieve-for-${id.create()}`,
        name: "Test User",
        password: "password123",
        role: role.key,
      });

      const { result } = await renderHookSuspended(
        () => Role.useChangeRoleForm({ query: { key: testUser.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.form.value().role).toEqual(role.key);
    });

    it("should change a user's role when saved", async () => {
      const [from, to] = await Promise.all([
        createEmptyRole(client),
        createEmptyRole(client),
      ]);
      const testUser = await client.users.create({
        username: `change-role-${id.create()}`,
        name: "Change Role",
        password: "password123",
        role: from.key,
      });

      const { result } = await renderHookSuspended(
        () => Role.useChangeRoleForm({ query: { key: testUser.key } }),
        { wrapper },
      );

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.value().role).toEqual(from.key);

      act(() => {
        result.current.form.set(path.role, to.key);
      });

      await act(async () => {
        result.current.save();
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const stored = await createTestClient().users.retrieve(testUser.key);
      expect(stored.role).toEqual(to.key);
    });

    it("should report an error when the Core refuses the change", async () => {
      const role = await createEmptyRole(client);
      const root = await client.users.retrieve({
        username: TEST_CREDENTIALS.username,
      });

      const { result } = await renderHookSuspended(
        () => Role.useChangeRoleForm({ query: { key: root.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      act(() => {
        result.current.form.set(path.role, role.key);
      });
      await act(async () => {
        result.current.save();
      });

      await waitFor(() => expect(result.current.variant).toEqual("error"));
      expect(result.current.status.description).toEqual(
        `cannot change the role of root user ${root.name}: validation error`,
      );
      const stored = await createTestClient().users.retrieve(root.key);
      expect(stored.role).toEqual(root.role);
    });
  });
});
