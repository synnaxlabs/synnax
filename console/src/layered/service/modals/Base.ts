// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { useStore } from "@/layered/session/modals/Provider";
import {
  type Renderer,
  type RenderProps,
  type Size,
} from "@/layered/session/modals/store";

/**
 * The static spec a modal hook carries so it can also be opened imperatively by
 * reference (via the placer returned from {@link use}). Read off the hook through the
 * {@link SPEC} symbol.
 */
export interface Spec<Args, Result> {
  render: Renderer<Args, Result>;
  size?: Size;
}

export const SPEC = Symbol("modals.spec");

/** A typed fire-and-forget opener: opens a modal and returns immediately. */
export interface Opener<Args> {
  (args: Args): void;
}

/** A typed prompt opener: opens a modal and resolves with its result, or null. */
export interface Prompt<Result, Args> {
  (args: Args): Promise<Result | null>;
}

/**
 * A fire-and-forget modal. Calling the hook returns an opener that pushes the modal and
 * returns immediately. The hook doubles as the modal's by-reference handle.
 */
export interface OpenHook<Args> {
  (): Opener<Args>;
  [SPEC]: Spec<Args, void>;
}

/**
 * A result-returning modal. Calling the hook returns an opener that resolves with the
 * value the renderer passes to close, or null on dismissal. The hook doubles as the
 * modal's by-reference handle.
 */
export interface PromptHook<Args, Result> {
  (): Prompt<Result, Args>;
  [SPEC]: Spec<Args, Result>;
}

export interface Options {
  /** The fixed maximum dimensions of the dialog. */
  size?: Size;
}

/**
 * renderer extracts the component a modal hook mounts, for rendering a modal body in
 * isolation (tests, previews) without going through the modal store.
 */
export const renderer = <Args, Result>(
  modal: OpenHook<Args> | PromptHook<Args, Result>,
): Renderer<Args, Result> => modal[SPEC].render as Renderer<Args, Result>;

/**
 * create defines a fire-and-forget modal (a form that performs its own side effects and
 * dismisses itself). It returns the open hook directly; the hook's function takes the
 * modal's typed args.
 */
export const create = <Args = void>(
  options: Options,
  Component: FC<RenderProps<Args, void>>,
): OpenHook<Args> => {
  const spec: Spec<Args, void> = { render: Component, size: options.size };
  const useOpen = (): ((args: Args) => void) => {
    const store = useStore();
    return useCallback(
      (args: Args) =>
        store.push({ key: id.create(), ...spec, args, resolve: () => {} }),
      [store],
    );
  };
  (useOpen as OpenHook<Args>)[SPEC] = spec;
  return useOpen as OpenHook<Args>;
};

/**
 * prompt defines a result-returning modal. It returns the prompt hook directly; the
 * hook's function takes the modal's typed args and resolves with the renderer's result
 * (or null on dismissal).
 */
export const prompt = <Result, Args = void>(
  options: Options,
  Component: FC<RenderProps<Args, Result>>,
): PromptHook<Args, Result> => {
  const spec: Spec<Args, Result> = { render: Component, size: options.size };
  const usePrompt = (): ((args: Args) => Promise<Result | null>) => {
    const store = useStore();
    return useCallback(
      (args: Args) =>
        new Promise<Result | null>((resolve) =>
          store.push({ key: id.create(), ...spec, args, resolve }),
        ),
      [store],
    );
  };
  (usePrompt as PromptHook<Args, Result>)[SPEC] = spec;
  return usePrompt as PromptHook<Args, Result>;
};
