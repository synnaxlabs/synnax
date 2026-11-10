// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, NotFoundError } from "@synnaxlabs/client";
import {
  createEmptyRole,
  createTestClient,
  TEST_CREDENTIALS,
} from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Role } from "@/access/role";
import { Flux } from "@/flux";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const path = Role.changeRoleFormPath;

const client = createTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("list", () => {
    it("should list the matching roles in name order", async () => {
      const prefix = id.create();
      const b = await client.access.roles.create({ name: `${prefix}-b` });
      const a = await client.access.roles.create({ name: `${prefix}-a` });
      const { result } = renderHook(
        () =>
          Flux.useList((c) => c.access.roles, {
            name: "roles",
            query: {
              where: { name: { in: [a.name, b.name] } },
              order: { field: "name" },
            },
          }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual([a.key, b.key]);
      });
      expect(result.current.getItem(a.key)?.name).toEqual(a.name);
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

  describe("useMultiple", () => {
    it("should retrieve the roles with the keys", async () => {
      const prefix = id.create();
      const a = await client.access.roles.create({ name: `${prefix}-a` });
      const b = await client.access.roles.create({ name: `${prefix}-b` });
      const { result } = await renderHookSuspended(
        () => Role.useMultiple({ keys: [a.key, b.key] }),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current?.map(({ key }) => key).sort()).toEqual(
          [a.key, b.key].sort(),
        ),
      );
    });

    it("should update when one of the roles changes", async () => {
      const prefix = id.create();
      const a = await client.access.roles.create({ name: `${prefix}-a` });
      const b = await client.access.roles.create({ name: `${prefix}-b` });
      const { result } = await renderHookSuspended(
        () => Role.useMultiple({ keys: [a.key, b.key] }),
        { wrapper },
      );
      await waitFor(() => expect(result.current).toHaveLength(2));
      await act(async () => {
        await client.access.roles.rename(a.key, `${prefix}-renamed`);
      });
      await waitFor(() =>
        expect(result.current).toContainEqual(
          expect.objectContaining({ key: a.key, name: `${prefix}-renamed` }),
        ),
      );
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
      await remote.access.roles.dispatch(
        role.key,
        access.role.set({ path: ["description"], value: "After" }),
      );
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
      const { result } = renderHook(() => Role.useRename(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: role.key, name: "newName" });
      });
      expect(client.access.roles.watch(role.key).getCached()).toMatchObject({
        name: "newName",
      });
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
