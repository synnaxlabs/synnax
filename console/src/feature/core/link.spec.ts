// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Synnax } from "@synnaxlabs/pluto";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Core } from "@/feature/core";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import { createSessionConsoleWrapper } from "@/testutil";

// The live Core a link must name, since useLink refuses a client that landed on a
// cluster the link did not name.
let LIVE = "";

beforeAll(async () => {
  const client = createTestClient();
  await client.connect();
  LIVE = client.connection.status.details.clusterKey;
  expect(LIVE).not.toEqual("");
});

const renderLink = async (cores: Session.Core.Core[], selected: string) => {
  const { wrapper, store } = await createSessionConsoleWrapper({
    client: null,
    preloadedState: createCoreState(cores, selected),
  });
  const { result } = renderHook(
    () => ({ link: Core.useLink(), client: Synnax.use() }),
    { wrapper },
  );
  await waitFor(() => expect(result.current.client).not.toBeNull());
  return { result, store };
};

describe("useLink", () => {
  it("should resolve the active Core's managed client", async () => {
    const core = createCore("Local", { clusterKey: LIVE });
    const { result } = await renderLink([core], core.key);
    const active = result.current.client;
    expect(await result.current.link(LIVE)).toBe(active);
  });

  it("should throw if no Core reaches the cluster", async () => {
    const core = createCore("Local", { clusterKey: LIVE });
    const { result, store } = await renderLink([core], core.key);
    await expect(result.current.link("missing")).rejects.toThrow(
      "No Core connects to cluster missing",
    );
    expect(Session.Core.selectSelectedKey(store.getState())).toBe(core.key);
  });

  // Two records reaching one cluster is the whole point of keying links by cluster: a
  // link made against one machine's record opens through whichever record this machine
  // holds for the same cluster.
  it("should switch to the Core that names the cluster", async () => {
    const here = createCore("Here", { clusterKey: "cluster-a" });
    const there = createCore("There", { clusterKey: LIVE });
    const { result, store } = await renderLink([here, there], here.key);
    const prior = result.current.client;
    const next = await result.current.link(LIVE);
    expect(next).not.toBe(prior);
    expect(Session.Core.selectSelectedKey(store.getState())).toBe(there.key);
    await waitFor(() => expect(result.current.client).toBe(next));
  });

  it("should reject when the Core the link names cannot be reached", async () => {
    const live = createCore("Live", { clusterKey: LIVE });
    const dead = createCore("Dead", { clusterKey: "dead", port: 9999 });
    const { result } = await renderLink([live, dead], live.key);
    await expect(result.current.link("dead")).rejects.toThrow();
  });

  // A record's cached cluster is only as fresh as its last connection, so a link
  // resolved through a repointed record can land on another cluster entirely.
  it("should throw when the Core reaches a different cluster", async () => {
    const core = createCore("Stale", { clusterKey: "stale" });
    const { result } = await renderLink([core], core.key);
    await expect(result.current.link("stale")).rejects.toThrow(
      `Connected to cluster ${LIVE}, not stale`,
    );
  });
});
