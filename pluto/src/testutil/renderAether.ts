// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { type z } from "zod";

import { type aether } from "@/aether/aether";
import { aetherTest } from "@/aether/test";
import { state } from "@/state";
import {
  buildStack,
  type MountedProviders,
  type ProviderOptions,
} from "@/testutil/providers";
import { type canvasTest } from "@/vis/render/test";

/** Initial state for a child mounted under the component under test. */
export interface MountChild {
  type: string;
  state: state.State;
}

/** Constructor signature for an aether component class accepted by {@link renderAether}.
 * The state schema is read for typing only (the Leaf parses incoming state itself), and
 * is accepted under either the `stateZ` or `z` static — both conventions exist in the
 * codebase. */
export type ComponentClass<S extends z.ZodType<state.State>> = {
  TYPE: string;
  new (props: aether.ComponentConstructorProps): aether.Component;
} & ({ stateZ: S } | { z: S });

/** Handle returned by {@link renderAether}. Drives state updates, inspects the
 * worker-side tree, and tears down the stack. */
export interface Handle<S extends z.ZodType<state.State>> {
  /** The component under test. */
  readonly component: aether.Component;
  /** Current parsed state of the component under test. */
  readonly state: z.infer<S>;
  /** Replace the component's state. Accepts a value or a `(prev) => next` setter; the
   * result is parsed against the component's schema. */
  setState(next: state.SetArg<z.infer<S>>): void;
  /** Create or replace a child of the component under test. `type` must be registered. */
  setChildState(key: string, type: string, childState: state.State): void;
  /** Delete a child of the component under test. */
  deleteChild(key: string): void;
  /** Look up a child instance by key. The `T` cast is unchecked; align it with the
   * registered class. */
  child<T extends aether.Component = aether.Component>(key: string): T;
  /** Provider instances in the stack, exposed for direct manipulation. */
  readonly providers: MountedProviders;
  /** The render recorder, if `render` was enabled; otherwise `null`. */
  readonly recorder: canvasTest.Recorder | null;
  /** Delete the entire tree and release references. */
  unmount(): void;
}

/** Options for {@link renderAether}. `state` is typed as the schema's input, so
 * fields with Zod defaults may be omitted. Provider toggles come from
 * {@link ProviderOptions}. */
export interface RenderAetherOptions<
  S extends z.ZodType<state.State>,
> extends ProviderOptions {
  state: z.input<S>;
  /** Instance key for the component under test. Defaults to the component's TYPE. */
  key?: string;
  /** Initial children to mount under the component under test. */
  children?: Record<string, MountChild>;
}

/**
 * Mount an aether component on the worker side, wrapped in a Synnax provider stack.
 *
 * Builds the chain `Root → [alamos → status → synnax → theming → telem → render] →
 * <component>`, including only the providers enabled in {@link ProviderOptions} (all on
 * by default except `render`). Every level runs its real `afterUpdate` lifecycle and
 * propagates context the way production does; the test never wires `parent` by hand.
 *
 * For tests that exercise the React + worker boundary, use `render` instead.
 */
export const renderAether = <S extends z.ZodType<state.State>>(
  Component: ComponentClass<S>,
  options: RenderAetherOptions<S>,
): Handle<S> => {
  const {
    state: initialState,
    key = Component.TYPE,
    children = {},
    ...providerOptions
  } = options;

  const stack = buildStack({
    ...providerOptions,
    registry: { ...providerOptions.registry, [Component.TYPE]: Component },
  });

  const componentPath = [...stack.basePath, key];
  stack.driver.update(componentPath, Component.TYPE, initialState as state.State);

  for (const [childKey, child] of Object.entries(children))
    stack.driver.update([...componentPath, childKey], child.type, child.state);

  const component = stack.driver.find(componentPath);

  return {
    component,
    get state(): z.infer<S> {
      return (component as aether.Leaf<S>).state;
    },
    setState(next: state.SetArg<z.infer<S>>): void {
      const leaf = component as aether.Leaf<S>;
      stack.driver.update(
        componentPath,
        Component.TYPE,
        state.executeSetter(next, leaf.state),
      );
    },
    setChildState(childKey, type, childState) {
      stack.driver.update([...componentPath, childKey], type, childState);
    },
    deleteChild(childKey) {
      stack.driver.delete([...componentPath, childKey]);
    },
    child<T extends aether.Component = aether.Component>(childKey: string): T {
      const composite = component as aether.Composite<S>;
      const found = composite.getChild(childKey);
      if (found == null)
        throw new UnexpectedError(
          `[renderAether] child ${childKey} not found on ${Component.TYPE}`,
        );
      return found as T;
    },
    providers: stack.providers,
    recorder: stack.recorder,
    unmount(): void {
      stack.driver.delete([aetherTest.ROOT_KEY]);
    },
  };
};
