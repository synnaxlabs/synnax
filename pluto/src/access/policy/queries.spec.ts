// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access, NotFoundError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { assert, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Policy } from "@/access/policy";
import { Flux } from "@/flux";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const path = Policy.formPath;

const client = createTestClient();

// The role that holds every policy these specs create.
let owner: access.role.Role;
beforeAll(async () => {
  owner = await client.access.roles.create({ name: "policy-queries" });
});

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("list", () => {
    it("should list the policies of a role in name order", async () => {
      const role = await client.access.roles.create({ name: "policy-list" });
      const b = await client.access.policies.create({
        role: role.key,
        name: "b",
        objects: [{ type: "channel", key: "1" }],
        actions: ["create"],
      });
      const a = await client.access.policies.create({
        role: role.key,
        name: "a",
        objects: [{ type: "channel", key: "2" }],
        actions: ["delete"],
      });
      const { result } = renderHook(
        () =>
          Flux.useList((c) => c.access.policies, {
            name: "policies",
            query: { where: { roles: [role.key] }, order: { field: "name" } },
          }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual([a.key, b.key]);
      });
      expect(result.current.getItem(a.key)?.name).toEqual("a");
    });
  });

  describe("use", () => {
    it("should retrieve a single policy by key", async () => {
      const testPolicy = await client.access.policies.create({
        role: owner.key,
        name: "singlePolicy",
        objects: [{ type: "channel", key: "single" }],
        actions: ["create", "retrieve", "update"],
      });

      const { result } = await renderHookSuspended(
        () => Policy.use({ key: testPolicy.key }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current).not.toBeNull());

      expect(result.current?.key).toEqual(testPolicy.key);
      expect(result.current?.name).toEqual("singlePolicy");
      expect(result.current?.actions).toContain("create");
    });

    it("should handle retrieve with valid policy key", async () => {
      const policy = await client.access.policies.create({
        role: owner.key,
        name: "validPolicy",
        objects: [{ type: "user", key: "user1" }],
        actions: ["delete"],
      });

      const { result } = await renderHookSuspended(
        () => Policy.use({ key: policy.key }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current).not.toBeNull());

      expect(result.current).not.toBeNull();
      expect(result.current?.key).toEqual(policy.key);
    });
  });

  describe("useRename", () => {
    it("should correctly rename a policy", async () => {
      const policy = await client.access.policies.create({
        role: owner.key,
        name: "testPolicy",
        objects: [{ type: "channel", key: "test" }],
        actions: ["create"],
      });

      const { result } = await renderHookSuspended(
        () => ({
          retrieve: Policy.use({ key: policy.key }),
          rename: Policy.useRename(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.rename.update({ key: policy.key, name: "newName" });
      });
      await waitFor(() => expect(result.current.retrieve?.name).toEqual("newName"));
    });
  });

  describe("useDelete", () => {
    it("should correctly delete a policy", async () => {
      const policy = await client.access.policies.create({
        role: owner.key,
        name: "testPolicy",
        objects: [{ type: "channel", key: "test" }],
        actions: ["create"],
      });

      const { result } = renderHook(() => Policy.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(policy.key);
      });
      await waitFor(async () => {
        await expect(client.access.policies.retrieve(policy.key)).rejects.toThrow(
          NotFoundError,
        );
      });
    });

    it("should delete multiple policies", async () => {
      const policy1 = await client.access.policies.create({
        role: owner.key,
        name: "deletePolicy1",
        objects: [{ type: "channel", key: "1" }],
        actions: ["create"],
      });
      const policy2 = await client.access.policies.create({
        role: owner.key,
        name: "deletePolicy2",
        objects: [{ type: "channel", key: "2" }],
        actions: ["delete"],
      });

      const { result } = renderHook(() => Policy.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync([policy1.key, policy2.key]);
      });
      await waitFor(async () => {
        await expect(client.access.policies.retrieve(policy1.key)).rejects.toThrow(
          NotFoundError,
        );
        await expect(client.access.policies.retrieve(policy2.key)).rejects.toThrow(
          NotFoundError,
        );
      });
    });
  });

  describe("useForm", () => {
    it("should create a new policy", async () => {
      const { result } = renderHook(() => Policy.useForm({ query: null }), { wrapper });

      await act(async () => {
        result.current.form.set(path.name, "formPolicy");
        result.current.form.set(path.role, owner.key);
        result.current.form.set(path.objects, [{ type: "channel", key: "form" }]);
        result.current.form.set(path.actions, ["create"]);
      });

      act(() => {
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.form.get(path.key).value).toBeDefined();
      });

      const createdKey = result.current.form.get(path.key).value;
      assert(createdKey != null);
      const retrieved = await client.access.policies.retrieve(createdKey);
      expect(retrieved.name).toEqual("formPolicy");
    });

    it("should retrieve and populate form with existing policy", async () => {
      const existingPolicy = await client.access.policies.create({
        role: owner.key,
        name: "existingPolicy",
        objects: [{ type: "user", key: "user1" }],
        actions: ["delete", "update"],
      });

      const { result } = await renderHookSuspended(
        () => Policy.useForm({ query: { key: existingPolicy.key } }),
        {
          wrapper,
        },
      );

      await waitFor(() => {
        expect(result.current.form.get(path.name).value).toEqual("existingPolicy");
        expect(result.current.form.get(path.actions).value).toContain("delete");
      });
    });

    it("should save edits to an existing policy", async () => {
      const policy = await client.access.policies.create({
        role: owner.key,
        name: "editedPolicy",
        objects: [{ type: "channel", key: "edited" }],
        actions: ["create"],
      });
      const { result } = await renderHookSuspended(
        () => Policy.useForm({ query: { key: policy.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      act(() => {
        result.current.form.set(path.name, "renamedPolicy");
        result.current.form.set(path.actions, ["create", "delete"]);
      });
      await act(async () => {
        await result.current.saveAsync();
      });
      expect(result.current.form.value().key).toEqual(policy.key);
      await waitFor(async () => {
        const stored = await client.access.policies.retrieve(policy.key);
        expect(stored).toMatchObject({
          name: "renamedPolicy",
          objects: [{ type: "channel", key: "edited" }],
          actions: ["create", "delete"],
        });
      });
    });
  });
});
