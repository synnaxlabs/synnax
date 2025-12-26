// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useRef } from "react";

import { useTick } from "@/perf/hooks/useTick";

/**
 * Timer hook that counts elapsed seconds while running.
 * Uses the shared tick context to ensure synchronized updates
 * with other timer-dependent components.
 *
 * Resets to 0 when `running` transitions from false to true.
 */
export const useTimer = (running: boolean): number => {
  const tick = useTick();
  const startTickRef = useRef<number | null>(null);

  // Handle transitions synchronously during render
  if (running && startTickRef.current === null) startTickRef.current = tick;
  else if (!running && startTickRef.current !== null) startTickRef.current = null;

  if (!running || startTickRef.current === null) return 0;
  return tick - startTickRef.current;
};
