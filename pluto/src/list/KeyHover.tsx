import { useCallback, useRef } from "react";

import { Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { useContext } from "@/list/Context";
import { Trigger, Triggers, UseTriggerEvent } from "@/triggers";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface HoverProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> {}

const UP_TRIGGER: Trigger = ["ArrowUp"];
const DOWN_TRIGGER: Trigger = ["ArrowDown"];
const SELECT_TRIGGER: Trigger = ["Enter"];

export const Hover = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>(
  props: HoverProps<K, E>
): null => {
  const {
    data,
    select: { onSelect },
    hover: { value, onChange },
  } = useContext<K, E>();
  const posRef = useRef<number>(value);

  const handleTrigger = useCallback(
    ({ triggers, stage }: UseTriggerEvent) => {
      if (stage !== "start") return;
      if (Triggers.match(triggers, [UP_TRIGGER]))
        onChange((pos) => {
          const v = pos === 0 ? data.length - 1 : pos - 1;
          posRef.current = v;
          return v;
        });
      else if (Triggers.match(triggers, [DOWN_TRIGGER]))
        onChange((pos) => {
          const v = pos === data.length - 1 ? 0 : pos + 1;
          posRef.current = v;
          return v;
        });
      else if (Triggers.match(triggers, [SELECT_TRIGGER]))
        onSelect?.(data[posRef.current].key);
    },
    [data, onSelect]
  );

  Triggers.use({
    triggers: [UP_TRIGGER, DOWN_TRIGGER, SELECT_TRIGGER],
    callback: handleTrigger,
    loose: true,
  });

  return null;
};
