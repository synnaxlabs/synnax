// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use,
  useState,
  useSyncExternalStore,
} from "react";

import { Loop } from "@/schematic/node/common/custom/loop";

const REDUCED_MOTION = "(prefers-reduced-motion: reduce)";

const subscribe = (onChange: () => void): (() => void) => {
  const query = window.matchMedia(REDUCED_MOTION);
  query.addEventListener("change", onChange);
  return () => query.removeEventListener("change", onChange);
};

const isReduced = (): boolean => window.matchMedia(REDUCED_MOTION).matches;

interface MotionContextValue {
  loop: Loop;
  /** reduced is true under the operating system's Reduce motion setting. */
  reduced: boolean;
}

const Context = createContext<MotionContextValue | null>(null);
Context.displayName = "Custom.MotionContext";

/**
 * @returns the loop and Reduce motion setting of the nearest MotionProvider, or null
 * outside of one.
 */
export const useMotionContext = (): MotionContextValue | null => use(Context);

/**
 * MotionProvider moves the parts of every drawing inside it on one loop. Under the
 * operating system's Reduce motion setting, parts stop.
 */
export const MotionProvider = ({ children }: PropsWithChildren): ReactElement => {
  const [loop] = useState(() => new Loop());
  const reduced = useSyncExternalStore(subscribe, isReduced);
  return <Context value={{ loop, reduced }}>{children}</Context>;
};
