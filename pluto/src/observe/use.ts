import { compare, observe, Optional, Primitive } from "@synnaxlabs/x";
import { useState as reactUseState } from "react";

import { useAsyncEffect } from "@/hooks";
import { useMemoCompare } from "@/memo";

interface UseProps<D> {
  key: Primitive[];
  open?: () => Promise<observe.ObservableAsyncCloseable<D> | undefined>;
  onChange: observe.Handler<D>;
}

export const useListener = <D>({ key, open, onChange }: UseProps<D>) => {
  const memoKey = useMemoCompare(
    () => key,
    ([prev], [next]) => compare.unorderedPrimitiveArrays(prev, next) === compare.EQUAL,
    [key],
  );
  useAsyncEffect(async () => {
    if (open == null) return;
    const obs = await open();
    if (obs == null) return;
    obs.onChange(onChange);
    return () => obs.close();
  }, [open == null, memoKey]);
};

type UseStateProps<D> = Optional<UseProps<D>, "onChange">;

export interface UseStatePropsWithInitial<D> extends UseStateProps<D> {
  initialValue: D;
}

interface UseState {
  <D>(props: UseStatePropsWithInitial<D>): D;
  <D>(props: UseStateProps<D>): D | undefined;
}

export const useState = (<D>(props: UseStatePropsWithInitial<D>) => {
  const [v, setV] = reactUseState<D | undefined>(props.initialValue as D);
  useListener({ ...props, onChange: setV });
  return v;
}) as UseState;
