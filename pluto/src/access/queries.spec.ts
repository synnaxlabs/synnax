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
  type ontology,
  project,
  query,
  ranger,
  type Synnax,
  user,
} from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { afterAll, assert, beforeEach, describe, expect, it } from "vitest";

import { Access } from "@/access";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const subjectOf = (c: Synnax): ontology.ID => {
  const u = c.auth?.user;
  assert(u != null, "client has no authenticated user");
  return user.ontologyID(u.key);
};

const cachedPoliciesOf = (c: Synnax): access.policy.Policy[] => {
  const cached = c.access.policies.getCached({ for: subjectOf(c) });
  return query.isLive(cached) ? cached : [];
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

  const baseObjects: ontology.ID[] = [
    channel.TYPE_ONTOLOGY_ID,
    framer.TYPE_ONTOLOGY_ID,
    user.TYPE_ONTOLOGY_ID,
    access.role.TYPE_ONTOLOGY_ID,
    access.policy.TYPE_ONTOLOGY_ID,
  ];

  describe("useGranted", () => {
    it("should return true when the user has the appropriate permissions", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.TYPE_ONTOLOGY_ID, action: "retrieve" }),
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
          Access.useGranted({ objects: ranger.TYPE_ONTOLOGY_ID, action: "retrieve" }),
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
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.TYPE_ONTOLOGY_ID, action: "retrieve" }),
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
      const u = await client.users.create({
        username: id.create(),
        password: "test",
        firstName: "test",
        lastName: "test",
      });
      const p = await client.access.policies.create({
        name: id.create(),
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const r = await client.access.roles.create({
        name: id.create(),
        description: "test",
      });
      await client.ontology.addChildren(
        access.role.ontologyID(r.key),
        access.policy.ontologyID(p.key),
      );
      await client.access.roles.assign({ user: u.key, role: r.key });
      const subject = user.ontologyID(u.key);
      const { result } = renderHook(
        () =>
          Access.useGranted({
            subject,
            objects: ranger.TYPE_ONTOLOGY_ID,
            action: "retrieve",
          }),
        { wrapper: await createAsyncSynnaxWrapper({ client }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
      await client.access.policies.delete(p.key);
      await waitFor(() => {
        expect(result.current).toBe(false);
      });
    });

    it("should ignore relationship changes that cannot affect permissions", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_ONTOLOGY_ID, project.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve", "create"],
      });
      let renders = 0;
      const { result } = renderHook(
        () => {
          renders++;
          return Access.useGranted({
            objects: ranger.TYPE_ONTOLOGY_ID,
            action: "retrieve",
          });
        },
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
      const rendersWhenSettled = renders;
      // A project's group -> project link is not a role link, so the gate must
      // drop it: no re-evaluation, no re-render.
      const proj = await userClient.projects.create({
        name: id.create(),
        layout: {},
      });
      // Wait until the link reaches the cache (event delivered)...
      await waitFor(() => {
        const rels = userClient.ontology.cache.relationships.get(
          (rel) => rel.to.type === "project" && rel.to.key === proj.key,
        );
        expect(rels.length).toBeGreaterThan(0);
      });
      // ...then confirm it caused no re-evaluation.
      expect(renders).toBe(rendersWhenSettled);
      expect(result.current).toBe(true);
    });

    it("should handle multiple objects correctly", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_ONTOLOGY_ID, channel.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () =>
          Access.useGranted({
            objects: [ranger.TYPE_ONTOLOGY_ID, channel.TYPE_ONTOLOGY_ID],
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
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.TYPE_ONTOLOGY_ID, action: "retrieve" }),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const result = Access.isGranted({
        client: userClient,
        query: { objects: ranger.TYPE_ONTOLOGY_ID, action: "retrieve" },
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
          Access.useGranted({ objects: ranger.TYPE_ONTOLOGY_ID, action: "retrieve" }),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(false);
      });
      const result = Access.isGranted({
        client: userClient,
        query: { objects: ranger.TYPE_ONTOLOGY_ID, action: "retrieve" },
      });
      expect(result).toBe(false);
    });

    it("should return false when client is null", () => {
      const result = Access.isGranted({
        client: null,
        query: { objects: ranger.TYPE_ONTOLOGY_ID, action: "retrieve" },
      });
      expect(result).toBe(false);
    });

    it("should use cached policies", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve", "create"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.TYPE_ONTOLOGY_ID, action: "retrieve" }),
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
        query: { objects: ranger.TYPE_ONTOLOGY_ID, action: "retrieve" },
      });
      expect(resultRetrieve).toBe(true);
      const resultCreate = Access.isGranted({
        client: userClient,
        query: { objects: ranger.TYPE_ONTOLOGY_ID, action: "create" },
      });
      expect(resultCreate).toBe(true);
      const resultDelete = Access.isGranted({
        client: userClient,
        query: { objects: ranger.TYPE_ONTOLOGY_ID, action: "delete" },
      });
      expect(resultDelete).toBe(false);
    });
  });

  describe("useViewGranted", () => {
    it("should return true when user has retrieve permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () => Access.useRetrieveGranted(ranger.TYPE_ONTOLOGY_ID),
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
      const { result } = renderHook(
        () => Access.useRetrieveGranted(ranger.TYPE_ONTOLOGY_ID),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(false);
      });
    });

    it("should handle multiple ontology IDs", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_ONTOLOGY_ID, channel.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () =>
          Access.useRetrieveGranted([
            ranger.TYPE_ONTOLOGY_ID,
            channel.TYPE_ONTOLOGY_ID,
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
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve", "update", "create"],
      });
      const { result } = renderHook(
        () => Access.useUpdateGranted(ranger.TYPE_ONTOLOGY_ID),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when user only has retrieve permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () => Access.useUpdateGranted(ranger.TYPE_ONTOLOGY_ID),
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
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve", "update", "delete"],
      });
      const { result } = renderHook(
        () => Access.useDeleteGranted(ranger.TYPE_ONTOLOGY_ID),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when user lacks delete permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve", "update"],
      });
      const { result } = renderHook(
        () => Access.useDeleteGranted(ranger.TYPE_ONTOLOGY_ID),
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
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve", "create"],
      });
      const { result } = renderHook(
        () => Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when user lacks create permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () => Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID),
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
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () => Access.useRetrieveGranted(ranger.TYPE_ONTOLOGY_ID),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const result = Access.viewGranted({
        client: userClient,
        id: ranger.TYPE_ONTOLOGY_ID,
      });
      expect(result).toBe(true);
    });
  });

  describe("updateGranted", () => {
    it("should return true when the user has retrieve and update permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve", "update", "create"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () => Access.useUpdateGranted(ranger.TYPE_ONTOLOGY_ID),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const result = Access.updateGranted({
        client: userClient,
        id: ranger.TYPE_ONTOLOGY_ID,
      });
      expect(result).toBe(true);
    });
  });

  describe("deleteGranted", () => {
    it("should return true when the user has all required permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve", "update", "delete"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () => Access.useDeleteGranted(ranger.TYPE_ONTOLOGY_ID),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const result = Access.deleteGranted({
        client: userClient,
        id: ranger.TYPE_ONTOLOGY_ID,
      });
      expect(result).toBe(true);
    });
  });

  describe("createGranted", () => {
    it("should return true when the user has retrieve and create permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve", "create"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () => Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const result = Access.createGranted({
        client: userClient,
        id: ranger.TYPE_ONTOLOGY_ID,
      });
      expect(result).toBe(true);
    });
  });

  describe("useLoadPermissions", () => {
    it("should load all policies for the current user", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result } = renderHook(() => Access.useLoadPermissions({}), { wrapper });
      await waitFor(() => {
        expect(result.current.data).toBeDefined();
      });
      expect(result.current.data!.length).toBeGreaterThan(0);
      const policy = result.current.data!.find((p) => p.name === policyName);
      expect(policy).toBeDefined();
      expect(policy!.actions).toContain("retrieve");
    });

    it("should load policies for a specific subject", async () => {
      const policyName = id.create();
      const u = await client.users.create({
        username: id.create(),
        password: "test",
        firstName: "test",
        lastName: "test",
      });
      const p = await client.access.policies.create({
        name: policyName,
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const r = await client.access.roles.create({
        name: id.create(),
        description: "test",
      });
      await client.ontology.addChildren(
        access.role.ontologyID(r.key),
        access.policy.ontologyID(p.key),
      );
      await client.access.roles.assign({
        user: u.key,
        role: r.key,
      });
      const wrapper = await createAsyncSynnaxWrapper({ client });
      const { result } = renderHook(
        () => Access.useLoadPermissions({ subject: user.ontologyID(u.key) }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.data).toBeDefined();
      });
      expect(result.current.data!.length).toBeGreaterThan(0);
      const policy = result.current.data!.find((pol) => pol.name === policyName);
      expect(policy).toBeDefined();
      expect(policy!.actions).toContain("retrieve");
    });

    it("should return empty array when user has no policies", async () => {
      const u = await client.users.create({
        username: id.create(),
        password: "test",
        firstName: "test",
        lastName: "test",
      });
      const wrapper = await createAsyncSynnaxWrapper({ client });
      const { result } = renderHook(
        () => Access.useLoadPermissions({ subject: user.ontologyID(u.key) }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.data).toBeDefined();
      });
      expect(result.current.data!.length).toBe(0);
    });

    it("should cache loaded policies", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve", "create"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result } = renderHook(() => Access.useLoadPermissions({}), { wrapper });
      await waitFor(() => {
        expect(result.current.data).toBeDefined();
      });
      const policies = cachedPoliciesOf(userClient).filter(
        (p) => p.name === policyName,
      );
      expect(policies.length).toBe(1);
      expect(policies[0].name).toBe(policyName);
      expect(policies[0].actions).toContain("retrieve");
      expect(policies[0].actions).toContain("create");
    });

    it("should cache ontology relationships between roles and policies", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        objects: [ranger.TYPE_ONTOLOGY_ID, ...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result } = renderHook(() => Access.useLoadPermissions({}), { wrapper });
      await waitFor(() => {
        expect(result.current.data).toBeDefined();
      });
      const policy = result.current.data!.find((p) => p.name === policyName);
      expect(policy).toBeDefined();
      const policyID = access.policy.ontologyID(policy!.key);
      const relationships = userClient.ontology.cache.relationships.get(
        (r) => r.from.type === "role" && r.to.type === "policy",
      );
      expect(relationships.length).toBeGreaterThan(0);
      const roleToPolicyRel = relationships.find((r) => r.to.key === policy!.key);
      expect(roleToPolicyRel).toBeDefined();
      expect(roleToPolicyRel!.to).toEqual(policyID);
    });
  });
});
