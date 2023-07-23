import { useCallback, useMemo, useRef } from "react";

import { Bounds, Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { useListContext } from "./ListContext";

import { Trigger, Triggers, UseTriggerEvent } from "@/core/triggers";

export interface ListHoverProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> {}

const UPTRIGGER: Trigger = ["ArrowUp"];
const DOWNTRIGGER: Trigger = ["ArrowDown"];
const ENTERTRIGGER: Trigger = ["Enter"];

export const ListHover = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>(
  props: ListHoverProps<K, E>
): null => {
  const {
    data,
    select: { onSelect },
    hover: { value, onChange },
  } = useListContext<K, E>();
  const posRef = useRef<number>(value);

  const handleTrigger = useCallback(
    ({ triggers, stage }: UseTriggerEvent) => {
      if (stage !== "start") return;
      if (Triggers.match(triggers, [UPTRIGGER]))
        onChange((pos) => {
          const v = pos === 0 ? data.length - 1 : pos - 1;
          posRef.current = v;
          return v;
        });
      else if (Triggers.match(triggers, [DOWNTRIGGER]))
        onChange((pos) => {
          const v = pos === data.length - 1 ? 0 : pos + 1;
          posRef.current = v;
          return v;
        });
      else if (Triggers.match(triggers, [ENTERTRIGGER]))
        onSelect?.(data[posRef.current].key);
    },
    [data, onSelect]
  );

  Triggers.use({
    triggers: [UPTRIGGER, DOWNTRIGGER, ENTERTRIGGER],
    callback: handleTrigger,
    loose: true,
  });

  return null;
};
