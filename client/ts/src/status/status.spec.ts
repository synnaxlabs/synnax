// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, id, TimeStamp, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import z from "zod";

import { NotFoundError } from "@/errors";
import { group } from "@/group";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { status } from "@/status";
import { createTestClient, expectLive, spyOnSend } from "@/testutil";

const client = createTestClient();

describe("Status", () => {
  describe("set", () => {
    it("should create a new status", async () => {
      const s = await client.statuses.set({
        name: "Test Status",
        key: "test-status-1",
        variant: "info",
        message: "This is a test status",
        time: TimeStamp.now(),
      });
      expect(s.key).toBe("test-status-1");
      expect(s.name).toBe("Test Status");
      expect(s.variant).toBe("info");
      expect(s.message).toBe("This is a test status");
    });

    it("should update an existing status", async () => {
      const key = "test-status-update";
      await client.statuses.set({
        name: "Original Status",
        key,
        variant: "info",
        message: "Original message",
        time: TimeStamp.now(),
      });

      const updated = await client.statuses.set({
        name: "Updated Status",
        key,
        variant: "warning",
        message: "Updated message",
        time: TimeStamp.now(),
      });

      expect(updated.key).toBe(key);
      expect(updated.name).toBe("Updated Status");
      expect(updated.variant).toBe("warning");
      expect(updated.message).toBe("Updated message");
    });

    it("should create multiple statuses at once", async () => {
      const statuses = await client.statuses.set([
        {
          name: "Status 1",
          key: "batch-1",
          variant: "success",
          message: "First batch status",
          time: TimeStamp.now(),
        },
        {
          name: "Status 2",
          key: "batch-2",
          variant: "error",
          message: "Second batch status",
          time: TimeStamp.now(),
        },
      ]);

      expect(statuses).toHaveLength(2);
      expect(statuses[0].key).toBe("batch-1");
      expect(statuses[1].key).toBe("batch-2");
    });

    it("should set a status with a parent", async () => {
      const parentGroup = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "Parent Group",
      });
      const parentOntologyID = group.ontologyID(parentGroup.key);
      const s = await client.statuses.set(
        {
          name: "Child Status",
          key: "child-status",
          variant: "info",
          message: "Status with parent",
          time: TimeStamp.now(),
        },
        { parent: parentOntologyID },
      );

      expect(s.key).toBe("child-status");

      const resources = await client.ontology.children.retrieve({
        ids: parentOntologyID,
      });

      const statusResource = resources.find((r) => r.id.key === "child-status");
      expect(statusResource).toBeDefined();
    });
  });

  describe("optimistic set", () => {
    const createCrude = (overrides: Partial<status.Crude> = {}): status.Crude => ({
      key: `optimistic-${id.create()}`,
      name: "Optimistic Status",
      variant: "info",
      message: "Optimistic message",
      time: TimeStamp.now(),
      ...overrides,
    });

    it("should cache the status before the write commits", async () => {
      const crude = createCrude();
      let duringWrite: query.Cached<status.Status> | undefined;
      await client.statuses.set(crude, {
        onOptimistic: () => {
          duringWrite = client.statuses.getCached(crude.key as status.Key);
        },
      });
      expect(expectLive(duringWrite).message).toEqual(crude.message);
    });

    it("should cache every status of a batch before the write commits", async () => {
      const first = createCrude();
      const second = createCrude();
      let duringWrite: Array<query.Cached<status.Status> | undefined> = [];
      await client.statuses.set([first, second], {
        onOptimistic: () => {
          duringWrite = [first, second].map(({ key }) =>
            client.statuses.getCached(key as status.Key),
          );
        },
      });
      expect(duringWrite.map((s) => expectLive(s).message)).toEqual([
        first.message,
        second.message,
      ]);
    });

    it("should stamp a key and a time onto a status that carries neither", async () => {
      const message = `unkeyed-${id.create()}`;
      const created = await client.statuses.set({ variant: "info", message });
      expect(created.key).not.toHaveLength(0);
      expect(created.time.valueOf()).toBeGreaterThan(0n);
      expect(expectLive(client.statuses.getCached(created.key)).message).toEqual(
        message,
      );
    });

    it("should keep the details of a schema-parametrized status", async () => {
      const detailsSchema = z.object({ count: z.number() });
      const crude = createCrude();
      let duringWrite: query.Cached<status.Status<typeof detailsSchema>> | undefined;
      await client.statuses.set<typeof detailsSchema>(
        { ...crude, details: { count: 5 } },
        {
          detailsSchema,
          onOptimistic: () => {
            duringWrite = client.statuses.getCached(
              crude.key as status.Key,
            ) as query.Cached<status.Status<typeof detailsSchema>>;
          },
        },
      );
      expect(expectLive(duringWrite).details).toEqual({ count: 5 });
    });

    it("should drop the optimistic status when the write fails", async () => {
      const crude = createCrude();
      await expect(
        client.statuses.set(crude, {
          onOptimistic: () => {
            throw new Error("write failed");
          },
        }),
      ).rejects.toThrow("write failed");
      expect(client.statuses.getCached(crude.key as status.Key)).toBeUndefined();
    });

    it("should leave no corpse behind when the write fails", async () => {
      const crude = createCrude();
      const key = crude.key as status.Key;
      await client.statuses.set(crude);
      // Another client owns the record, and this one neither streams nor subscribes, so
      // nothing puts the record in its cache before the rollback.
      const observer = createTestClient();
      await expect(
        observer.statuses.set(
          { ...crude, message: "replacement" },
          {
            onOptimistic: () => {
              throw new Error("write failed");
            },
          },
        ),
      ).rejects.toThrow("write failed");
      expect(observer.statuses.getCached(key)).toBeUndefined();
      expect((await observer.statuses.retrieve(key)).message).toEqual(crude.message);
    });

    it("should restore the previous status when the write fails", async () => {
      const crude = createCrude();
      const created = await client.statuses.set(crude);
      await expect(
        client.statuses.set(
          { ...crude, message: "replacement" },
          {
            onOptimistic: () => {
              throw new Error("write failed");
            },
          },
        ),
      ).rejects.toThrow("write failed");
      expect(expectLive(client.statuses.getCached(created.key)).message).toEqual(
        created.message,
      );
    });
  });

  describe("retrieve", () => {
    it("coalesces concurrent single retrieves into one request", async () => {
      const keys = [id.create(), id.create()];
      await Promise.all(
        keys.map(
          async (key) =>
            await client.statuses.set({
              key,
              name: "Coalesce Test",
              variant: "info",
              message: "coalesce",
              time: TimeStamp.now(),
            }),
        ),
      );
      const local = createTestClient();
      await local.connect();
      const send = spyOnSend(local);
      const res = await Promise.all(
        keys.map(async (key) => await local.statuses.retrieve(key)),
      );
      expect(res.map(({ key }) => key)).toEqual(keys);
      expect(
        send.mock.calls.filter(([target]) => target === "/status/retrieve"),
      ).toHaveLength(1);
    });

    it("does not reject concurrent retrieves when a key in the window is missing", async () => {
      const s = await client.statuses.set({
        key: id.create(),
        name: "Isolation Test",
        variant: "info",
        message: "isolation",
        time: TimeStamp.now(),
      });
      const local = createTestClient();
      await local.connect();
      const [ok, missing] = await Promise.allSettled([
        local.statuses.retrieve(s.key),
        local.statuses.retrieve(`missing-${id.create()}`),
      ]);
      expect(ok.status).toEqual("fulfilled");
      expect(missing.status).toEqual("rejected");
      if (missing.status === "rejected")
        expect(NotFoundError.matches(missing.reason)).toBe(true);
    });

    it("should retrieve a status by key", async () => {
      const created = await client.statuses.set({
        name: "Retrieve Test",
        key: "retrieve-test",
        variant: "loading",
        message: "Test retrieve",
        time: TimeStamp.now(),
      });

      const retrieved = await client.statuses.retrieve("retrieve-test");
      expect(retrieved.key).toBe(created.key);
      expect(retrieved.name).toBe(created.name);
      expect(retrieved.variant).toBe(created.variant);
    });

    it("should retrieve a single status when detailsSchema is explicitly undefined", async () => {
      const created = await client.statuses.set({
        name: "Undefined Schema Test",
        key: "undefined-schema-test",
        variant: "info",
        message: "Test undefined schema",
        time: TimeStamp.now(),
      });

      const retrieved = await client.statuses.retrieve({
        key: "undefined-schema-test",
        includeLabels: true,
        detailsSchema: undefined,
      });
      expect(Array.isArray(retrieved)).toBe(false);
      expect(retrieved.key).toBe(created.key);
      expect(retrieved.name).toBe(created.name);
    });

    it("should retrieve multiple statuses by keys", async () => {
      await client.statuses.set([
        {
          name: "Multi 1",
          key: "multi-1",
          variant: "info",
          message: "First",
          time: TimeStamp.now(),
        },
        {
          name: "Multi 2",
          key: "multi-2",
          variant: "warning",
          message: "Second",
          time: TimeStamp.now(),
        },
      ]);

      const statuses = await client.statuses.retrieve({
        keys: ["multi-1", "multi-2"],
      });

      expect(statuses).toHaveLength(2);
      const keys = statuses.map((s) => s.key);
      expect(keys).toContain("multi-1");
      expect(keys).toContain("multi-2");
    });

    it("should search for statuses", async () => {
      const uniqueName = `SearchableStatus_${Date.now()}`;
      await client.statuses.set({
        name: uniqueName,
        key: `searchable-${Date.now()}`,
        variant: "info",
        message: "Searchable status",
        time: TimeStamp.now(),
      });

      await expect
        .poll(async () =>
          (await client.statuses.retrieve({ searchTerm: uniqueName })).some(
            (s) => s.name === uniqueName,
          ),
        )
        .toBe(true);
    });

    it("should paginate results", async () => {
      const keys = [];
      for (let i = 0; i < 5; i++) {
        const key = `paginate-${i}-${Date.now()}`;
        keys.push(key);
        await client.statuses.set({
          name: `Paginate ${i}`,
          key,
          variant: "info",
          message: `Message ${i}`,
          time: TimeStamp.now(),
        });
      }

      const page1 = await client.statuses.retrieve({
        keys,
        limit: 2,
        offset: 0,
      });

      const page2 = await client.statuses.retrieve({
        keys,
        limit: 2,
        offset: 2,
      });

      expect(page1).toHaveLength(2);
      expect(page2).toHaveLength(2);

      const page1Keys = page1.map((s) => s.key);
      const page2Keys = page2.map((s) => s.key);
      expect(page1Keys.some((k) => page2Keys.includes(k))).toBe(false);
    });

    it("should retrieve a status with a details schema", async () => {
      const detailsSchema = z.object({
        name: z.string(),
        key: z.string(),
      });
      const s = await client.statuses.set<typeof detailsSchema>({
        name: "Details Schema",
        key: "details-schema",
        variant: "info",
        message: "Test",
        time: TimeStamp.now(),
        details: {
          name: "Details Schema",
          key: "details-schema",
        },
      });
      const retrieved = await client.statuses.retrieve<typeof detailsSchema>({
        key: s.key,
        detailsSchema,
      });
      expect(retrieved.key).toBe(s.key);
      expect(retrieved.name).toBe(s.name);
      expect(retrieved.details).toBeDefined();
      expect(retrieved.details.name).toBe(s.details.name);
      expect(retrieved.details.key).toBe(s.details.key);
    });
  });

  describe("delete", () => {
    it("should delete a status by key", async () => {
      const s = await client.statuses.set({
        name: "To Delete",
        key: "delete-me",
        variant: "error",
        message: "Will be deleted",
        time: TimeStamp.now(),
      });

      await client.statuses.delete(s.key);

      await expect(async () => await client.statuses.retrieve(s.key)).rejects.toThrow();
    });

    it("should delete multiple statuses", async () => {
      const keys = ["del-1", "del-2", "del-3"];
      await client.statuses.set(
        keys.map<status.Crude>((key) => ({
          name: `Delete ${key}`,
          key,
          variant: "info",
          message: "To be deleted",
          time: TimeStamp.now(),
        })),
      );

      await client.statuses.delete(keys);

      const results = await client.statuses.retrieve({ keys }).catch(() => []);
      expect(results).toHaveLength(0);
    });

    it("should be idempotent", async () => {
      const key = "idempotent-delete";

      await expect(client.statuses.delete(key)).resolves.not.toThrow();

      await client.statuses.set({
        name: "Idempotent",
        key,
        variant: "info",
        message: "Test",
        time: TimeStamp.now(),
      });

      await client.statuses.delete(key);

      await expect(client.statuses.delete(key)).resolves.not.toThrow();
    });
  });

  describe("with labels", () => {
    it("should correctly retrieve a status with labels attached", async () => {
      const label1 = await client.labels.create({
        name: "Label 1",
        color: color.construct("#0000FF"),
      });
      const label2 = await client.labels.create({
        name: "Label 2",
        color: color.construct("#FF0000"),
      });
      const stat = await client.statuses.set({
        name: "Idempotent",
        key: uuid.create(),
        variant: "info",
        message: "Test",
        time: TimeStamp.now(),
      });
      await client.labels.label(status.ontologyID(stat.key), [label1.key, label2.key], {
        replace: true,
      });

      const retrievedStat = await client.statuses.retrieve({
        key: stat.key,
        includeLabels: true,
      });
      expect(retrievedStat.key).toEqual(stat.key);
      expect(retrievedStat.labels).toHaveLength(2);
    });

    it("should update a subscribed hasLabels answer when labels change", async () => {
      const lbl = await client.labels.create({
        name: "Membership Label",
        color: color.construct("#00FF00"),
      });
      const member = uuid.create();
      const joiner = uuid.create();
      await client.statuses.set([
        {
          name: "Member",
          key: member,
          variant: "info",
          message: "member",
          time: TimeStamp.now(),
        },
        {
          name: "Joiner",
          key: joiner,
          variant: "info",
          message: "joiner",
          time: TimeStamp.now(),
        },
      ]);
      await client.labels.label(status.ontologyID(member), [lbl.key]);
      const params = { hasLabels: [lbl.key] };
      const off = client.statuses.onChange(params, () => {});
      try {
        const initial = await client.statuses.retrieve(params);
        expect(initial.map((s) => s.key)).toEqual([member]);
        await client.labels.label(status.ontologyID(joiner), [lbl.key]);
        await expect
          .poll(() => {
            const cached = client.statuses.getCached(params);
            return query.isLive(cached) && cached.some((s) => s.key === joiner);
          })
          .toBe(true);
        await client.labels.remove(status.ontologyID(member), [lbl.key]);
        await expect
          .poll(() => {
            const cached = client.statuses.getCached(params);
            return query.isLive(cached) && !cached.some((s) => s.key === member);
          })
          .toBe(true);
      } finally {
        off();
      }
    });
  });

  describe("retrieve with variants filter", () => {
    it("should filter statuses by a single variant", async () => {
      const prefix = `variant-filter-${Date.now()}`;
      await client.statuses.set([
        {
          name: "Success",
          key: `${prefix}-success`,
          variant: "success",
          message: "ok",
          time: TimeStamp.now(),
        },
        {
          name: "Error",
          key: `${prefix}-error`,
          variant: "error",
          message: "fail",
          time: TimeStamp.now(),
        },
        {
          name: "Warning",
          key: `${prefix}-warning`,
          variant: "warning",
          message: "warn",
          time: TimeStamp.now(),
        },
      ]);

      const results = await client.statuses.retrieve({
        keys: [`${prefix}-success`, `${prefix}-error`, `${prefix}-warning`],
        variants: ["error"],
      });

      expect(results).toHaveLength(1);
      expect(results[0].variant).toBe("error");
    });

    it("should filter statuses by multiple variants", async () => {
      const prefix = `variant-multi-${Date.now()}`;
      await client.statuses.set([
        {
          name: "Info",
          key: `${prefix}-info`,
          variant: "info",
          message: "info",
          time: TimeStamp.now(),
        },
        {
          name: "Error",
          key: `${prefix}-error`,
          variant: "error",
          message: "error",
          time: TimeStamp.now(),
        },
        {
          name: "Success",
          key: `${prefix}-success`,
          variant: "success",
          message: "success",
          time: TimeStamp.now(),
        },
      ]);

      const results = await client.statuses.retrieve({
        keys: [`${prefix}-info`, `${prefix}-error`, `${prefix}-success`],
        variants: ["info", "success"],
      });

      expect(results).toHaveLength(2);
      const variants = results.map((s) => s.variant);
      expect(variants).toContain("info");
      expect(variants).toContain("success");
      expect(variants).not.toContain("error");
    });

    it("should return empty when no statuses match variant", async () => {
      const prefix = `variant-none-${Date.now()}`;
      await client.statuses.set({
        name: "Info Only",
        key: `${prefix}-info`,
        variant: "info",
        message: "info",
        time: TimeStamp.now(),
      });

      const results = await client.statuses.retrieve({
        keys: [`${prefix}-info`],
        variants: ["error"],
      });

      expect(results).toHaveLength(0);
    });
  });

  describe("status variants", () => {
    it("should support all status variants", async () => {
      const variants: status.Variant[] = [
        "success",
        "info",
        "warning",
        "error",
        "loading",
        "disabled",
      ];

      const statuses = await client.statuses.set(
        variants.map((variant) => ({
          name: `Variant ${variant}`,
          key: `variant-${variant}-${Date.now()}`,
          variant,
          message: `Testing ${variant} variant`,
          time: TimeStamp.now(),
        })),
      );

      expect(statuses).toHaveLength(variants.length);
      statuses.forEach((s, i) => {
        expect(s.variant).toBe(variants[i]);
      });
    });
  });
});

describe("fromException", () => {
  class CustomError extends Error {
    toStatus() {
      return {
        message: "Failed to parse task config",
        description: "the formatted breakdown",
        details: { taskKey: "tk-1" },
      };
    }
  }

  it("should derive variant, message, and details from a plain Error", () => {
    const s = status.fromException(new Error("boom"));
    expect(s.variant).toBe("error");
    expect(s.message).toBe("boom");
    const details = s.details;
    expect(typeof details.stack).toBe("string");
    expect(details.error).toBeInstanceOf(Error);
  });

  it("should set the status message from a custom toStatus()", () => {
    expect(status.fromException(new CustomError("boom")).message).toBe(
      "Failed to parse task config",
    );
  });

  it("should set the status description from a custom toStatus()", () => {
    expect(status.fromException(new CustomError("boom")).description).toBe(
      "the formatted breakdown",
    );
  });

  it("should merge custom toStatus() details with the stack and original error", () => {
    const details = status.fromException(new CustomError("boom")).details;
    expect(details.taskKey).toBe("tk-1");
    expect(typeof details.stack).toBe("string");
    expect(details.error).toBeInstanceOf(Error);
  });

  it("should prefix a caller-provided message with the custom toStatus() message", () => {
    expect(status.fromException(new CustomError("boom"), "Saving failed").message).toBe(
      "Saving failed: Failed to parse task config",
    );
  });

  it("should surface the cause chain as the description", () => {
    const err = new Error("failed to reconcile projects cache", {
      cause: new Error("projects not found", { cause: new Error("query") }),
    });
    const s = status.fromException(err);
    expect(s.message).toBe("failed to reconcile projects cache");
    expect(s.description).toBe("projects not found: query");
  });

  it("should append the cause chain after a caller-provided message", () => {
    const err = new Error("outer", { cause: new Error("inner") });
    const s = status.fromException(err, "Saving failed");
    expect(s.message).toBe("Saving failed");
    expect(s.description).toBe("outer: inner");
  });

  it("should leave the description empty without a cause or message", () => {
    expect(status.fromException(new Error("boom")).description).toBe("");
  });

  describe("clone safety", () => {
    it("should keep the original error instance when it is cloneable", () => {
      const err = new Error("boom", { cause: new Error("root") });
      expect(status.fromException(err).details.error).toBe(err);
    });

    it("should rebuild an error whose cause cannot be cloned", () => {
      const err = new Error("boom", { cause: () => {} });
      err.name = "SocketError";
      const stored = status.fromException(err).details.error;
      expect(stored).not.toBe(err);
      expect(stored.message).toBe("boom");
      expect(stored.name).toBe("SocketError");
      expect(stored.stack).toBe(err.stack);
      expect(typeof stored.cause).toBe("string");
      expect(() => structuredClone(stored)).not.toThrow();
    });

    it("should rebuild an error carrying an un-cloneable own field", () => {
      const err = new Error("boom");
      (err as unknown as { cb: () => void }).cb = () => {};
      const stored = status.fromException(err).details.error;
      expect(stored).not.toBe(err);
      expect(() => structuredClone(stored)).not.toThrow();
    });

    it("should preserve error causes as errors in the rebuilt chain", () => {
      const root = new Error("root", { cause: () => {} });
      const err = new Error("boom", { cause: root });
      const stored = status.fromException(err).details.error;
      expect(stored.cause).toBeInstanceOf(Error);
      expect((stored.cause as Error).message).toBe("root");
      expect(() => structuredClone(stored)).not.toThrow();
    });

    it("should terminate on a cyclic cause chain", () => {
      const err = new Error("boom");
      err.cause = err;
      (err as unknown as { cb: () => void }).cb = () => {};
      const stored = status.fromException(err).details.error;
      expect(stored.message).toBe("boom");
      expect(() => structuredClone(stored)).not.toThrow();
    });

    it("should rebuild an error carrying a throwing enumerable getter", () => {
      const err = new Error("boom");
      Object.defineProperty(err, "trap", {
        enumerable: true,
        get() {
          throw new Error("trapped");
        },
      });
      const stored = status.fromException(err).details.error;
      // Passing err to expect would trip the getter while vitest inspects it.
      expect(stored === err).toBe(false);
      expect(stored.message).toBe("boom");
      expect(() => structuredClone(stored)).not.toThrow();
    });

    it("should rebuild when the cause carries a throwing enumerable getter", () => {
      const cause: Record<string, unknown> = { ok: 1 };
      Object.defineProperty(cause, "trap", {
        enumerable: true,
        get() {
          throw new Error("trapped");
        },
      });
      const err = new Error("boom", { cause });
      const stored = status.fromException(err).details.error;
      expect(stored).not.toBe(err);
      expect(typeof stored.cause).toBe("string");
      expect(() => structuredClone(stored)).not.toThrow();
    });

    it("should rebuild conservatively when the clone check budget is exhausted", () => {
      const wide: Record<string, number> = {};
      for (let i = 0; i < 100; i++) wide[`k${i}`] = i;
      const err = new Error("boom", { cause: wide });
      const stored = status.fromException(err).details.error;
      expect(stored).not.toBe(err);
      expect(() => structuredClone(stored)).not.toThrow();
    });
  });
});

describe("keepVariants", () => {
  it("should return undefined when variant is null", () => {
    expect(status.keepVariants(undefined, "success")).toBeUndefined();
  });

  it("should return undefined when variant is not in keep list", () => {
    expect(status.keepVariants("error", "success")).toBeUndefined();
    expect(status.keepVariants("error", ["success", "info"])).toBeUndefined();
  });

  it("should return variant when it matches single keep variant", () => {
    expect(status.keepVariants("success", "success")).toBe("success");
  });

  it("should return variant when it is in keep array", () => {
    expect(status.keepVariants("success", ["success", "info"])).toBe("success");
    expect(status.keepVariants("info", ["success", "info"])).toBe("info");
  });

  it("should return undefined when keep is empty array", () => {
    expect(status.keepVariants("success", [])).toBeUndefined();
  });
});

describe("removeVariants", () => {
  it("should return undefined when variant is null", () => {
    expect(status.removeVariants(undefined, "success")).toBeUndefined();
  });

  it("should return undefined when variant matches single remove variant", () => {
    expect(status.removeVariants("success", "success")).toBeUndefined();
  });

  it("should return undefined when variant is in remove array", () => {
    expect(status.removeVariants("success", ["success", "error"])).toBeUndefined();
    expect(status.removeVariants("error", ["success", "error"])).toBeUndefined();
  });

  it("should return variant when it does not match single remove variant", () => {
    expect(status.removeVariants("success", "error")).toBe("success");
  });

  it("should return variant when it is not in remove array", () => {
    expect(status.removeVariants("warning", ["success", "error"])).toBe("warning");
    expect(status.removeVariants("info", ["success", "error"])).toBe("info");
  });

  it("should return variant when remove is empty array", () => {
    expect(status.removeVariants("success", [])).toBe("success");
  });
});

describe("toString", () => {
  it("should render the variant, name, and message as the header", () => {
    const s = status.create({
      variant: "error",
      name: "Task",
      message: "failed to start",
    });
    expect(status.toString(s)).toBe("ERROR [Task]: failed to start");
  });

  it("should omit the description when the status carries an empty one", () => {
    const s = status.create({ variant: "info", message: "connected" });
    expect(status.toString(s)).toBe("INFO: connected");
  });

  it("should render a description when the status carries one", () => {
    const s = status.create({
      variant: "error",
      message: "failed to start",
      description: "channel is not writable",
    });
    expect(status.toString(s)).toBe(
      "ERROR: failed to start\n\nDescription: channel is not writable",
    );
  });
});
