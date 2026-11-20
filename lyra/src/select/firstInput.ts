// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";

import { type Registry } from "@/select/registry";

const EVENTS = ["wheel", "touchmove", "pointerdown"] as const;
const KEYS: ReadonlySet<string> = new Set([
  "ArrowUp",
  "ArrowDown",
  "PageUp",
  "PageDown",
  "Home",
  "End",
]);

/**
 * Calls the listener once, at the user's first input that scrolls a list, moves its
 * cursor, or searches it: a wheel, a touch, a pointer press, a scroll key, or a search
 * term in the registry.
 * @returns a destructor that stops the wait.
 */
export const onFirstInput = (
  registry: Registry,
  listener: () => void,
): destructor.Destructor => {
  const controller = new AbortController();
  const stops: destructor.Destructor[] = [() => controller.abort()];
  const stop = (): void => stops.splice(0).forEach((s) => s());
  const handle = (): void => {
    stop();
    listener();
  };
  const options = { capture: true, passive: true, signal: controller.signal };
  EVENTS.forEach((type) => window.addEventListener(type, handle, options));
  window.addEventListener(
    "keydown",
    (e) => {
      if (KEYS.has(e.key)) handle();
    },
    options,
  );
  stops.push(
    registry.subscribe(() => {
      if (registry.getTerm() !== "") handle();
    }),
  );
  return stop;
};
