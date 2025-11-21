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

import { Policy } from "@/access/policy";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useList", () => {
    it("should return a list of policy keys", async () => {
      const policy1 = await client.access.policies.create({
        name: "policy1",
        effect: "allow",
        objects: [{ type: "channel", key: "1" }],
        actions: ["create"],
      });
      const policy2 = await client.access.policies.create({
        name: "policy2",
        effect: "deny",
        objects: [{ type: "channel", key: "2" }],
        actions: ["delete"],
      });

      const { result } = renderHook(() => Policy.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(policy1.key);
      expect(result.current.data).toContain(policy2.key);
    });

    it("should get individual policies using getItem", async () => {
      const testPolicy = await client.access.policies.create({
        name: "testPolicy",
        effect: "allow",
        objects: [{ type: "channel", key: "test" }],
        actions: ["create", "retrieve"],
      });

      const { result } = renderHook(() => Policy.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedPolicy = result.current.getItem(testPolicy.key);
      expect(retrievedPolicy?.key).toEqual(testPolicy.key);
      expect(retrievedPolicy?.name).toEqual("testPolicy");
      expect(retrievedPolicy?.effect).toEqual("allow");
    });

    it("should update the list when a policy is created", async () => {
      const { result } = renderHook(() => Policy.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const initialLength = result.current.data.length;

      const newPolicy = await client.access.policies.create({
        name: "newPolicy",
        effect: "allow",
        objects: [{ type: "channel", key: "new" }],
        actions: ["*"],
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(initialLength + 1);
        expect(result.current.data).toContain(newPolicy.key);
      });
    });

    it("should remove policy from list when deleted", async () => {
      const testPolicy = await client.access.policies.create({
        name: "toDeletePolicy",
        effect: "deny",
        objects: [{ type: "channel", key: "delete" }],
        actions: ["delete"],
      });

      const { result } = renderHook(() => Policy.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(testPolicy.key);

      await client.access.policies.delete(testPolicy.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(testPolicy.key);
      });
    });

    it("should handle multiple policy updates simultaneously", async () => {
      const policy1 = await client.access.policies.create({
        name: "multiUpdate1",
        effect: "allow",
        objects: [{ type: "channel", key: "1" }],
        actions: ["create"],
      });
      const policy2 = await client.access.policies.create({
        name: "multiUpdate2",
        effect: "allow",
        objects: [{ type: "channel", key: "2" }],
        actions: ["create"],
      });

      const { result } = renderHook(() => Policy.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      await Promise.all([
        client.access.policies.create({ ...policy1, name: "updated1" }),
        client.access.policies.create({ ...policy2, name: "updated2" }),
      ]);

      await waitFor(() => {
        expect(result.current.getItem(policy1.key)?.name).toEqual("updated1");
        expect(result.current.getItem(policy2.key)?.name).toEqual("updated2");
      });
    });
  });

  describe("useRetrieve", () => {
    it("should retrieve a single policy by key", async () => {
      const testPolicy = await client.access.policies.create({
        name: "singlePolicy",
        effect: "allow",
        objects: [{ type: "channel", key: "single" }],
        actions: ["create", "retrieve", "update"],
      });

      const { result } = renderHook(() => Policy.useRetrieve({ key: testPolicy.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data?.key).toEqual(testPolicy.key);
      expect(result.current.data?.name).toEqual("singlePolicy");
      expect(result.current.data?.effect).toEqual("allow");
      expect(result.current.data?.actions).toContain("create");
    });

    it("should handle retrieve with valid policy key", async () => {
      const policy = await client.access.policies.create({
        name: "validPolicy",
        effect: "deny",
        objects: [{ type: "user", key: "user1" }],
        actions: ["delete"],
      });

      const { result } = renderHook(() => Policy.useRetrieve({ key: policy.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data).toBeDefined();
      expect(result.current.data?.key).toEqual(policy.key);
      expect(result.current.data?.effect).toEqual("deny");
    });
  });

  describe("useRename", () => {
    it("should correctly rename a policy", async () => {
      const policy = await client.access.policies.create({
        name: "testPolicy",
        effect: "allow",
        objects: [{ type: "channel", key: "test" }],
        actions: ["create"],
      });

      const { result } = renderHook(
        () => ({
          retrieve: Policy.useRetrieve({ key: policy.key }),
          rename: Policy.useRename(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.rename.update({ key: policy.key, name: "newName" });
      });
      await waitFor(() =>
        expect(result.current.retrieve.data?.name).toEqual("newName"),
      );
    });
  });

  describe("useDelete", () => {
    it("should correctly delete a policy", async () => {
      const policy = await client.access.policies.create({
        name: "testPolicy",
        effect: "allow",
        objects: [{ type: "channel", key: "test" }],
        actions: ["create"],
      });

      const { result } = renderHook(() => Policy.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(policy.key);
      });
      await waitFor(async () => {
        await expect(
          client.access.policies.retrieve({ key: policy.key }),
        ).rejects.toThrow(NotFoundError);
      });
    });

    it("should delete multiple policies", async () => {
      const policy1 = await client.access.policies.create({
        name: "deletePolicy1",
        effect: "allow",
        objects: [{ type: "channel", key: "1" }],
        actions: ["create"],
      });
      const policy2 = await client.access.policies.create({
        name: "deletePolicy2",
        effect: "deny",
        objects: [{ type: "channel", key: "2" }],
        actions: ["delete"],
      });

      const { result } = renderHook(() => Policy.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync([policy1.key, policy2.key]);
      });
      await waitFor(async () => {
        await expect(
          client.access.policies.retrieve({ key: policy1.key }),
        ).rejects.toThrow(NotFoundError);
        await expect(
          client.access.policies.retrieve({ key: policy2.key }),
        ).rejects.toThrow(NotFoundError);
      });
    });
  });

  describe("useForm", () => {
    it("should create a new policy", async () => {
      const { result } = renderHook(() => Policy.useForm({ query: {} }), { wrapper });

      await act(async () => {
        result.current.form.set("name", "formPolicy");
        result.current.form.set("effect", "allow");
        result.current.form.set("objects", [{ type: "channel", key: "form" }]);
        result.current.form.set("actions", ["create"]);
      });

      act(() => {
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.form.get("key")).toBeDefined();
      });

      const createdKey = result.current.form.get<string>("key").value;
      const retrieved = await client.access.policies.retrieve({ key: createdKey });
      expect(retrieved.name).toEqual("formPolicy");
      expect(retrieved.effect).toEqual("allow");
    });

    it("should retrieve and populate form with existing policy", async () => {
      const existingPolicy = await client.access.policies.create({
        name: "existingPolicy",
        effect: "deny",
        objects: [{ type: "user", key: "user1" }],
        actions: ["delete", "update"],
      });

      const { result } = renderHook(
        () => Policy.useForm({ query: { key: existingPolicy.key } }),
        {
          wrapper,
        },
      );

      await waitFor(() => {
        expect(result.current.form.get("name").value).toEqual("existingPolicy");
        expect(result.current.form.get("effect").value).toEqual("deny");
        expect(result.current.form.get("actions").value).toContain("delete");
      });
    });
  });
});
