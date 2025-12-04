// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, NotFoundError } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Role } from "@/access/role";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

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
      expect(result.current.data).toHaveLength(2);
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

  describe("useRetrieve", () => {
    it("should retrieve a single role by key", async () => {
      const testRole = await client.access.roles.create({
        name: "singleRole",
        description: "Single role description",
      });

      const { result } = renderHook(() => Role.useRetrieve({ key: testRole.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data?.key).toEqual(testRole.key);
      expect(result.current.data?.name).toEqual("singleRole");
      expect(result.current.data?.description).toEqual("Single role description");
    });

    it("should handle retrieve with valid role key", async () => {
      const role = await client.access.roles.create({
        name: "validRole",
        description: "Valid description",
      });

      const { result } = renderHook(() => Role.useRetrieve({ key: role.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data).toBeDefined();
      expect(result.current.data?.key).toEqual(role.key);
      expect(result.current.data?.description).toEqual("Valid description");
    });
  });

  describe("useRename", () => {
    it("should correctly rename a role", async () => {
      const role = await client.access.roles.create({
        name: "testRole",
        description: "Test description",
      });

      const { result } = renderHook(
        () => ({
          retrieve: Role.useRetrieve({ key: role.key }),
          rename: Role.useRename(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.rename.update({ key: role.key, name: "newName" });
      });
      await waitFor(() =>
        expect(result.current.retrieve.data?.name).toEqual("newName"),
      );
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
        await expect(client.access.roles.retrieve({ key: role.key })).rejects.toThrow(
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
        await expect(client.access.roles.retrieve({ key: role1.key })).rejects.toThrow(
          NotFoundError,
        );
        await expect(client.access.roles.retrieve({ key: role2.key })).rejects.toThrow(
          NotFoundError,
        );
      });
    });
  });
});
