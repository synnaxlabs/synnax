// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type record } from "@synnaxlabs/x";
import {
  createContext,
  memo,
  type PropsWithChildren,
  type ReactElement,
  use,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from "react";

import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { useData } from "@/list/Data";
import { use, type UseSelectProps } from "@/select/use";
import { Triggers } from "@/triggers";

const UP_TRIGGER: Triggers.Trigger = ["ArrowUp"];
const DOWN_TRIGGER: Triggers.Trigger = ["ArrowDown"];
const SELECT_TRIGGER: Triggers.Trigger = ["Enter"];
const TRIGGERS: Triggers.Trigger[] = [UP_TRIGGER, DOWN_TRIGGER, SELECT_TRIGGER];

interface SelectContextValue<K extends record.Key = record.Key> {
  selected: K[];
  hover: number;
}

interface SelectUtilsContextValue<K extends record.Key = record.Key> {
  onSelect: (key: K) => void;
  clear: () => void;
  getSelected: () => K[];
  setHover: (hover: number) => void;
}

export type SelectorProps<K extends record.Key = record.Key> = PropsWithChildren<
  UseSelectProps<K>
> & {
  initialHover?: number;
  disableHover?: boolean;
};

const Context = createContext<SelectContextValue>({ selected: [], hover: -1 });

const UtilsContext = createContext<SelectUtilsContextValue>({
  onSelect: () => {},
  clear: () => {},
  getSelected: () => [],
  setHover: () => {},
});

export const useSelectionContext = <K extends record.Key = record.Key>() =>
  use(Context) as SelectContextValue<K>;

export const useSelection = <K extends record.Key = record.Key>() =>
  useSelectionContext<K>().selected;

export const useSelectionUtils = <K extends record.Key = record.Key>() =>
  use(UtilsContext) as unknown as SelectUtilsContextValue<K>;

/**
 * Implements selection behavior for a list.
 *
 * @param props - The props for the List.Selector component. These props are identical
 * to the props for {@link use} hook.
 */
const Base = memo(
  <K extends record.Key = record.Key>({
    value,
    children,
    initialHover = -1,
    disableHover = false,
    ...rest
  }: SelectorProps<K>): ReactElement => {
    const { items } = useData<K>();
    const { onSelect, clear } = use<K>({
      ...rest,
      value,
      data: items,
    } as const as UseSelectProps<K>);
    const selectedRef = useSyncedRef(value);

    const [hover, setHover, hoverRef] = useCombinedStateAndRef<number>(initialHover);
    const beforeDisabledRef = useRef(initialHover);

    useEffect(() => {
      if (disableHover) beforeDisabledRef.current = hover;
      setHover(disableHover ? -1 : beforeDisabledRef.current);
    }, [disableHover, items.length]);

    useEffect(() => {
      if (hover >= items.length) setHover(0);
    }, [items.length]);

    const intervalRef = useRef<NodeJS.Timeout | null>(null);

    const handleTrigger = useCallback(
      ({ triggers, stage }: Triggers.UseEvent) => {
        if (intervalRef.current != null) {
          clearInterval(intervalRef.current);
          intervalRef.current = null;
        }
        if (stage !== "start" || disableHover) return;

        if (Triggers.match(triggers, [SELECT_TRIGGER])) {
          if (hoverRef.current === -1) return;
          onSelect?.(items[hoverRef.current]);
          return;
        }
        const move = () => {
          if (Triggers.match(triggers, [UP_TRIGGER], { loose: true }))
            setHover((pos) => (pos <= 0 ? items.length - 1 : pos - 1));
          else if (Triggers.match(triggers, [DOWN_TRIGGER], { loose: true }))
            setHover((pos) => (pos >= items.length - 1 ? 0 : pos + 1));
        };
        move();
        intervalRef.current = setTimeout(() => {
          intervalRef.current = setInterval(move, 100);
        }, 200);
      },
      [onSelect, disableHover],
    );

    Triggers.use({ triggers: TRIGGERS, callback: handleTrigger, loose: true });
    const ctxValue: SelectContextValue<K> = useMemo(
      () => ({ selected: array.toArray(value), hover }),
      [value],
    );
    const utilCtxValue: SelectUtilsContextValue<K> = useMemo(
      () => ({
        onSelect,
        clear,
        getSelected: () => array.toArray(selectedRef.current),
        setHover,
      }),
      [onSelect, clear, setHover],
    );
    return (
      <UtilsContext value={utilCtxValue as unknown as SelectUtilsContextValue}>
        <Context value={ctxValue}>{children}</Context>
      </UtilsContext>
    );
  },
);
Base.displayName = "List.Selector";

export const Selector = Base as <K extends record.Key = record.Key>(
  props: SelectorProps<K>,
) => ReactElement;
