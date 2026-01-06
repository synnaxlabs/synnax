// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useInterval } from "@synnaxlabs/pluto";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useContext,
  useState,
} from "react";

const TickContext = createContext<number>(0);
TickContext.displayName = "TickContext";

/**
 * Provider that emits a global tick every second.
 * All components using useTick() will update on the same render cycle,
 * ensuring synchronized timer displays.
 */
export const TickProvider = ({ children }: PropsWithChildren): ReactElement => {
  const [tick, setTick] = useState(0);
  useInterval(() => setTick((t) => t + 1), 1000);
  return <TickContext value={tick}>{children}</TickContext>;
};

/**
 * Returns the current global tick count.
 * Use this to synchronize timer-dependent components.
 */
export const useTick = (): number => useContext(TickContext);
