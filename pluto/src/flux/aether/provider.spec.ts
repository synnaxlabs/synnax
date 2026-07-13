// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { type SynnaxParams } from "@synnaxlabs/client";
import { createTestClient, TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { describe, expect, it, vi } from "vitest";

import { type aether } from "@/aether/aether";
import { flux } from "@/flux/aether";
import { base } from "@/flux/base";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";

const MockSender = { send: vi.fn() };
const NOOP = alamos.Instrumentation.NOOP;

const STORE_CONFIG: base.StoreConfig<base.Store> = {};

const shouldNotCreate = (): never => {
  throw new Error("should not create a child");
};

const makeChild = <T extends aether.Node>(
  Ctor: new (props: aether.ComponentConstructorProps) => T,
  key: string,
  type: string,
  parent: aether.Node | null,
): T =>
  new Ctor({ path: [key], type, sender: MockSender, instrumentation: NOOP, parent });

/** Builds a real status.Aggregator -> synnax.aether.Provider -> flux.aether.Provider
 * chain, connecting the synnax node to the local test cluster with the given params.
 * Both ancestors are real, unmocked components — flux.aether.Provider resolves its
 * client through the same context-propagation path production uses. */
const makeTree = (
  key: string,
  props: SynnaxParams | null,
): { agg: aether.Node; synnaxProvider: aether.Node; flux: aether.Component } => {
  const agg = makeChild(
    status.Aggregator,
    `${key}-status`,
    status.Aggregator.TYPE,
    null,
  );
  agg._updateState({
    path: [`${key}-status`],
    state: { statuses: [] },
    type: status.Aggregator.TYPE,
    create: shouldNotCreate,
  });

  const synnaxProvider = makeChild(
    synnax.Provider,
    `${key}-synnax`,
    synnax.Provider.TYPE,
    agg,
  );
  synnaxProvider._updateState({
    path: [`${key}-synnax`],
    state: { props, state: null },
    type: synnax.Provider.TYPE,
    create: shouldNotCreate,
  });

  const registry = flux.createRegistry({ storeConfig: STORE_CONFIG });
  const FluxProvider = registry[flux.PROVIDER_TYPE];
  const fluxProvider = new FluxProvider({
    path: [`${key}-flux`],
    type: flux.PROVIDER_TYPE,
    sender: MockSender,
    instrumentation: NOOP,
    parent: synnaxProvider,
  });

  return { agg, synnaxProvider, flux: fluxProvider };
};

const updateFlux = (tree: { flux: aether.Component }, key: string) =>
  tree.flux._updateState({
    path: [`${key}-flux`],
    state: {},
    type: flux.PROVIDER_TYPE,
    create: shouldNotCreate,
  });

describe("flux.aether.Provider", () => {
  it("closes the owned store when the node is deleted", async () => {
    const key = "flux-provider-delete";
    const tree = makeTree(key, TEST_CLIENT_PARAMS);
    updateFlux(tree, key);

    const closeSpy = vi.spyOn(base.Client.prototype, "close");
    tree.flux._delete([`${key}-flux`]);
    await expect.poll(() => closeSpy.mock.calls.length).toBe(1);
    closeSpy.mockRestore();

    tree.synnaxProvider._delete([`${key}-synnax`]);
  });

  it("closes the previous store when the underlying client swaps", async () => {
    const key = "flux-provider-swap";
    const tree = makeTree(key, TEST_CLIENT_PARAMS);
    updateFlux(tree, key);

    const closeSpy = vi.spyOn(base.Client.prototype, "close");

    // A different `name` forces synnax.aether.Provider to treat this as a distinct
    // connection (deep.equal on props fails), so it opens a second real connection and
    // closes the first — which should cascade into flux.aether.Provider swapping (and
    // closing) its own store.
    tree.synnaxProvider._updateState({
      path: [`${key}-synnax`],
      state: { props: { ...TEST_CLIENT_PARAMS, name: "swap" }, state: null },
      type: synnax.Provider.TYPE,
      create: shouldNotCreate,
    });
    updateFlux(tree, key);

    await expect.poll(() => closeSpy.mock.calls.length).toBe(1);
    closeSpy.mockRestore();

    tree.flux._delete([`${key}-flux`]);
    tree.synnaxProvider._delete([`${key}-synnax`]);
  });

  it("never closes a caller-supplied client", async () => {
    const suppliedClient = new base.Client({
      client: createTestClient(),
      storeConfig: STORE_CONFIG,
      handleError: () => {},
      handleAsyncError: async () => {},
    });
    const registry = flux.createRegistry({ client: suppliedClient });
    const key = "flux-provider-unowned";
    const FluxProvider = registry[flux.PROVIDER_TYPE];
    const provider = new FluxProvider({
      path: [key],
      type: flux.PROVIDER_TYPE,
      sender: MockSender,
      instrumentation: NOOP,
      parent: null,
    });

    const closeSpy = vi.spyOn(suppliedClient, "close");
    provider._updateState({
      path: [key],
      state: {},
      type: flux.PROVIDER_TYPE,
      create: shouldNotCreate,
    });
    provider._delete([key]);
    expect(closeSpy).not.toHaveBeenCalled();

    await suppliedClient.close();
  });
});
