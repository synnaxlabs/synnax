// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, type ReactElement, type MouseEventHandler } from "react";

import { Icon } from "@synnaxlabs/media";
import {
  Synnax,
  Dropdown,
  Button,
  Input,
  Align,
  Menu,
  componentRenderProp,
} from "@synnaxlabs/pluto";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { createWindowLayout } from "@/workspace/Create";
import { useSelectActive } from "@/workspace/selectors";
import { add, setActive } from "@/workspace/slice";

import "@/workspace/Selector.css";

export const Selector = (): ReactElement => {
  const client = Synnax.use();
  const d = useDispatch();
  const place = Layout.usePlacer();
  const active = useSelectActive();
  const dProps = Dropdown.use();
  const handleChange = useCallback(
    ([v]: string[]) => {
      dProps.close();
      if (v === null) return;
      void (async () => {
        if (v == null) {
          d(setActive(null));
          return;
        } else if (client == null) return;
        const ws = await client.workspaces.retrieve(v);
        d(add({ workspaces: [ws] }));
      })();
    },
    [active, client, d, dProps.close],
  );

  const ContextMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
    const handleSelect = (key: string): void => {
      switch (key) {
        case "delete":
          return;
        case "edit":
          place(createWindowLayout("Edit Workspace"));
      }
      if (key === "new") {
        place(createWindowLayout());
      }
    };
    return (
      <Menu.ContextMenuMenu keys={keys} onChange={handleSelect}>
        <Menu.ContextMenuItem itemKey="new">New Workspace</Menu.ContextMenuItem>
        <Menu.ContextMenuItem itemKey="delete">Delete Workspace</Menu.ContextMenuItem>
        <Menu.ContextMenuItem itemKey="edit">Edit Workspace</Menu.ContextMenuItem>
      </Menu.ContextMenuMenu>
    );
  };

  return (
    <Dropdown.Dialog
      {...dProps}
      bordered={false}
      variant="floating"
      className={CSS(CSS.BE("workspace", "selector"))}
      keepMounted={false}
    >
      <Button.Button
        startIcon={<Icon.Workspace key="workspace" />}
        endIcon={<Icon.Caret.Down key="down" />}
        variant="text"
        onClick={() => dProps.toggle()}
        size="medium"
        className={CSS.B("trigger")}
        shade={8}
        weight={400}
      >
        {active?.name ?? "No Workspace"}
      </Button.Button>
      <Align.Pack direction="y" style={{ width: 500 }}>
        <List.List>
          <List.Selector
            value={active == null ? [] : [active.key]}
            onChange={handleChange}
            allowMultiple={false}
          />
          <List.Search searcher={client?.workspaces}>
            {(p) => (
              <Input.Text
                size="large"
                placeholder={
                  <Text.WithIcon level="p" startIcon={<Icon.Search key="search" />}>
                    Search Workspaces
                  </Text.WithIcon>
                }
                {...p}
              >
                <Button.Button
                  startIcon={<Icon.Add />}
                  variant="outlined"
                  onClick={() => place(createWindowLayout())}
                  iconSpacing="small"
                >
                  New
                </Button.Button>
              </Input.Text>
            )}
          </List.Search>
          <List.Core>{componentRenderProp(SelectorListItem)}</List.Core>
        </List.List>
      </Align.Pack>
    </Dropdown.Dialog>
  );
};

export const SelectorListItem = ({
  onSelect,
  ...props
}: List.ItemProps): ReactElement => {
  const { entry } = props;
  const handleSelect: MouseEventHandler = (e): void => {
    e.stopPropagation();
    onSelect?.(entry.key);
  };
  return (
    <List.ItemFrame {...props} onClick={handleSelect}>
      <Text.Text level="p">{entry.name}</Text.Text>
    </List.ItemFrame>
  );
};
