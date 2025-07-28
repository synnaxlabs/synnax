// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Destructor } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";

import { Sync } from "@/flux/sync";

/**
 * Internal reference object for managing synchronizer lifecycle.
 * @internal
 */
interface SynchronizerRef {
  /** Whether the synchronizer is currently mounted */
  mountCalled: boolean;
  /** Function to clean up all active listeners */
  destructor: Destructor;
}

export interface UseMountSynchronizersProps {
  onOpen?: () => void;
  listeners?: Sync.Subscriber[];
}

/**
 * Hook that manages the lifecycle of real-time synchronizers for flux queries.
 *
 * This hook provides a function to mount synchronizers that listen to real-time
 * data changes from the server. It automatically handles cleanup when the component
 * unmounts and ensures that listeners are properly managed.
 *
 * @returns A function to mount synchronizers with the given listeners
 *
 * @example
 * ```typescript
 * const mountSynchronizers = useMountSynchronizers();
 *
 * // Mount listeners for real-time updates
 * const listeners = [
 *   {
 *     channel: "user_updates",
 *     handler: (frame) => {
 *       // Handle real-time user updates
 *       console.log("User updated:", frame.get("user_updates"));
 *     }
 *   }
 * ];
 *
 * mountSynchronizers(listeners);
 * ```
 */
export const useMountSynchronizers = (): ((
  props: UseMountSynchronizersProps,
) => void) => {
  const ref = useRef<SynchronizerRef>({
    mountCalled: false,
    destructor: () => {},
  });
  const addListener = Sync.useAddListener();
  // Clean up listeners when component unmounts
  useEffect(() => () => ref.current.destructor(), []);
  return useCallback(
    ({ listeners, onOpen }: UseMountSynchronizersProps) => {
      if (listeners == null || listeners.length === 0 || ref.current.mountCalled)
        return;
      ref.current.mountCalled = true;
      let openCount = 0;
      const handleOpen = () => {
        openCount++;
        if (openCount === listeners.length) onOpen?.();
      };
      const destructors = listeners.map(({ channel, handler }) =>
        addListener({ channel, handler, onOpen: handleOpen }),
      );
      ref.current.destructor = () => destructors.forEach((d) => d());
    },
    [addListener],
  );
};
