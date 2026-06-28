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
import { type RenderProps } from "@/layered/session/modals/store";

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
 * returns immediately.
 */
export interface OpenHook<Args> {
  (): Opener<Args>;
}

/**
 * A result-returning modal. Calling the hook returns an opener that resolves with the
 * value the renderer passes to close, or null on dismissal.
 */
export interface PromptHook<Args, Result> {
  (): Prompt<Result, Args>;
}

/**
 * create defines a fire-and-forget modal (a form that performs its own side effects and
 * dismisses itself). It returns the open hook directly; the hook's function takes the
 * modal's typed args.
 */
export const create = <Args = void>(
  Component: FC<RenderProps<Args, void>>,
): OpenHook<Args> => {
  const useOpen = (): ((args: Args) => void) => {
    const store = useStore();
    return useCallback(
      (args: Args) =>
        store.push({ key: id.create(), render: Component, args, resolve: () => {} }),
      [store],
    );
  };
  return useOpen;
};

/**
 * prompt defines a result-returning modal. It returns the prompt hook directly; the
 * hook's function takes the modal's typed args and resolves with the renderer's result
 * (or null on dismissal).
 */
export const prompt = <Result, Args = void>(
  Component: FC<RenderProps<Args, Result>>,
): PromptHook<Args, Result> => {
  const usePrompt = (): ((args: Args) => Promise<Result | null>) => {
    const store = useStore();
    return useCallback(
      (args: Args) =>
        new Promise<Result | null>((resolve) =>
          store.push({ key: id.create(), render: Component, args, resolve }),
        ),
      [store],
    );
  };
  return usePrompt;
};
