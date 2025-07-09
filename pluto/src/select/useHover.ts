import { type record, TimeSpan } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";

import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { Triggers } from "@/triggers";

export interface UseHoverProps<K extends record.Key> {
  initialHover?: number;
  data: K[];
  onSelect: (key: K) => void;
}

const UP_TRIGGER: Triggers.Trigger = ["ArrowUp"];
const DOWN_TRIGGER: Triggers.Trigger = ["ArrowDown"];
const SELECT_TRIGGER: Triggers.Trigger = ["Enter"];
const TRIGGERS: Triggers.Trigger[] = [UP_TRIGGER, DOWN_TRIGGER, SELECT_TRIGGER];

const INITIAL_HOVER_DELAY = TimeSpan.milliseconds(200).milliseconds;
const HOVER_INTERVAL = TimeSpan.milliseconds(100).milliseconds;

export interface UseHoverReturn<K extends record.Key> {
  hover: K;
}

export const useHover = <K extends record.Key>({
  data,
  initialHover = -1,
  onSelect,
}: UseHoverProps<K>): UseHoverReturn<K> => {
  const dataRef = useSyncedRef(data);
  const [hover, setHover, hoverRef] = useCombinedStateAndRef<number>(initialHover);
  useEffect(() => {
    if (hover >= data.length) setHover(0);
  }, [data.length]);

  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  const handleTrigger = useCallback(
    ({ triggers, stage }: Triggers.UseEvent) => {
      if (intervalRef.current != null) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
      if (stage !== "start") return;

      const data = dataRef.current;
      if (Triggers.match(triggers, [SELECT_TRIGGER])) {
        if (hoverRef.current === -1) return;
        onSelect?.(data[hoverRef.current]);
        return;
      }
      const move = () => {
        const data = dataRef.current;
        if (Triggers.match(triggers, [UP_TRIGGER], { loose: true }))
          setHover((pos) => (pos <= 0 ? data.length - 1 : pos - 1));
        else if (Triggers.match(triggers, [DOWN_TRIGGER], { loose: true }))
          setHover((pos) => (pos >= data.length - 1 ? 0 : pos + 1));
      };
      move();
      intervalRef.current = setTimeout(() => {
        intervalRef.current = setInterval(move, HOVER_INTERVAL);
      }, INITIAL_HOVER_DELAY);
    },
    [onSelect],
  );

  Triggers.use({ triggers: TRIGGERS, callback: handleTrigger, loose: true });
  return { hover: data[hover] };
};
