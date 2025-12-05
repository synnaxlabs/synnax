// Copyright 2025 Synnax Labs, Inc.
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
  createTestClient,
  createTestClientWithPolicy,
  type ontology,
  ranger,
  user,
} from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { afterAll, beforeEach, describe, expect, it } from "vitest";

import { Access } from "@/access";
import { Flux } from "@/flux";
import { type Pluto } from "@/pluto";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

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
    channel.ontologyID(0),
    { type: "framer", key: "" },
    user.ontologyID(""),
    access.role.ontologyID(""),
    access.policy.ontologyID(""),
  ];

  describe("useGranted", () => {
    it("should return true when the user has the appropriate permissions", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.ontologyID(""), actions: "retrieve" }),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when the user does not have the appropriate permissions", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        effect: "allow",
        objects: [...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.ontologyID(""), actions: "retrieve" }),
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
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.ontologyID(""), actions: "retrieve" }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
      expect(result.current).toBe(true);
      const { result: storeResult } = renderHook(
        () => Flux.useStore<Pluto.FluxStore>(),
        { wrapper },
      );
      const policies = storeResult.current.policies.get((p) => p.name === policyName);
      expect(policies.length).toBe(1);
      expect(policies[0].name).toBe(policyName);
    });

    it("should handle multiple objects correctly", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        effect: "allow",
        objects: [ranger.ontologyID(""), channel.ontologyID(0), ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () =>
          Access.useGranted({
            objects: [ranger.ontologyID(""), channel.ontologyID(0)],
            actions: "retrieve",
          }),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should handle multiple actions correctly", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve", "create", "update"],
      });
      const { result } = renderHook(
        () =>
          Access.useGranted({
            objects: ranger.ontologyID(""),
            actions: ["retrieve", "create"],
          }),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when missing partial permissions", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () =>
          Access.useGranted({
            objects: ranger.ontologyID(""),
            actions: ["retrieve", "delete"],
          }),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(false);
      });
    });
  });

  describe("isGranted", () => {
    it("should return true when the user has the appropriate permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.ontologyID(""), actions: "retrieve" }),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const { result: storeResult } = renderHook(
        () => Flux.useStore<Pluto.FluxStore>(),
        { wrapper },
      );
      const result = Access.isGranted({
        store: storeResult.current,
        client: userClient,
        query: { objects: ranger.ontologyID(""), actions: "retrieve" },
      });
      expect(result).toBe(true);
    });

    it("should return false when the user does not have the appropriate permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        effect: "allow",
        objects: [...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.ontologyID(""), actions: "retrieve" }),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(false);
      });
      const { result: storeResult } = renderHook(
        () => Flux.useStore<Pluto.FluxStore>(),
        { wrapper },
      );
      const result = Access.isGranted({
        store: storeResult.current,
        client: userClient,
        query: { objects: ranger.ontologyID(""), actions: "retrieve" },
      });
      expect(result).toBe(false);
    });

    it("should return false when client is null", async () => {
      const wrapper = await createAsyncSynnaxWrapper({ client });
      const { result: storeResult } = renderHook(
        () => Flux.useStore<Pluto.FluxStore>(),
        { wrapper },
      );
      const result = Access.isGranted({
        store: storeResult.current,
        client: null,
        query: { objects: ranger.ontologyID(""), actions: "retrieve" },
      });
      expect(result).toBe(false);
    });

    it("should use cached policies from the store", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve", "create"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () =>
          Access.useGranted({ objects: ranger.ontologyID(""), actions: "retrieve" }),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const { result: storeResult } = renderHook(
        () => Flux.useStore<Pluto.FluxStore>(),
        { wrapper },
      );
      const policies = storeResult.current.policies.get((p) => p.name === policyName);
      expect(policies.length).toBe(1);
      const resultRetrieve = Access.isGranted({
        store: storeResult.current,
        client: userClient,
        query: { objects: ranger.ontologyID(""), actions: "retrieve" },
      });
      expect(resultRetrieve).toBe(true);
      const resultCreate = Access.isGranted({
        store: storeResult.current,
        client: userClient,
        query: { objects: ranger.ontologyID(""), actions: "create" },
      });
      expect(resultCreate).toBe(true);
      const resultDelete = Access.isGranted({
        store: storeResult.current,
        client: userClient,
        query: { objects: ranger.ontologyID(""), actions: "delete" },
      });
      expect(resultDelete).toBe(false);
    });
  });

  describe("useViewGranted", () => {
    it("should return true when user has retrieve permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () => Access.useViewGranted(ranger.ontologyID("")),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when user lacks retrieve permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        effect: "allow",
        objects: [...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () => Access.useViewGranted(ranger.ontologyID("")),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(false);
      });
    });

    it("should handle multiple ontology IDs", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        effect: "allow",
        objects: [ranger.ontologyID(""), channel.ontologyID(0), ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () => Access.useViewGranted([ranger.ontologyID(""), channel.ontologyID(0)]),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });
  });

  describe("useEditGranted", () => {
    it("should return true when user has retrieve and update permissions", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve", "update", "create"],
      });
      const { result } = renderHook(
        () => Access.useEditGranted(ranger.ontologyID("")),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when user only has retrieve permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () => Access.useEditGranted(ranger.ontologyID("")),
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
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve", "update", "delete"],
      });
      const { result } = renderHook(
        () => Access.useDeleteGranted(ranger.ontologyID("")),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when user lacks delete permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve", "update"],
      });
      const { result } = renderHook(
        () => Access.useDeleteGranted(ranger.ontologyID("")),
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
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve", "create"],
      });
      const { result } = renderHook(
        () => Access.useCreateGranted(ranger.ontologyID("")),
        { wrapper: await createAsyncSynnaxWrapper({ client: userClient }) },
      );
      await waitFor(() => {
        expect(result.current).toBe(true);
      });
    });

    it("should return false when user lacks create permission", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: id.create(),
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve"],
      });
      const { result } = renderHook(
        () => Access.useCreateGranted(ranger.ontologyID("")),
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
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () => Access.useViewGranted(ranger.ontologyID("")),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const { result: storeResult } = renderHook(
        () => Flux.useStore<Pluto.FluxStore>(),
        { wrapper },
      );
      const result = Access.viewGranted({
        store: storeResult.current,
        client: userClient,
        id: ranger.ontologyID(""),
      });
      expect(result).toBe(true);
    });
  });

  describe("editGranted", () => {
    it("should return true when the user has retrieve and update permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve", "update", "create"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () => Access.useEditGranted(ranger.ontologyID("")),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const { result: storeResult } = renderHook(
        () => Flux.useStore<Pluto.FluxStore>(),
        { wrapper },
      );
      const result = Access.editGranted({
        store: storeResult.current,
        client: userClient,
        id: ranger.ontologyID(""),
      });
      expect(result).toBe(true);
    });
  });

  describe("deleteGranted", () => {
    it("should return true when the user has all required permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve", "update", "delete"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () => Access.useDeleteGranted(ranger.ontologyID("")),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const { result: storeResult } = renderHook(
        () => Flux.useStore<Pluto.FluxStore>(),
        { wrapper },
      );
      const result = Access.deleteGranted({
        store: storeResult.current,
        client: userClient,
        id: ranger.ontologyID(""),
      });
      expect(result).toBe(true);
    });
  });

  describe("createGranted", () => {
    it("should return true when the user has retrieve and create permissions", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve", "create"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result: grantedResult } = renderHook(
        () => Access.useCreateGranted(ranger.ontologyID("")),
        { wrapper },
      );
      await waitFor(() => {
        expect(grantedResult.current).toBe(true);
      });
      const { result: storeResult } = renderHook(
        () => Flux.useStore<Pluto.FluxStore>(),
        { wrapper },
      );
      const result = Access.createGranted({
        store: storeResult.current,
        client: userClient,
        id: ranger.ontologyID(""),
      });
      expect(result).toBe(true);
    });
  });

  describe("useLoadPermissions", () => {
    it("should load all policies for the current user", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
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
      expect(policy!.effect).toBe("allow");
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
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
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
      expect(policy!.effect).toBe("allow");
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

    it("should cache loaded policies in the store", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve", "create"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result } = renderHook(() => Access.useLoadPermissions({}), { wrapper });
      await waitFor(() => {
        expect(result.current.data).toBeDefined();
      });
      const { result: storeResult } = renderHook(
        () => Flux.useStore<Pluto.FluxStore>(),
        { wrapper },
      );
      const policies = storeResult.current.policies.get((p) => p.name === policyName);
      expect(policies.length).toBe(1);
      expect(policies[0].name).toBe(policyName);
      expect(policies[0].effect).toBe("allow");
      expect(policies[0].actions).toContain("retrieve");
      expect(policies[0].actions).toContain("create");
    });

    it("should cache ontology relationships between roles and policies", async () => {
      const policyName = id.create();
      const userClient = await createTestClientWithPolicy(client, {
        name: policyName,
        effect: "allow",
        objects: [ranger.ontologyID(""), ...baseObjects],
        actions: ["retrieve"],
      });
      const wrapper = await createAsyncSynnaxWrapper({ client: userClient });
      const { result } = renderHook(() => Access.useLoadPermissions({}), { wrapper });
      await waitFor(() => {
        expect(result.current.data).toBeDefined();
      });
      const { result: storeResult } = renderHook(
        () => Flux.useStore<Pluto.FluxStore>(),
        { wrapper },
      );
      const policies = storeResult.current.policies.get((p) => p.name === policyName);
      expect(policies.length).toBe(1);
      const policyID = access.policy.ontologyID(policies[0].key);
      const relationships = storeResult.current.relationships.get(
        (r) => r.from.type === "role" && r.to.type === "policy",
      );
      expect(relationships.length).toBeGreaterThan(0);
      const roleToPolicyRel = relationships.find((r) => r.to.key === policies[0].key);
      expect(roleToPolicyRel).toBeDefined();
      expect(roleToPolicyRel!.to).toEqual(policyID);
    });
  });
});
