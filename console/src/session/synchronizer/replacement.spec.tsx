// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ChildProcess, spawn } from "node:child_process";
import { existsSync } from "node:fs";
import path from "node:path";

import { Drift } from "@synnaxlabs/drift";
import { Synnax } from "@synnaxlabs/pluto";
import { kv } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, describe, expect, it } from "vitest";

import { Session } from "@/session";
import { createSessionConsoleWrapper, type TestStore, uniqueName } from "@/testutil";

const PORT = 9091;
const ADDRESS = `localhost:${PORT}`;
const BINARY = path.resolve(__dirname, "../../../../core/synnax");

const portOccupied = async (): Promise<boolean> => {
  try {
    await fetch(`http://${ADDRESS}`);
    return true;
  } catch {
    return false;
  }
};

const startCore = async (): Promise<ChildProcess> => {
  // A leaked core from an earlier run would answer the readiness poll and make
  // every replacement assertion meaningless: refuse to race it.
  if (await portOccupied()) throw new Error(`port ${PORT} is already occupied`);
  const core = spawn(BINARY, ["start", "-mi", "-l", ADDRESS], { stdio: "ignore" });
  await waitFor(
    async () => {
      if (core.exitCode != null) throw new Error("core exited during startup");
      await fetch(`http://${ADDRESS}`);
    },
    { timeout: 20000 },
  );
  return core;
};

const stopCore = async (core: ChildProcess): Promise<void> => {
  if (core.exitCode != null || core.signalCode != null) return;
  const exited = new Promise<void>((resolve) => core.once("exit", () => resolve()));
  core.kill();
  await exited;
};

/** The cluster key the core at {@link ADDRESS} currently reports. */
const serverClusterKey = async (): Promise<string> => {
  const login = await fetch(`http://${ADDRESS}/api/v1/auth/login/`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username: "synnax", password: "seldon" }),
  });
  const { token } = (await login.json()) as { token: string };
  const res = await fetch(`http://${ADDRESS}/api/v1/connectivity/check/`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: "{}",
  });
  const body = (await res.json()) as { cluster_key?: string };
  return body.cluster_key ?? "";
};

const CLUSTER_KEY = "replacement-test";

const CLUSTER_ENTRY = {
  key: CLUSTER_KEY,
  name: "Replacement",
  host: "localhost",
  port: PORT,
  username: "synnax",
  password: "seldon",
  secure: false,
};

// Boots its own cores on a dedicated port, so it needs the core binary built.
describe.skipIf(!existsSync(BINARY))("cluster replacement reconciliation", () => {
  let core: ChildProcess | null = null;

  afterEach(async () => {
    if (core != null) await stopCore(core);
    core = null;
  });

  const replaceCore = async (adopted: string): Promise<void> => {
    if (core == null) throw new Error("no core running");
    await stopCore(core);
    core = await startCore();
    // Guard the harness itself: a replacement only happened if the address now
    // answers with a different cluster key.
    expect(await serverClusterKey()).not.toBe(adopted);
  };

  const createPersistedStore = async (db: kv.MockAsync): Promise<TestStore> =>
    await Session.createStore({
      runtime: new Drift.NoopRuntime<Session.State, Session.Action>(),
      enablePrerender: false,
      openKV: () => db,
    });

  const readPartition = async (
    db: kv.MockAsync,
    base: string,
  ): Promise<Record<string, unknown>> => {
    const version =
      (await db.get<{ version: number }>(`${base}.version`))?.version ?? 0;
    return (await db.get(`${base}.${version}`)) ?? {};
  };

  interface Harness {
    store: TestStore;
    result: { current: { settled: boolean; client: ReturnType<typeof Synnax.use> } };
    unmount: () => void;
  }

  const mountConsole = async (db: kv.MockAsync | null): Promise<Harness> => {
    const store = db == null ? undefined : await createPersistedStore(db);
    const { wrapper: Console, store: resolvedStore } =
      await createSessionConsoleWrapper({ client: null, store });
    const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <Console>
        <Session.Settled.Provider>{children}</Session.Settled.Provider>
      </Console>
    );
    const { result, unmount } = renderHook(
      () => ({ settled: Session.Settled.use(), client: Synnax.use() }),
      { wrapper: Wrapper },
    );
    return { store: resolvedStore, result, unmount };
  };

  const connectAndSelectProject = async ({
    store,
    result,
  }: Harness): Promise<{ adopted: string; projectKey: string }> => {
    act(() => {
      store.dispatch(Session.Cluster.set(CLUSTER_ENTRY));
      store.dispatch(Session.Cluster.select(CLUSTER_KEY));
    });
    await waitFor(() => expect(result.current.settled).toBe(true), {
      timeout: 20000,
    });
    const adopted = Session.Cluster.selectSelectedKey(store.getState());
    if (adopted == null) throw new Error("no cluster adopted");
    const client = result.current.client;
    if (client == null) throw new Error("no client");
    const project = await client.projects.create({
      name: uniqueName("replaced"),
      layout: {},
    });
    act(() => {
      store.dispatch(Session.Project.select(project.key));
    });
    await waitFor(() => expect(result.current.settled).toBe(true));
    return { adopted, projectKey: project.key };
  };

  // The bound is deliberately tight: replacement detection must come from the
  // probe backoff (capped at 5s) or the stream-reopen probe, never from
  // waiting out the 30s heartbeat.
  const expectRecovered = async (
    { store, result }: Harness,
    adopted: string,
  ): Promise<void> => {
    await waitFor(
      () => {
        const state = store.getState();
        expect(Session.Cluster.selectSelectedKey(state)).not.toBe(adopted);
        expect(state[Session.Project.SLICE_NAME].selected).toBeUndefined();
        expect(Session.Persist.selectSwapping(state)).toBe(false);
        expect(result.current.settled).toBe(true);
      },
      { timeout: 20000 },
    );
  };

  it("adopts the new key, clears the dead project, and resettles", async () => {
    core = await startCore();
    const harness = await mountConsole(null);
    try {
      const { adopted } = await connectAndSelectProject(harness);
      await replaceCore(adopted);
      await expectRecovered(harness, adopted);
    } finally {
      harness.unmount();
    }
  }, 120000);

  it("reconciles a replacement with the persistence middleware active", async () => {
    core = await startCore();
    const db = new kv.MockAsync();
    const harness = await mountConsole(db);
    try {
      const { adopted } = await connectAndSelectProject(harness);
      await replaceCore(adopted);
      await expectRecovered(harness, adopted);
    } finally {
      harness.unmount();
    }
  }, 120000);

  it("reconciles a stale disk state on a cold boot after replacement", async () => {
    core = await startCore();
    const db = new kv.MockAsync();
    const first = await mountConsole(db);
    let adopted: string, projectKey: string;
    try {
      ({ adopted, projectKey } = await connectAndSelectProject(first));
      // The debounced persist must land the selected project on disk before
      // the app "quits", or the cold boot has nothing stale to recover from.
      await waitFor(async () => {
        const partition = await readPartition(db, `cluster.${adopted}`);
        expect((partition.project as { selected?: string })?.selected).toBe(projectKey);
      });
    } finally {
      first.unmount();
    }
    await replaceCore(adopted);
    const second = await mountConsole(db);
    try {
      // Composed from disk: the dead cluster's key and project are selected.
      expect(Session.Cluster.selectSelectedKey(second.store.getState())).toBe(adopted);
      expect(second.store.getState()[Session.Project.SLICE_NAME].selected).toBe(
        projectKey,
      );
      await expectRecovered(second, adopted);
    } finally {
      second.unmount();
    }
  }, 120000);
});
