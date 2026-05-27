// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos as alamosLib, type Instrumentation } from "@synnaxlabs/alamos";
import { UnexpectedError } from "@synnaxlabs/client";
import { type z } from "zod";

import { aether } from "@/aether/aether";
import { RenderProvider } from "@/aether/test/RenderProvider";
import { TestComposite } from "@/aether/test/TestComposite";
import { TestLeaf } from "@/aether/test/TestLeaf";
import { alamos } from "@/alamos/aether";
import { state } from "@/state";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/test";
import { theming } from "@/theming/aether";
import { SYNNAX_LIGHT } from "@/theming/base/theme";

const ROOT_KEY = "root";
const ALAMOS_KEY = "alamos";
const STATUS_KEY = "status";
const SYNNAX_KEY = "synnax";
const THEMING_KEY = "theming";
const TELEM_KEY = "telem";
const RENDER_KEY = "render";

/** Per-provider state overrides accepted by the harness. Each key is parsed against
 * the corresponding provider's real Zod schema and merged on top of harness defaults
 * before mounting. */
export interface ProviderOverrides {
  alamos?: z.input<typeof alamos.providerStateZ>;
  status?: z.input<typeof status.aggregatorStateZ>;
  synnax?: z.input<typeof synnax.Provider.stateZ>;
  theming?: z.input<typeof theming.Provider.z>;
  telem?: {
    factories?: telem.Factory[];
  };
}

/** Initial state for a child component to mount under the component under test. */
export interface MountChild {
  type: string;
  state: state.State;
}

/** References to each provider in the test stack, exposed for tests that need to drive
 * mid-run state changes (e.g., disconnect the synnax client, emit a status). */
export interface MountedProviders {
  alamos: alamos.Provider;
  status: status.Aggregator;
  synnax: synnax.Provider;
  theming: theming.Provider;
  telem: aether.Composite<typeof telem.providerStateZ>;
  render: RenderProvider | null;
}

/** Common options for building a provider stack on the worker side. */
export interface StackOptions {
  registry?: aether.ComponentRegistry;
  providers?: ProviderOverrides;
  instrumentation?: Instrumentation;
  /** Render context (or duck-typed equivalent) to inject. Omit for tests that don't
   * render. */
  renderContext?: unknown;
}

/** Internal handle returned by {@link buildProviderStack}; consumed by both
 * {@link mount} and the React-integration wrapper. */
export interface ProviderStack {
  root: aether.Root;
  workerSide: aether.WorkerComms;
  mainSide: aether.MainComms;
  /** Path of the deepest provider, where children of the stack should mount. */
  basePath: readonly string[];
  providers: MountedProviders;
  fullRegistry: aether.ComponentRegistry;
  driveUpdate: (path: readonly string[], type: string, stateValue: state.State) => void;
  driveDelete: (path: readonly string[]) => void;
}

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
 * Builds the worker-side provider stack — `Root → alamos → status → synnax → theming
 * → telem` plus an optional `RenderProvider`. Returns the root, the worker/main comms
 * pair, the path under which descendants should mount, references to each provider
 * instance, and direct-drive helpers.
 *
 * Used internally by {@link mount} (Layer 1) and `renderAether` (Layer 2). Most tests
 * should not call this directly.
 */
export const buildProviderStack = (options: StackOptions = {}): ProviderStack => {
  const {
    registry = {},
    providers: providerOverrides = {},
    instrumentation = alamosLib.Instrumentation.NOOP,
    renderContext,
  } = options;

  const [workerSide, mainSide] = aether.createMockPair();

  const TelemProvider = buildTelemProvider(providerOverrides.telem?.factories);

  const fullRegistry: aether.ComponentRegistry = {
    ...alamos.REGISTRY,
    ...status.REGISTRY,
    ...synnax.REGISTRY,
    ...theming.REGISTRY,
    [telem.PROVIDER_TYPE]: TelemProvider,
    [RenderProvider.TYPE]: RenderProvider,
    [TestLeaf.TYPE]: TestLeaf,
    [TestComposite.TYPE]: TestComposite,
    ...registry,
  };

  const root = aether.render({
    worker: workerSide,
    registry: fullRegistry,
    instrumentation,
  });

  const driveUpdate = (
    path: readonly string[],
    type: string,
    stateValue: state.State,
  ): void => {
    root._updateState({
      path,
      type,
      state: stateValue,
      create: (parentCtxValues) => {
        const Constructor = fullRegistry[type];
        if (Constructor == null)
          throw new UnexpectedError(`[aetherTest] type '${type}' not in registry`);
        return new Constructor({
          key: path[path.length - 1],
          type,
          sender: workerSide,
          instrumentation,
          parentCtxValues,
        });
      },
    });
  };

  const driveDelete = (path: readonly string[]): void => {
    root._delete(path);
  };

  driveUpdate(
    [ROOT_KEY, ALAMOS_KEY],
    alamos.Provider.TYPE,
    alamos.providerStateZ.parse(providerOverrides.alamos ?? {}),
  );
  driveUpdate(
    [ROOT_KEY, ALAMOS_KEY, STATUS_KEY],
    status.Aggregator.TYPE,
    status.aggregatorStateZ.parse(providerOverrides.status ?? { statuses: [] }),
  );
  driveUpdate(
    [ROOT_KEY, ALAMOS_KEY, STATUS_KEY, SYNNAX_KEY],
    synnax.Provider.TYPE,
    synnax.Provider.stateZ.parse(
      providerOverrides.synnax ?? { props: null, state: null },
    ),
  );
  driveUpdate(
    [ROOT_KEY, ALAMOS_KEY, STATUS_KEY, SYNNAX_KEY, THEMING_KEY],
    theming.Provider.TYPE,
    theming.Provider.z.parse(
      providerOverrides.theming ?? { theme: SYNNAX_LIGHT, fontURLs: [] },
    ),
  );
  driveUpdate(
    [ROOT_KEY, ALAMOS_KEY, STATUS_KEY, SYNNAX_KEY, THEMING_KEY, TELEM_KEY],
    telem.PROVIDER_TYPE,
    {},
  );

  const basePath: string[] = [
    ROOT_KEY,
    ALAMOS_KEY,
    STATUS_KEY,
    SYNNAX_KEY,
    THEMING_KEY,
    TELEM_KEY,
  ];

  if (renderContext !== undefined) {
    driveUpdate([...basePath, RENDER_KEY], RenderProvider.TYPE, {
      context: renderContext,
    });
    basePath.push(RENDER_KEY);
  }

  const findAt = <T extends aether.Component>(path: readonly string[]): T => {
    const subPath = path.slice(1);
    const found = root.findChildAtPath([...subPath]);
    if (found == null)
      throw new UnexpectedError(
        `[aetherTest] component at ${path.join(".")} not found after mount`,
      );
    return found as T;
  };

  const providers: MountedProviders = {
    alamos: findAt<alamos.Provider>([ROOT_KEY, ALAMOS_KEY]),
    status: findAt<status.Aggregator>([ROOT_KEY, ALAMOS_KEY, STATUS_KEY]),
    synnax: findAt<synnax.Provider>([ROOT_KEY, ALAMOS_KEY, STATUS_KEY, SYNNAX_KEY]),
    theming: findAt<theming.Provider>([
      ROOT_KEY,
      ALAMOS_KEY,
      STATUS_KEY,
      SYNNAX_KEY,
      THEMING_KEY,
    ]),
    telem: findAt<aether.Composite<typeof telem.providerStateZ>>([
      ROOT_KEY,
      ALAMOS_KEY,
      STATUS_KEY,
      SYNNAX_KEY,
      THEMING_KEY,
      TELEM_KEY,
    ]),
    render: renderContext !== undefined ? findAt<RenderProvider>([...basePath]) : null,
  };

  return {
    root,
    workerSide,
    mainSide,
    basePath,
    providers,
    fullRegistry,
    driveUpdate,
    driveDelete,
  };
};

/** Options for {@link mount}. `state` is typed as the schema's input — fields with
 * Zod defaults may be omitted. Everything else is optional. */
export interface MountOptions<S extends z.ZodType<state.State>> extends StackOptions {
  state: z.input<S>;
  /** Instance key for the component under test. Defaults to the component's TYPE. */
  key?: string;
  /** Initial children to mount under the component under test. */
  children?: Record<string, MountChild>;
}

/** Constructor signature for an aether component class accepted by {@link mount}. */
export interface ComponentClass<S extends z.ZodType<state.State>> {
  TYPE: string;
  stateZ: S;
  new (props: aether.ComponentConstructorProps): aether.Component;
}

/** Handle returned by {@link mount}. Drives state updates, inspects the worker-side
 * tree, and tears down the stack. */
export interface Handle<C extends aether.Component, S extends z.ZodType<state.State>> {
  /** The component under test, with the type the test passed to {@link mount}. */
  readonly component: C;
  /** Current parsed state of the component under test. */
  readonly state: z.infer<S>;
  /** Replace the component's state. Accepts either a value or a `(prev) => next`
   * setter. State is parsed against the component's Zod schema. */
  setState(next: state.SetArg<z.infer<S>>): void;
  /** Create or replace a child of the component under test. `type` must be in the
   * registry passed to {@link mount} or in the bundled stubs. */
  setChildState(key: string, type: string, childState: state.State): void;
  /** Delete a child of the component under test. */
  deleteChild(key: string): void;
  /** Look up a child instance by key. The `T` cast is unchecked; callers must align
   * the type with the registered class. */
  child<T extends aether.Component = aether.Component>(key: string): T;
  /** Provider instances in the stack, exposed for direct manipulation. */
  readonly providers: MountedProviders;
  /** Delete the entire tree and release references. */
  unmount(): void;
}

/**
 * Mount an aether component for a test, with a full production-shaped provider stack
 * wrapped around it on the worker side.
 *
 * The harness builds the real chain `Root → alamos.Provider → status.Aggregator →
 * synnax.Provider → theming.Provider → telem.Provider → <component>`. Every level runs
 * its real `afterUpdate` lifecycle and propagates context the way production does. The
 * test never constructs `parentCtxValues` by hand.
 *
 * Defaults: alamos at `level: "info"`, status with no statuses, synnax with no client,
 * theming with `SYNNAX_LIGHT`, telem with `TestFactory` + `NoopFactory`. Each is
 * overridable via {@link ProviderOverrides}. Pass `renderContext` to inject a
 * render-context-shaped value (typically `canvasTest.record()`).
 */
export const mount = <S extends z.ZodType<state.State>>(
  Component: ComponentClass<S>,
  options: MountOptions<S>,
): Handle<aether.Component, S> => {
  const {
    state: initialState,
    key = Component.TYPE,
    children = {},
    ...stackOptions
  } = options;

  const stack = buildProviderStack({
    ...stackOptions,
    registry: {
      ...stackOptions.registry,
      [Component.TYPE]: Component,
    },
  });

  const componentPath = [...stack.basePath, key];
  stack.driveUpdate(
    componentPath,
    Component.TYPE,
    Component.stateZ.parse(initialState),
  );

  for (const [childKey, child] of Object.entries(children))
    stack.driveUpdate([...componentPath, childKey], child.type, child.state);

  const subPath = componentPath.slice(1);
  const component = stack.root.findChildAtPath([...subPath]);
  if (component == null)
    throw new UnexpectedError(
      `[aetherTest.mount] component at ${componentPath.join(".")} not found after mount`,
    );

  return {
    component,
    get state(): z.infer<S> {
      const leaf = component as aether.Leaf<S>;
      return leaf.state;
    },
    setState(next: state.SetArg<z.infer<S>>): void {
      const leaf = component as aether.Leaf<S>;
      const nextState = state.executeSetter(next, leaf.state);
      stack.driveUpdate(componentPath, Component.TYPE, nextState);
    },
    setChildState(childKey, type, childState) {
      stack.driveUpdate([...componentPath, childKey], type, childState);
    },
    deleteChild(childKey) {
      stack.driveDelete([...componentPath, childKey]);
    },
    child<T extends aether.Component = aether.Component>(childKey: string): T {
      const composite = component as aether.Composite<S>;
      const found = composite.getChild(childKey);
      if (found == null)
        throw new UnexpectedError(
          `[aetherTest.mount] child ${childKey} not found on ${Component.TYPE}`,
        );
      return found as T;
    },
    providers: stack.providers,
    unmount(): void {
      stack.driveDelete([ROOT_KEY]);
    },
  };
};
