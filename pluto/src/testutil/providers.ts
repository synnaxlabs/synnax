// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import { TimeSpan } from "@synnaxlabs/x";
import { type z } from "zod";

import { type aether } from "@/aether/aether";
import { aetherTest } from "@/aether/test";
import { alamos } from "@/alamos/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/test";
import { theming } from "@/theming/aether";
import { SYNNAX_LIGHT } from "@/theming/base/theme";
import { type render } from "@/vis/render";
import { canvasTest } from "@/vis/render/test";
import { staleness } from "@/vis/staleness/aether";

const ALAMOS_KEY = "alamos";
const STATUS_KEY = "status";
const SYNNAX_KEY = "synnax";
const THEMING_KEY = "theming";
const TELEM_KEY = "telem";
const STALENESS_KEY = "staleness";
const RENDER_KEY = "render";

/**
 * Toggleable provider stack for Synnax test utilities. Each provider defaults to `true`
 * (mounted with harness defaults) except `render`, which is off until a recorder is
 * requested. Set a provider to `false` to drop it from the tree, or pass a config
 * object to override its initial state. The providers form a fixed nesting order
 * (`alamos → status → synnax → theming → telem → staleness → render`); disabling one
 * mounts the next directly under the previous enabled provider. A component that reads
 * context from a provider it disabled will not find it.
 */
export interface ProviderOptions {
  /** Alamos instrumentation provider. Defaults on. */
  alamos?: false | z.input<typeof alamos.providerStateZ>;
  /** Status aggregator. Defaults on with no statuses. */
  status?: false | z.input<typeof status.aggregatorStateZ>;
  /** Synnax client provider. Defaults on with no client. */
  synnax?: false | z.input<typeof synnax.Provider.stateZ>;
  /** Theming provider. Defaults on with `SYNNAX_LIGHT`. */
  theming?: false | z.input<typeof theming.Provider.z>;
  /** Telemetry provider. Defaults on with `TestFactory` + `NoopFactory`. */
  telem?: false | { factories?: telem.Factory[] };
  /** Staleness provider. Defaults on with a 250ms sweep. */
  staleness?: false | z.input<typeof staleness.Provider.z>;
  /** Canvas render context. Off by default. `true` injects a fresh recorder; pass a
   * {@link canvasTest.Recorder} to supply your own and assert on it afterward, or any
   * other `render.Context`-shaped value to draw through it. */
  render?: boolean | canvasTest.Recorder | render.Context;
  /** Extra component types to register on the worker tree. */
  registry?: aether.ComponentRegistry;
  /** Instrumentation for the worker driver. */
  instrumentation?: Instrumentation;
}

/** Provider instances mounted by {@link buildStack}, by name. A field is `null` when its
 * provider was disabled. */
export interface MountedProviders {
  alamos: alamos.Provider | null;
  status: status.Aggregator | null;
  synnax: synnax.Provider | null;
  theming: theming.Provider | null;
  telem: aether.Composite<typeof telem.providerStateZ> | null;
  staleness: staleness.Provider | null;
  render: canvasTest.RenderProvider | null;
}

/** Result of {@link buildStack}: the worker driver, the path where the deepest provider
 * sits (descendants mount under it), the mounted provider instances, and the render
 * recorder if one was created or supplied. The recorder is null when the caller
 * supplied a plain render context instead. */
export interface BuiltStack {
  driver: aetherTest.Driver;
  basePath: string[];
  providers: MountedProviders;
  recorder: canvasTest.Recorder | null;
}

const SWEEP_INTERVAL = TimeSpan.milliseconds(250);

const isOn = (opt: unknown): boolean => opt !== false;

const stateOf = (opt: unknown, fallback: unknown): unknown =>
  typeof opt === "object" && opt !== null ? opt : fallback;

const buildTelemProvider = (
  factories: telem.Factory[] = [],
): aether.ComponentConstructor =>
  telem.createProvider(
    () =>
      new telem.CompoundFactory([
        new telemTest.TestFactory(),
        new telem.NoopFactory(),
        ...factories,
      ]),
  );

/**
 * Build a worker-side Synnax provider stack from {@link ProviderOptions}, on top of a
 * fresh {@link aetherTest.Driver}. Shared by `render` (main side) and `renderAether`
 * (worker side); most tests should call one of those, not this directly.
 */
export const buildStack = (options: ProviderOptions = {}): BuiltStack => {
  const { registry = {}, instrumentation, render: renderOpt } = options;

  const supplied =
    renderOpt == null || typeof renderOpt === "boolean" ? null : renderOpt;
  const recorder: canvasTest.Recorder | null =
    renderOpt === true
      ? canvasTest.record()
      : supplied instanceof canvasTest.Recorder
        ? supplied
        : null;
  const renderContext = recorder ?? supplied;

  const telemFactories =
    typeof options.telem === "object" ? options.telem.factories : undefined;

  const fullRegistry: aether.ComponentRegistry = {
    ...alamos.REGISTRY,
    ...status.REGISTRY,
    ...synnax.REGISTRY,
    ...theming.REGISTRY,
    [telem.PROVIDER_TYPE]: buildTelemProvider(telemFactories),
    ...staleness.REGISTRY,
    [canvasTest.RenderProvider.TYPE]: canvasTest.RenderProvider,
    [aetherTest.Leaf.TYPE]: aetherTest.Leaf,
    [aetherTest.Composite.TYPE]: aetherTest.Composite,
    ...registry,
  };

  const driver = aetherTest.createDriver(fullRegistry, instrumentation);

  const providers: MountedProviders = {
    alamos: null,
    status: null,
    synnax: null,
    theming: null,
    telem: null,
    staleness: null,
    render: null,
  };
  const path: string[] = [aetherTest.ROOT_KEY];

  if (isOn(options.alamos)) {
    path.push(ALAMOS_KEY);
    driver.update(
      path,
      alamos.Provider.TYPE,
      alamos.providerStateZ.parse(stateOf(options.alamos, {})),
    );
    providers.alamos = driver.find<alamos.Provider>([...path]);
  }
  if (isOn(options.status)) {
    path.push(STATUS_KEY);
    driver.update(
      path,
      status.Aggregator.TYPE,
      status.aggregatorStateZ.parse(stateOf(options.status, { statuses: [] })),
    );
    providers.status = driver.find<status.Aggregator>([...path]);
  }
  if (isOn(options.synnax)) {
    path.push(SYNNAX_KEY);
    driver.update(
      path,
      synnax.Provider.TYPE,
      synnax.Provider.stateZ.parse(stateOf(options.synnax, { props: null })),
    );
    providers.synnax = driver.find<synnax.Provider>([...path]);
  }
  if (isOn(options.theming)) {
    path.push(THEMING_KEY);
    driver.update(
      path,
      theming.Provider.TYPE,
      theming.Provider.z.parse(
        stateOf(options.theming, { theme: SYNNAX_LIGHT, fontURLs: [] }),
      ),
    );
    providers.theming = driver.find<theming.Provider>([...path]);
  }
  if (isOn(options.telem)) {
    path.push(TELEM_KEY);
    driver.update(path, telem.PROVIDER_TYPE, {});
    providers.telem = driver.find<aether.Composite<typeof telem.providerStateZ>>([
      ...path,
    ]);
  }
  if (isOn(options.staleness)) {
    path.push(STALENESS_KEY);
    driver.update(
      path,
      staleness.Provider.TYPE,
      staleness.Provider.z.parse(
        stateOf(options.staleness, { sweepInterval: SWEEP_INTERVAL }),
      ),
    );
    providers.staleness = driver.find<staleness.Provider>([...path]);
  }
  if (renderContext != null) {
    path.push(RENDER_KEY);
    driver.update(path, canvasTest.RenderProvider.TYPE, { context: renderContext });
    providers.render = driver.find<canvasTest.RenderProvider>([...path]);
  }

  return { driver, basePath: [...path], providers, recorder };
};
