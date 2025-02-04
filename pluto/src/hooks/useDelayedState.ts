// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";

/**
 * A custom hook that introduces a delay before setting a state to a final value.
 *
 * @param initial: The initial state value.
 * @param final: The final state value.
 * @param delay: The delay in milliseconds before setting the state to the final value.
 * The default delay is 150ms.
 * @returns The current state.
 */
export const useDelayedState = <T>(initial: T, final: T, delay = 150): T => {
  const [state, setState] = useState<T>(initial);
  useEffect(() => {
    const timeout = setTimeout(() => setState(final), delay);
    return () => clearTimeout(timeout);
  }, [final, delay]);
  return state;
};
