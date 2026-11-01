// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { disableActEnvironment } from "@synnaxlabs/lyra/testutil";
import { render, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";
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
    schema = probeStateZ;

    afterUpdate(ctx: aether.Context): void {
      seen.push(synnax.use(ctx));
    }

    afterDelete(): void {}
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
    const Tree = (): ReactElement => (
      <Aether.Provider worker={mainSide}>
        <Status.Aggregator log={() => {}}>
          <Synnax.Provider connParams={TEST_CLIENT_PARAMS}>
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
    const Tree = ({ host }: { host: string }): ReactElement => (
      <Aether.Provider worker={mainSide}>
        <Status.Aggregator log={() => {}}>
          <Synnax.Provider connParams={{ ...TEST_CLIENT_PARAMS, host }}>
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

  it("should keep its client when the error handler changes", () => {
    // A new log makes the aggregator hand out a new error handler.
    let log = (): void => {};
    const wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <AetherProvider>
        <Status.Aggregator log={log}>
          <Synnax.Provider connParams={TEST_CLIENT_PARAMS}>{children}</Synnax.Provider>
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
});
