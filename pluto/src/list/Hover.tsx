// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Key, type Keyed } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from "react";

import { useCombinedStateAndRef } from "@/hooks";
import { useDataContext, useGetTransformedData } from "@/list/Data";
import { useSelectionUtils } from "@/list/Selector";
import { Triggers } from "@/triggers";

export interface HoverProps extends PropsWithChildren<{}> {
  disabled?: boolean;
  initialHover?: number;
}

const UP_TRIGGER: Triggers.Trigger = ["ArrowUp"];
const DOWN_TRIGGER: Triggers.Trigger = ["ArrowDown"];
const SELECT_TRIGGER: Triggers.Trigger = ["Enter"];
const TRIGGERS: Triggers.Trigger[] = [UP_TRIGGER, DOWN_TRIGGER, SELECT_TRIGGER];

export interface HoverContextValue {
  hover: number;
  setHover: (hover: number) => void;
}

const Context = createContext<HoverContextValue>({
  hover: -1,
  setHover: () => {},
});

export const useHoverContext = () => use(Context);

export const Hover = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  children,
  initialHover = -1,
  disabled = false,
}: HoverProps): ReactElement => {
  const getData = useGetTransformedData<K, E>();
  const { transformedData: data } = useDataContext<K, E>();
  const { onSelect } = useSelectionUtils();
  const [hover, setHover, hoverRef] = useCombinedStateAndRef<number>(initialHover);
  const beforeDisabledRef = useRef(initialHover);
  useEffect(() => {
    if (disabled) beforeDisabledRef.current = hover;
    setHover(disabled ? -1 : beforeDisabledRef.current);
  }, [disabled]);

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
      if (disabled) return;

      const data = getData();
      if (Triggers.match(triggers, [SELECT_TRIGGER])) {
        if (hoverRef.current === -1) return;
        onSelect?.(data[hoverRef.current].key);
        return;
      }
      const move = () => {
        const data = getData();
        if (Triggers.match(triggers, [UP_TRIGGER], { loose: true }))
          setHover((pos) => (pos <= 0 ? data.length - 1 : pos - 1));
        else if (Triggers.match(triggers, [DOWN_TRIGGER], { loose: true }))
          setHover((pos) => (pos >= data.length - 1 ? 0 : pos + 1));
      };
      move();
      intervalRef.current = setTimeout(() => {
        intervalRef.current = setInterval(move, 100);
      }, 200);
    },
    [onSelect, disabled],
  );

  Triggers.use({ triggers: TRIGGERS, callback: handleTrigger, loose: true });

  const ctxValue = useMemo<HoverContextValue>(
    () => ({ hover, setHover }),
    [hover, setHover],
  );

  return <Context value={ctxValue}>{children}</Context>;
};
