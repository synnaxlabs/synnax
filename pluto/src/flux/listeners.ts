import { type Destructor, type MultiSeries, type record } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";

import { Sync } from "@/flux/sync";
import { Synnax as PSynnax } from "@/synnax";

interface ListenerRef {
  mounted: boolean;
  destructor: Destructor;
}

export interface UseListenersArgs {
  listeners: () => Sync.Subscriber[];
}

export const useListeners = ({ listeners }: UseListenersArgs): (() => void) => {
  const ref = useRef<ListenerRef>({
    mounted: false,
    destructor: () => {},
  });
  const addListener = Sync.useAddListener();
  const client = PSynnax.use();

  useEffect(() => () => ref.current.destructor(), []);

  return useCallback(() => {
    if (listeners == null || listeners.length === 0 || client == null) return;
    ref.current.mounted = true;
    const destructors = listeners().map(({ channel, handler }) =>
      addListener({ channel, handler }),
    );
    ref.current.destructor = () => destructors.forEach((d) => d());
  }, [addListener, client]);
};
