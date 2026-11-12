// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { vcs } from "@/vcs";

const HASH = "3f9a1c7e5b2d4f6a8c0e1b3d5f7a9c2e4b6d8f0a";

const COMMIT = {
  hash: HASH,
  shortHash: "3f9a1c7",
  message: "Tune the regulator",
  author: "synnax",
  time: 1_700_000_000_000_000_000n,
  changes: null,
};

const CHANGE = {
  resource: "schematic:6f1c2a3b-4d5e-4f60-8a71-9b2c3d4e5f60",
  type: "schematic",
  name: "Engine",
  kind: "modified",
  path: "schematics/engine.json",
  project: "",
  digest: "a1b2",
  review: null,
};

const STATUS = {
  enabled: true,
  configuration: "main",
  remoteUrl: "",
  tokenSet: false,
  head: COMMIT,
  changes: [CHANGE],
  incoming: null,
  incomingChanges: null,
  waiting: 0,
  fetchedAt: 0,
  error: "",
};

const INVERSE = { files: null, removed: null, configuration: "" };

// A Core encodes an empty list as null.
const RESPONSES: Record<string, unknown> = {
  "/vcs/status": STATUS,
  "/vcs/connect": STATUS,
  "/vcs/fetch": STATUS,
  "/vcs/clone": STATUS,
  "/vcs/commit": { commit: COMMIT },
  "/vcs/discard": INVERSE,
  "/vcs/apply": INVERSE,
  "/vcs/write": {},
  "/vcs/plan": { target: { kind: "pull", resources: null }, items: null },
  "/vcs/files": { files: null },
  "/vcs/log": { commits: [COMMIT] },
  "/vcs/show": { commit: { ...COMMIT, changes: [CHANGE] } },
  "/vcs/configurations": {
    configurations: [{ name: "main", head: COMMIT, current: true }],
  },
  "/vcs/compare": { changes: null },
  "/vcs/snapshot": {
    snapshot: {
      range: "r1",
      tag: "snapshot/r1/1",
      commit: COMMIT,
      configuration: "main",
      uncommitted: 2,
    },
  },
  "/vcs/snapshots": { snapshots: null },
};

/** A client whose Core answers each route with its response in RESPONSES. */
const createClient = () => {
  const unary: UnaryClient = {
    use: () => {},
    send: async (target, req, reqSchema, resSchema) => {
      reqSchema.parse(req);
      return resSchema.parse(RESPONSES[target]);
    },
  };
  const send = vi.spyOn(unary, "send");
  return { client: new vcs.Client({ unary }), send };
};

const PULL: vcs.Target = { kind: "pull" };

describe("vcs", () => {
  describe("Client", () => {
    describe("requests", () => {
      it.each<[string, (c: vcs.Client) => Promise<unknown>, string, unknown]>([
        ["status", async (c) => await c.status(), "/vcs/status", {}],
        [
          "connect",
          async (c) =>
            await c.connect({ configuration: "main", remoteUrl: "", token: "" }),
          "/vcs/connect",
          { configuration: "main", remoteUrl: "", token: "" },
        ],
        ["fetch", async (c) => await c.fetch(), "/vcs/fetch", {}],
        [
          "commit",
          async (c) =>
            await c.commit({ message: "m", resources: ["a"], pullRequest: true }),
          "/vcs/commit",
          { message: "m", resources: ["a"], pullRequest: true },
        ],
        [
          "discard",
          async (c) => await c.discard(["a"]),
          "/vcs/discard",
          { resources: ["a"] },
        ],
        [
          "write",
          async (c) => await c.write({ files: [], removed: ["a"], configuration: "" }),
          "/vcs/write",
          { files: [], removed: ["a"], configuration: "" },
        ],
        ["plan", async (c) => await c.plan(PULL), "/vcs/plan", { target: PULL }],
        [
          "apply",
          async (c) => await c.apply(PULL),
          "/vcs/apply",
          { target: PULL, choices: [] },
        ],
        [
          "files",
          async (c) => await c.files(vcs.HEAD, "a"),
          "/vcs/files",
          { rev: "head", resource: "a" },
        ],
        [
          "log",
          async (c) => await c.log({ configuration: "main", limit: 5 }),
          "/vcs/log",
          { configuration: "main", limit: 5 },
        ],
        ["show", async (c) => await c.show(HASH), "/vcs/show", { commit: HASH }],
        [
          "configurations",
          async (c) => await c.configurations(),
          "/vcs/configurations",
          {},
        ],
        [
          "clone",
          async (c) => await c.clone("hotfire"),
          "/vcs/clone",
          { name: "hotfire" },
        ],
        [
          "compare",
          async (c) => await c.compare(vcs.HEAD, vcs.WORKING),
          "/vcs/compare",
          { from: "head", to: "working" },
        ],
        [
          "snapshot",
          async (c) => await c.snapshot("r1"),
          "/vcs/snapshot",
          { range: "r1" },
        ],
        [
          "snapshots",
          async (c) => await c.snapshots("r1"),
          "/vcs/snapshots",
          { range: "r1" },
        ],
      ])("should send %s to its route", async (_, call, target, req) => {
        const { client, send } = createClient();
        await call(client);
        expect(send).toHaveBeenCalledExactlyOnceWith(
          target,
          req,
          expect.anything(),
          expect.anything(),
        );
      });

      it("should send the choices of an apply", async () => {
        const { client, send } = createClient();
        await client.apply(PULL, [{ resource: "a", choice: "incoming" }]);
        expect(send.mock.calls[0][1]).toEqual({
          target: PULL,
          choices: [{ resource: "a", choice: "incoming" }],
        });
      });

      it("should name the rev of a configuration", () => {
        expect(vcs.configurationRev("hotfire")).toBe("configuration/hotfire");
      });
    });

    describe("responses", () => {
      it("should read null lists of a status as empty", async () => {
        const { client } = createClient();
        const status = await client.status();
        expect(status.incoming).toEqual([]);
        expect(status.incomingChanges).toEqual([]);
        expect(status.changes.map(({ name }) => name)).toEqual(["Engine"]);
        expect(status.changes[0].review).toBeUndefined();
        expect(status.head?.hash).toBe(HASH);
        expect(status.head?.changes).toBeUndefined();
        expect(status.head?.time).toEqual(new TimeStamp(1_700_000_000_000_000_000n));
      });

      it("should return a null list of a response as empty", async () => {
        const { client } = createClient();
        expect(await client.files(vcs.HEAD, "a")).toEqual([]);
        expect(await client.compare(vcs.HEAD, vcs.WORKING)).toEqual([]);
        expect(await client.snapshots("r1")).toEqual([]);
        expect(await client.discard(["a"])).toEqual({
          files: [],
          removed: [],
          configuration: "",
        });
      });

      it("should return the commits, configurations, and snapshot", async () => {
        const { client } = createClient();
        expect((await client.log({ limit: 1 })).map(({ hash }) => hash)).toEqual([
          HASH,
        ]);
        const [configuration] = await client.configurations();
        expect(configuration).toMatchObject({ name: "main", current: true });
        expect(configuration.head.hash).toBe(HASH);
        const snapshot = await client.snapshot("r1");
        expect(snapshot).toMatchObject({ range: "r1", uncommitted: 2 });
      });

      it("should return a shown commit with its changes", async () => {
        const { client } = createClient();
        const commit = await client.show(HASH);
        expect(commit.changes?.map(({ resource }) => resource)).toEqual([
          CHANGE.resource,
        ]);
      });

      it("should return an empty URL for a commit without a pull request", async () => {
        const { client } = createClient();
        const res = await client.commit({
          message: "m",
          resources: ["a"],
          pullRequest: false,
        });
        expect(res.pullRequestUrl).toBe("");
        expect(res.commit.hash).toBe(HASH);
      });

      it("should read a plan without items or target resources", async () => {
        const { client } = createClient();
        const plan = await client.plan(PULL);
        expect(plan.items).toEqual([]);
        expect(plan.target.kind).toBe("pull");
        expect(plan.target.resources).toBeUndefined();
      });
    });
  });
});
