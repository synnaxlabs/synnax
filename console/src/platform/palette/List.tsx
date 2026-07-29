// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/palette/List.css";

import {
  type Component,
  Dialog,
  Input,
  List as Base,
  Select,
  Triggers,
} from "@synnaxlabs/pluto";
import { type record, type state } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback } from "react";

import { CSS } from "@/platform/css";

export interface BaseListProps<E extends record.Keyed<string>>
  extends
    Pick<Base.FrameProps<string, E>, "getItem" | "subscribe">,
    Input.Control<string>,
    Pick<Base.ItemsProps<string>, "emptyContent"> {
  data: string[];
  prefix?: string;
  listItem: Component.RenderProp<Base.ItemProps<string>>;
  retrieve: (query: state.SetArg<Base.PagerParams>) => void;
  inputPlaceholder: ReactElement;
}

export interface ListProps<E extends record.Keyed<string>> extends Pick<
  BaseListProps<E>,
  "inputPlaceholder" | "onChange" | "value"
> {}

export interface List<E extends record.Keyed<string>> extends FC<ListProps<E>> {}

export const SYNTHETIC_CLICK_DETAIL = 0;

export interface ListItemProps extends Select.ListItemProps<string> {}

export const ListItem = ({
  onSelect,
  itemKey,
  ...rest
}: Select.ListItemProps<string>) => {
  const handleClick = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      // Only trigger on the synthetic click, which means we won't accidentally call
      // `onSelect` twice.
      if (e.detail === SYNTHETIC_CLICK_DETAIL) onSelect?.(itemKey);
    },
    [onSelect, itemKey],
  );
  return (
    <Select.ListItem
      highlightHovered
      justify="between"
      align="center"
      onClick={handleClick}
      itemKey={itemKey}
      data-palette-key={itemKey}
      {...rest}
    />
  );
};

export const BaseList = <E extends record.Keyed<string>>({
  listItem,
  retrieve,
  prefix = "",
  inputPlaceholder,
  emptyContent,
  onChange,
  value,
  ...rest
}: BaseListProps<E>) => {
  const { fetchMore, search } = Base.usePager({ retrieve, pageSize: 20 });
  const { close } = Dialog.useContext();
  const handleSelect = useCallback((key: string) => {
    const element = document.querySelector(`[data-palette-key="${key}"]`);
    if (element == null || !(element instanceof HTMLElement)) return;
    element.click();
  }, []);

  const handleSearch = useCallback(
    (v: string) => {
      onChange(v);
      if (v.startsWith(prefix)) v = v.slice(prefix.length);
      search(v);
    },
    [search, onChange],
  );
  return (
    <Select.Frame<string, E>
      {...rest}
      onChange={handleSelect}
      onFetchMore={fetchMore}
      itemHeight={36}
      initialHover={0}
      closeDialogOnSelect
      virtual
    >
      <Input.Text
        className={CSS(CSS.BE("palette", "input"))}
        placeholder={inputPlaceholder}
        size="huge"
        autoFocus
        onChange={handleSearch}
        borderColor={8}
        value={value}
        autoComplete="off"
        onKeyDown={Triggers.matchCallback([["Escape"]], close)}
        full="x"
      />
      <Base.Items
        className={CSS.BE("palette", "list")}
        emptyContent={emptyContent}
        bordered
        borderColor={8}
        displayItems={10}
      >
        {listItem}
      </Base.Items>
    </Select.Frame>
  );
};
