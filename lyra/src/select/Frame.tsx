// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type optional, type record } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useState,
  useSyncExternalStore,
} from "react";

import { Dialog } from "@/dialog";
import { List } from "@/list";
import { Context } from "@/select/Context";
import { type Registry, RegistryContext, useRegistry } from "@/select/registry";
import { KeyboardContext } from "@/select/scope";
import {
  useMultiple,
  type UseMultipleProps,
  useSingle,
  type UseSingleProps,
} from "@/select/use";

interface MultipleProviderProps<
  K extends record.Key = record.Key,
> extends PropsWithChildren<UseMultipleProps<K>> {}

const MultipleProvider = <K extends record.Key = record.Key>({
  children,
  ...rest
}: MultipleProviderProps<K>): ReactElement => {
  const { value, allowNone } = rest;
  const { keyboard, ...res } = useMultiple(rest);
  // Mirrors useMultiple: a click that would empty the selection returns before
  // onChange and before the dialog closes.
  const reselectNoop = allowNone === false;
  return (
    <Context value={value} reselectNoop={reselectNoop} {...res}>
      <KeyboardContext value={keyboard}>{children}</KeyboardContext>
    </Context>
  );
};

interface SingleProviderProps<
  K extends record.Key = record.Key,
> extends PropsWithChildren<UseSingleProps<K>> {}

const SingleProvider = <K extends record.Key = record.Key>({
  children,
  ...rest
}: SingleProviderProps<K>): ReactElement => {
  const { value, allowNone, closeDialogOnSelect } = rest;
  const { keyboard, ...res } = useSingle(rest);
  // Mirrors useSingle: a re-select closes the dialog when it closes on select, and
  // clears the selection otherwise when allowNone is set.
  const reselectNoop = allowNone !== true && closeDialogOnSelect !== true;
  return (
    <Context<K> value={value} reselectNoop={reselectNoop} {...res}>
      <KeyboardContext value={keyboard}>{children}</KeyboardContext>
    </Context>
  );
};

/** Props a custom {@link Frame} trigger receives. */
export interface TriggerProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> {
  value: K | null;
  /** Reads the entry for a key, subscribing to its changes. */
  useItem: (key: K) => E;
  /** Opens the dialog. */
  onClick: () => void;
}

interface BaseFrameProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> extends optional.Optional<Omit<List.FrameProps<K, E>, "onChange">, "data"> {}

export interface MultipleFrameProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
>
  extends BaseFrameProps<K, E>, UseMultipleProps<K> {
  multiple: true;
  /**
   * Whether to list the selected entries first each time the enclosing dialog opens.
   * The order holds while the dialog stays open, and a search lists its matches in
   * their own order.
   */
  selectedFirst?: boolean;
}

export interface SingleFrameProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>
  extends BaseFrameProps<K, E>, UseSingleProps<K> {
  multiple?: false;
}

/** Props for {@link Frame}. Set `multiple` to switch to a multi-entry selection. */
export type FrameProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
> = MultipleFrameProps<K, E> | SingleFrameProps<K, E>;

const EMPTY: never[] = [];

const NOOP_SUBSCRIBE = (): destructor.Destructor => () => {};

// Lists first the entries that were selected when the dialog opened. A null value
// turns the order off.
const useSelectedFirst = <K extends record.Key, E extends record.Keyed<K> | undefined>(
  registry: Registry,
  data: K[],
  getItem: List.GetItem<K, E> | undefined,
  value: K[] | null,
): K[] => {
  const { visible } = Dialog.useContext();
  const [opened, setOpened] = useState(false);
  const [front, setFront] = useState<K[]>(EMPTY);
  if (opened !== visible) {
    setOpened(visible);
    // Only a readable entry goes first. A fixed item keeps its own place, and a deleted
    // entry stays out of the list.
    const readable = (key: K): boolean =>
      getItem == null ? data.includes(key) : getItem(key) != null;
    setFront(visible && value != null ? value.filter(readable) : EMPTY);
  }
  const isSearching = (): boolean => registry.getTerm() !== "";
  const searching = useSyncExternalStore(
    value == null ? NOOP_SUBSCRIBE : registry.subscribe,
    isSearching,
    isSearching,
  );
  if (searching || front.length === 0) return data;
  const selected = new Set(front);
  return [...front, ...data.filter((k) => !selected.has(k))];
};

/**
 * Provides the options and the selection state to its children, with no UI of its own.
 * The options are its `data`, if any, and the {@link Item}s rendered outside an
 * {@link Items} block.
 */
export const Frame = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>(
  props: FrameProps<K, E>,
): ReactElement => {
  const {
    data = EMPTY,
    getItem,
    subscribe,
    itemHeight,
    multiple,
    onFetchMore,
    overscan,
    virtual = false,
    pinned,
    value,
    onChange,
    ...rest
  } = props;
  const registry = useRegistry();
  const ordered = useSelectedFirst(
    registry,
    data,
    getItem,
    props.multiple === true && props.selectedFirst === true ? props.value : null,
  );
  return (
    <List.Frame<K, E>
      data={ordered}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={onFetchMore}
      itemHeight={itemHeight}
      overscan={overscan}
      virtual={virtual}
      pinned={pinned}
    >
      <RegistryContext value={registry}>
        {multiple ? (
          <MultipleProvider value={value} onChange={onChange} {...rest} />
        ) : (
          <SingleProvider value={value} onChange={onChange} {...rest} />
        )}
      </RegistryContext>
    </List.Frame>
  );
};
Frame.displayName = "Select.Frame";
