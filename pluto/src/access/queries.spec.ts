// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  access,
  channel,
  framer,
  InvalidCredentialsError,
  query,
  ranger,
  type resource,
  type Synnax,
  user,
} from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { afterAll, assert, beforeEach, describe, expect, it, vi } from "vitest";

import { Access } from "@/access";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil/Synnax";

const cachedPoliciesOf = (c: Synnax): access.policy.Policy[] => {
  const u = c.users.current.watch().getCached();
  assert(query.isLive(u), "client has no signed-in user");
  return c.access.policies.watch({ roles: [u.role] }).getCached()?.records ?? [];
};

describe("Access Queries", () => {
  let controller: AbortController;
  const client = createTestClient();
  beforeEach(() => {
    controller = new AbortController();
  });
  afterAll(() => {
    controller.abort();
  });

  const baseObjects: resource.ID[] = [
    channel.TYPE_RESOURCE_ID,
    framer.TYPE_RESOURCE_ID,
    user.TYPE_RESOURCE_ID,
    access.role.TYPE_RESOURCE_ID,
    access.policy.TYPE_RESOURCE_ID,
  ];

  describe("useGranted", () => {
    it("should return true when the user has the appropriate permissions", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.TYPE_RESOURCE_ID, action: "retrieve" }),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when the user does not have the appropriate permissions", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.TYPE_RESOURCE_ID, action: "retrieve" }),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(false);
      });
    });

    it("should correctly cache the users permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.TYPE_RESOURCE_ID, action: "retrieve" }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
      expect(result.current).toBe(true);
      const policies = cachedPoliciesOf(userClient).filter(
        (p) => p.name === policyName,
      );
      expect(policies.length).toBe(1);
      expect(policies[0].name).toBe(policyName);
    });

    it("should re-evaluate the grant when a policy is removed, without a remount", async () => {
      const username = id.create();
      const r = await client.access.roles.create({
        name: id.create(),
        description: "test",
      });
      const p = await client.access.policies.create({
        role: r.key,
        name: id.create(),
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      await client.users.create({
        username,
        password: "test",
        name: "test test",
        role: r.key,
      });
      const userClient = createTestClient({ username, password: "test" });
      const { result } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.TYPE_RESOURCE_ID, action: "retrieve" }),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
      await client.access.policies.delete(p.key);
      await waitFor(() => {
        expect(result.current).toBe(false);
      });
    });

    describe("before the client knows its user", () => {
      let username: string;
      let role: access.role.Role;
      let rangePolicy: access.policy.Policy;
      beforeEach(async () => {
        username = id.create();
        role = await client.access.roles.create({
          name: id.create(),
          description: "test",
        });
        await client.access.policies.create({
          role: role.key,
          name: id.create(),
          objects: baseObjects,
          actions: ["retrieve"],
        });
        rangePolicy = await client.access.policies.create({
          role: role.key,
          name: id.create(),
          objects: [ranger.TYPE_RESOURCE_ID],
          actions: ["retrieve"],
        });
        await client.users.create({
          username,
          password: "test",
          name: "test test",
          role: role.key,
        });
      });

      const renderGranted = (userClient: Synnax) =>
        renderHook(
          () =>
            Access.useGranted({ objects: ranger.TYPE_RESOURCE_ID, action: "retrieve" }),
          { wrapper: createSynnaxWrapper({ client: userClient }) },
        );

      it("should re-evaluate the grant when mounted before the first login", async () => {
        const userClient = createTestClient({ username, password: "test" });
        expect(userClient.users.current.watch().getCached()).toBeUndefined();
        const { result } = renderGranted(userClient);
        await waitFor(() => {
          expect(result.current).toBe(true);
        });
        await client.access.policies.delete(rangePolicy.key);
        await waitFor(() => {
          expect(result.current).toBe(false);
        });
      });

      it("should re-evaluate the grant when mounted during a login with new credentials", async () => {
        const userClient = createTestClient({ username, password: "test" });
        await userClient.connect();
        userClient.reauthenticate({ username, password: "test" });
        expect(userClient.users.current.watch().getCached()).toBeUndefined();
        const { result } = renderGranted(userClient);
        await waitFor(() => {
          expect(result.current).toBe(true);
        });
        await client.access.policies.delete(rangePolicy.key);
        await waitFor(() => {
          expect(result.current).toBe(false);
        });
      });

      it("should re-evaluate the grant after a failed login", async () => {
        await client.access.policies.delete(rangePolicy.key);
        const userClient = createTestClient({ username, password: "wrong" });
        const { result } = renderGranted(userClient);
        await expect(userClient.connect()).rejects.toThrow(InvalidCredentialsError);
        userClient.reauthenticate({ username, password: "test" });
        await userClient.connect();
        await client.access.policies.create({
          role: role.key,
          name: id.create(),
          objects: [ranger.TYPE_RESOURCE_ID],
          actions: ["retrieve"],
        });
        await waitFor(() => {
          expect(result.current).toBe(true);
        });
      });
    });

    it("should pass an identity-stable query to the client on every render", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const spy = vi.spyOn(userClient.access.granted, "watch");
      const { result, rerender } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.TYPE_RESOURCE_ID, action: "retrieve" }),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
      rerender();
      rerender();
      // One query object across every read is what keys the client's verdict
      // memo; a fresh object per render would evaluate the policies each time.
      const queries = new Set(spy.mock.calls.map((call) => call[1]));
      expect(queries.size).toBe(1);
      spy.mockRestore();
    });

    it("should handle multiple objects correctly", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_RESOURCE_ID, channel.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () =>
          Access.useGranted({
            objects: [ranger.TYPE_RESOURCE_ID, channel.TYPE_RESOURCE_ID],
            action: "retrieve",
          }),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });
  });

  describe("isGranted", () => {
    it("should return true when the user has the appropriate permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.TYPE_RESOURCE_ID, action: "retrieve" }),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const result = Access.isGranted({
        client: userClient,
        query: { objects: ranger.TYPE_RESOURCE_ID, action: "retrieve" },
      });
      expect(result).toBe(true);
    });

    it("should return false when the user does not have the appropriate permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.TYPE_RESOURCE_ID, action: "retrieve" }),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(false);
      });
      const result = Access.isGranted({
        client: userClient,
        query: { objects: ranger.TYPE_RESOURCE_ID, action: "retrieve" },
      });
      expect(result).toBe(false);
    });

    it("should return false when client is null", () => {
      const result = Access.isGranted({
        client: null,
        query: { objects: ranger.TYPE_RESOURCE_ID, action: "retrieve" },
      });
      expect(result).toBe(false);
    });

    it("should use cached policies", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve", "create"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.TYPE_RESOURCE_ID, action: "retrieve" }),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const policies = cachedPoliciesOf(userClient).filter(
        (p) => p.name === policyName,
      );
      expect(policies.length).toBe(1);
      const resultRetrieve = Access.isGranted({
        client: userClient,
        query: { objects: ranger.TYPE_RESOURCE_ID, action: "retrieve" },
      });
      expect(resultRetrieve).toBe(true);
      const resultCreate = Access.isGranted({
        client: userClient,
        query: { objects: ranger.TYPE_RESOURCE_ID, action: "create" },
      });
      expect(resultCreate).toBe(true);
      const resultDelete = Access.isGranted({
        client: userClient,
        query: { objects: ranger.TYPE_RESOURCE_ID, action: "delete" },
      });
      expect(resultDelete).toBe(false);
    });
  });

  describe("useViewGranted", () => {
    it("should return true when user has retrieve permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = await renderHookSuspended(
        () => Access.useRetrieveGranted(ranger.TYPE_RESOURCE_ID),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when user lacks retrieve permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = await renderHookSuspended(
        () => Access.useRetrieveGranted(ranger.TYPE_RESOURCE_ID),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(false);
      });
    });

    it("should handle multiple resource IDs", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_RESOURCE_ID, channel.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = await renderHookSuspended(
        () =>
          Access.useRetrieveGranted([
            ranger.TYPE_RESOURCE_ID,
            channel.TYPE_RESOURCE_ID,
          ]),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });
  });

  describe("useUpdateGranted", () => {
    it("should return true when user has retrieve and update permissions", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve", "update", "create"],
      });
      const { result } = renderHook(
        () => Access.useUpdateGranted(ranger.TYPE_RESOURCE_ID),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when user only has retrieve permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () => Access.useUpdateGranted(ranger.TYPE_RESOURCE_ID),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(false);
      });
    });
  });

  describe("useDeleteGranted", () => {
    it("should return true when user has all required permissions", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve", "update", "delete"],
      });
      const { result } = renderHook(
        () => Access.useDeleteGranted(ranger.TYPE_RESOURCE_ID),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when user lacks delete permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve", "update"],
      });
      const { result } = renderHook(
        () => Access.useDeleteGranted(ranger.TYPE_RESOURCE_ID),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(false);
      });
    });
  });

  describe("useCreateGranted", () => {
    it("should return true when user has retrieve and create permissions", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve", "create"],
      });
      const { result } = renderHook(
        () => Access.useCreateGranted(ranger.TYPE_RESOURCE_ID),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when user lacks create permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () => Access.useCreateGranted(ranger.TYPE_RESOURCE_ID),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(false);
      });
    });
  });

  describe("viewGranted", () => {
    it("should return true when the user has retrieve permission", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = await renderHookSuspended(
        () => Access.useRetrieveGranted(ranger.TYPE_RESOURCE_ID),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const result = Access.viewGranted({
        client: userClient,
        id: ranger.TYPE_RESOURCE_ID,
      });
      expect(result).toBe(true);
    });
  });

  describe("updateGranted", () => {
    it("should return true when the user has retrieve and update permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve", "update", "create"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () => Access.useUpdateGranted(ranger.TYPE_RESOURCE_ID),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const result = Access.updateGranted({
        client: userClient,
        id: ranger.TYPE_RESOURCE_ID,
      });
      expect(result).toBe(true);
    });
  });

  describe("deleteGranted", () => {
    it("should return true when the user has all required permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve", "update", "delete"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () => Access.useDeleteGranted(ranger.TYPE_RESOURCE_ID),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const result = Access.deleteGranted({
        client: userClient,
        id: ranger.TYPE_RESOURCE_ID,
      });
      expect(result).toBe(true);
    });
  });

  describe("createGranted", () => {
    it("should return true when the user has retrieve and create permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve", "create"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () => Access.useCreateGranted(ranger.TYPE_RESOURCE_ID),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const result = Access.createGranted({
        client: userClient,
        id: ranger.TYPE_RESOURCE_ID,
      });
      expect(result).toBe(true);
    });
  });

  describe("useEnsurePermissions", () => {
    it("should grant a read on its first committed render", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = await renderHookSuspended(
        () => {
          Access.useEnsurePermissions({});
          return Access.useRetrieveGranted(ranger.TYPE_RESOURCE_ID);
        },
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      expect(result.current).toBe(true);
    });

    it("should cache every policy the current user holds", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_RESOURCE_ID, ...baseObjects],
        actions: ["retrieve", "create"],
      });
      await renderHookSuspended(() => Access.useEnsurePermissions({}), {
        wrapper: await createAsyncSynnaxWrapper({ client: userClient }),
      });
      const policies = cachedPoliciesOf(userClient).filter(
        (p) => p.name === policyName,
      );
      expect(policies.length).toBe(1);
      expect(policies[0].actions).toContain("retrieve");
      expect(policies[0].actions).toContain("create");
    });
  });
});
