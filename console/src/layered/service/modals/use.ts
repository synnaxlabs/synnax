// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { useMemo } from "react";

import { type OpenHook, type PromptHook, SPEC } from "@/layered/service/modals/Base";
import { useStore } from "@/layered/session/modals/Provider";

/**
 * The args parameter of a placer call: omitted when the modal takes no args, optional
 * when every field is optional, and required otherwise.
 */
type ArgsParam<Args> = [Args] extends [void]
  ? []
  : Partial<Args> extends Args
    ? [args?: Args]
    : [args: Args];

/**
 * An imperative modal opener, used by non-React callers (ontology services, deep-link
 * handlers) that cannot call a modal's hook directly. Modals are opened by reference —
 * passing the hook itself — and the open/prompt split mirrors the modal's kind, so a
 * fire-and-forget modal cannot be prompted and vice versa.
 */
export interface Placer {
  /** open opens a fire-and-forget modal by reference. */
  open: <Args>(modal: OpenHook<Args>, ...args: ArgsParam<Args>) => void;
  /** prompt opens a result-returning modal by reference and resolves with its result. */
  prompt: <Args, Result>(
    modal: PromptHook<Args, Result>,
    ...args: ArgsParam<Args>
  ) => Promise<Result | null>;
}

/** use returns an imperative {@link Placer} bound to the active modal store. */
export const use = (): Placer => {
  const store = useStore();
  return useMemo<Placer>(() => {
    const open = <Args>(modal: OpenHook<Args>, ...args: ArgsParam<Args>): void =>
      store.push({
        key: id.create(),
        ...modal[SPEC],
        args: args[0] as Args,
        resolve: () => {},
      });
    const prompt = <Args, Result>(
      modal: PromptHook<Args, Result>,
      ...args: ArgsParam<Args>
    ): Promise<Result | null> =>
      new Promise<Result | null>((resolve) =>
        store.push({ key: id.create(), ...modal[SPEC], args: args[0] as Args, resolve }),
      );
    return { open, prompt };
  }, [store]);
};
