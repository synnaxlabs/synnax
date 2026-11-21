// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connection, type Synnax as Client } from "@synnaxlabs/client";
import { testClientParams } from "@synnaxlabs/client/testutil";
import { Status as Base } from "@synnaxlabs/lyra/status";
import { disableActEnvironment } from "@synnaxlabs/lyra/testutil";
import { TimeSpan } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { aetherTest } from "@/aether/testutil";
import { Status } from "@/status";
import { status } from "@/status/aether";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";
import { createAetherProvider } from "@/testutil/Aether";

const PROBE_TYPE = "SynnaxProbe";

const probeStateZ = z.object({});

const createProbe = (seen: Array<Client | null>) =>
  class Probe extends aether.Leaf<typeof probeStateZ> {
    static readonly TYPE = PROBE_TYPE;
    override schema = probeStateZ;

    override afterUpdate(ctx: aether.Context): void {
      seen.push(synnax.use(ctx));
    }

    override afterDelete(): void {}
  };

const ProbeComponent = (): null => {
  Aether.useLifecycle({ type: PROBE_TYPE, schema: probeStateZ, initialState: {} });
  return null;
};

const PageClient = ({ onClient }: { onClient: (c: Client | null) => void }): null => {
  onClient(Synnax.use());
  return null;
};

const AetherProvider = createAetherProvider({
  ...synnax.REGISTRY,
  ...status.REGISTRY,
});

describe("Synnax.Provider", () => {
  it("should have the worker join the page client's session", async () => {
    disableActEnvironment();
    const seen: Array<Client | null> = [];
    const { mainSide } = aetherTest.createDriver({
      ...synnax.REGISTRY,
      ...status.REGISTRY,
      [PROBE_TYPE]: createProbe(seen),
    });
    let page: Client | null = null;
    const connParams = testClientParams();
    const Tree = (): ReactElement => (
      <Aether.Provider worker={mainSide}>
        <Status.Aggregator log={() => {}}>
          <Synnax.Provider connParams={connParams}>
            <PageClient onClient={(c) => (page = c)} />
            <ProbeComponent />
          </Synnax.Provider>
        </Status.Aggregator>
      </Aether.Provider>
    );
    const { unmount } = render(<Tree />);
    await waitFor(() => expect(seen.at(-1)).toBeDefined());
    await waitFor(() => expect(seen.at(-1)).not.toBeNull());
    const worker = seen.at(-1);
    if (worker == null || page == null) throw new Error("expected both clients");
    const pageClient: Client = page;
    expect(worker.params.password).toBeUndefined();
    expect(worker.params.token).toEqual(pageClient.auth.token);
    await worker.connect();
    expect(worker.auth.session?.key).toEqual(pageClient.auth.session?.key);
    unmount();
  });

  it("should not hand the worker one Core's token for another Core", async () => {
    disableActEnvironment();
    const seen: Array<Client | null> = [];
    const { mainSide } = aetherTest.createDriver({
      ...synnax.REGISTRY,
      ...status.REGISTRY,
      [PROBE_TYPE]: createProbe(seen),
    });
    const pages: Client[] = [];
    const connParams = testClientParams();
    const Tree = ({ host }: { host: string }): ReactElement => (
      <Aether.Provider worker={mainSide}>
        <Status.Aggregator log={() => {}}>
          <Synnax.Provider connParams={{ ...connParams, host }}>
            <PageClient onClient={(c) => c != null && pages.push(c)} />
            <ProbeComponent />
          </Synnax.Provider>
        </Status.Aggregator>
      </Aether.Provider>
    );
    const tokenOf = (host: string): string | undefined =>
      pages.findLast((p) => p.params.host === host)?.auth.token;
    const { rerender, unmount } = render(<Tree host="localhost" />);
    await waitFor(() => expect(seen.at(-1)?.params.host).toEqual("localhost"));
    const first = tokenOf("localhost");
    rerender(<Tree host="127.0.0.1" />);
    await waitFor(() => expect(seen.at(-1)?.params.host).toEqual("127.0.0.1"));
    const second = tokenOf("127.0.0.1");
    expect(second).not.toEqual(first);
    const tokens = seen
      .filter((w) => w?.params.host === "127.0.0.1")
      .map((w) => w?.params.token);
    expect(tokens).not.toContain(first);
    unmount();
  });

  describe("status", () => {
    const CONNECTED: connection.Status = {
      ...connection.DEFAULT_STATUS,
      variant: "success",
      message: "Connected",
    };
    const createStatusWrapper = () => {
      const control: { set: (status: connection.Status) => void } = { set: () => {} };
      const Holder = ({ children }: PropsWithChildren): ReactElement => {
        const [status, setStatus] = useState(connection.DEFAULT_STATUS);
        control.set = setStatus;
        return (
          <Synnax.TestProvider client={null} status={status}>
            {children}
          </Synnax.TestProvider>
        );
      };
      const wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <AetherProvider>
          <Holder>{children}</Holder>
        </AetherProvider>
      );
      return { control, wrapper };
    };

    it("should not re-render a reader of the client when the status changes", () => {
      const { control, wrapper } = createStatusWrapper();
      let renders = 0;
      const { unmount } = renderHook(
        () => {
          renders += 1;
          return Synnax.use();
        },
        { wrapper },
      );
      const before = renders;
      act(() => control.set(CONNECTED));
      expect(renders).toEqual(before);
      unmount();
    });

    it("should re-render a reader of the status when the status changes", () => {
      const { control, wrapper } = createStatusWrapper();
      const { result, unmount } = renderHook(Synnax.useConnectionStatus, { wrapper });
      expect(result.current).toBe(connection.DEFAULT_STATUS);
      act(() => control.set(CONNECTED));
      expect(result.current).toBe(CONNECTED);
      unmount();
    });
  });

  it("should keep its client when the error handler changes", () => {
    // A new log makes the aggregator hand out a new error handler.
    let log = (): void => {};
    const connParams = testClientParams();
    const wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <AetherProvider>
        <Status.Aggregator log={log}>
          <Synnax.Provider connParams={connParams}>{children}</Synnax.Provider>
        </Status.Aggregator>
      </AetherProvider>
    );
    const { result, rerender, unmount } = renderHook(Synnax.use, { wrapper });
    const client = result.current;
    expect(client).not.toBeNull();
    log = () => {};
    rerender();
    expect(result.current).toBe(client);
    unmount();
  });

  it("should add a status only when clock skew starts and when it ends", async () => {
    disableActEnvironment();
    // The page and worker clients both log the skew.
    vi.spyOn(console, "warn").mockImplementation(() => {});
    vi.spyOn(console, "info").mockImplementation(() => {});
    let skew = TimeSpan.hours(1);
    let checks = 0;
    const realFetch = globalThis.fetch;
    // Shifts the time the real Core reports, so this machine's clock looks off by skew.
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const res = await realFetch(input, init);
      if (typeof input !== "string" || !input.endsWith("/connectivity/check"))
        return res;
      checks += 1;
      const body = (await res.text()).replace(
        /"node_time":"(\d+)"/,
        (_, t: string) => `"node_time":"${BigInt(t) - skew.valueOf()}"`,
      );
      return new Response(body, { status: res.status });
    });
    const passChecks = async (count: number): Promise<void> => {
      const target = checks + count;
      await waitFor(() => expect(checks).toBeGreaterThanOrEqual(target));
    };
    const connParams = {
      ...testClientParams(),
      connectivityPollFrequency: TimeSpan.milliseconds(20),
    };
    const wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <AetherProvider>
        <Status.Aggregator log={() => {}}>
          <Synnax.Provider connParams={connParams}>{children}</Synnax.Provider>
        </Status.Aggregator>
      </AetherProvider>
    );
    const { result, unmount } = renderHook(
      () => Base.useNotifications({ expiration: TimeSpan.minutes(1) }),
      { wrapper },
    );
    const withMessage = (message: string) =>
      result.current.statuses.filter((s) => s.message === message);
    await waitFor(() => expect(withMessage("Clock skew detected")).toHaveLength(1));
    skew = TimeSpan.hours(2);
    await passChecks(6);
    expect(withMessage("Clock skew detected")).toEqual([
      expect.objectContaining({ variant: "warning", count: 1 }),
    ]);
    skew = TimeSpan.ZERO;
    await waitFor(() => expect(withMessage("Clock skew resolved")).toHaveLength(1));
    await passChecks(6);
    expect(withMessage("Clock skew resolved")).toEqual([
      expect.objectContaining({ variant: "success", count: 1 }),
    ]);
    expect(withMessage("Clock skew detected")).toHaveLength(1);
    unmount();
    vi.restoreAllMocks();
  });
});
