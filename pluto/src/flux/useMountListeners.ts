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

import { type Subscriber } from "@/flux/aether/types";
import { useAddListener } from "@/flux/Context";

interface SynchronizerRef {
  mountCalled: boolean;
  destructor: Destructor;
}

export interface UseMountListenersProps {
  onOpen?: () => void;
  listeners?: Subscriber[];
}

export const useMountListeners = (): ((props: UseMountListenersProps) => void) => {
  const ref = useRef<SynchronizerRef>({
    mountCalled: false,
    destructor: () => {},
  });
  const addListener = useAddListener();
  useEffect(() => () => ref.current.destructor(), []);
  return useCallback(
    ({ listeners, onOpen }: UseMountListenersProps) => {
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
