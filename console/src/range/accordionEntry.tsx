// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { TimeSpan, TimeStamp } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { List as Core, Menu as PMenu, Divider, Synnax } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { Menu } from "@/components";
import { Layout } from "@/layout";
import { defineWindowLayout } from "@/range/Define";
import type { Range } from "@/range/range";
import { useSelect, useSelectMultiple } from "@/range/selectors";
import { remove, setActive } from "@/range/slice";

export const listColumns: Array<Core.ColumnSpec<string, Range>> = [
  {
    key: "name",
    name: "Name",
  },
  {
    key: "start",
    name: "Start",
    width: 100,
    stringer: (r) => {
      if (r.variant === "dynamic") return `${new TimeSpan(r.span).toString()} ago`;
      return new TimeStamp(r.timeRange.start).fString("dateTime", "local");
    },
  },
  {
    key: "end",
    name: "End",
    stringer: (r) => {
      if (r.variant === "dynamic") return "Now";
      const startTS = new TimeStamp(r.timeRange.start);
      const endTS = new TimeStamp(r.timeRange.end);
      return endTS.fString(
        endTS.span(startTS) < TimeSpan.DAY ? "time" : "dateTime",
        "local",
      );
    },
  },
];

export const List = (): ReactElement => {
  const menuProps = PMenu.useContextMenu();
  const newLayout = Layout.usePlacer();
  const dispatch = useDispatch();
  const ranges = useSelectMultiple();
  const selectedRange = useSelect();
  const client = Synnax.use();

  const handleAddOrEdit = (key?: string): void => {
    newLayout({
      ...defineWindowLayout,
      key: key ?? defineWindowLayout.key,
    });
  };

  const handleRemove = (keys: string[]): void => {
    dispatch(remove({ keys }));
  };

  const handleSelect = (key: string): void => {
    dispatch(setActive(key));
  };

  const handleSetActive = (key: string): void => {
    if (client == null) return;
    client.ranges.setActive(key).catch(console.error);
  };

  const ContextMenu = ({ keys }: PMenu.ContextMenuMenuProps): ReactElement => {
    const handleClick = (key: string): void => {
      switch (key) {
        case "create":
          return handleAddOrEdit();
        case "edit":
          return handleAddOrEdit(keys[0]);
        case "remove":
          return handleRemove(keys);
        case "setActive":
          return handleSetActive(keys[0]);
      }
    };
    return (
      <PMenu.Menu onChange={handleClick}>
        <PMenu.Item startIcon={<Icon.Edit />} size="small" itemKey="edit">
          Edit Range
        </PMenu.Item>
        <PMenu.Item startIcon={<Icon.Delete />} size="small" itemKey="remove">
          Remove Range
        </PMenu.Item>
        <PMenu.Item startIcon={<Icon.Add />} size="small" itemKey="create">
          Create Range
        </PMenu.Item>
        <PMenu.Item startIcon={<Icon.Play />} size="small" itemKey="setActive">
          Set Active
        </PMenu.Item>
        <Divider.Divider direction="x" padded />
        <Menu.Item.HardReload />
      </PMenu.Menu>
    );
  };

  return (
    <div style={{ flexGrow: 1 }}>
      <PMenu.ContextMenu menu={(props) => <ContextMenu {...props} />} {...menuProps}>
        <Core.List data={ranges.filter((r) => r.variant === "static")}>
          <Core.Selector
            value={selectedRange == null ? [] : [selectedRange.key]}
            onChange={([key]: string[]) => handleSelect(key)}
            allowMultiple={false}
          />
          <Core.Column.Header columns={listColumns} />
          <Core.Core.Virtual
            itemHeight={30}
            style={{ height: "100%", overflowX: "hidden" }}
          >
            {Core.Column.Item}
          </Core.Core.Virtual>
        </Core.List>
      </PMenu.ContextMenu>
    </div>
  );
};
