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
  List,
  Input,
  Align,
  componentRenderProp,
} from "@synnaxlabs/pluto";
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
  const p = Layout.usePlacer();
  const active = useSelectActive();
  const dProps = Dropdown.use();
  const handleChange = useCallback(
    ([v]: string[]) => {
      dProps.close();
      void (async () => {
        if (v == null) {
          d(setActive(null));
          return;
        } else if (client == null) return;
        const ws = await client.workspaces.retrieve(v);
        d(add({ workspaces: [ws] }));
        d(
          Layout.setWorkspace({
            slice: ws.layout as unknown as Layout.SliceState,
            keepNav: false,
          })
        );
      })();
    },
    [client, d, dProps.close]
  );

  return (
    <Dropdown.Dialog
      {...dProps}
      bordered={false}
      matchTriggerWidth
      className={CSS(CSS.BE("workspace", "selector"))}
    >
      <Button.Button
        startIcon={<Icon.Workspace />}
        endIcon={<Icon.Caret.Down />}
        variant={dProps.visible ? "outlined" : "text"}
        onClick={() => dProps.toggle()}
        size="small"
        className={CSS.B("trigger")}
      >
        {active?.name ?? "No Workspace"}
      </Button.Button>
      <Align.Pack direction="y">
        <List.List>
          <List.Selector
            value={active == null ? [] : [active.key]}
            onChange={handleChange}
            allowMultiple={false}
            allowNone={false}
          />
          <Align.Pack direction="x">
            <List.Search searcher={client?.workspaces}>
              {(props) => <Input.Text {...props} />}
            </List.Search>
            <Button.Icon
              onClick={() => {
                d(setActive(null));
                dProps.close();
              }}
            >
              <Icon.Close />
            </Button.Icon>
            <Button.Icon
              onClick={() => p(createWindowLayout)}
              style={{ borderRadius: 0 }}
            >
              <Icon.Add />
            </Button.Icon>
          </Align.Pack>
          <List.Core>{componentRenderProp(SelectorListItem)}</List.Core>
        </List.List>
      </Align.Pack>
    </Dropdown.Dialog>
  );
};

export const SelectorListItem = ({
  entry: { key, name },
  hovered,
  onSelect,
  selected,
  ...props
}: List.ItemProps): ReactElement => {
  const handleSelect: MouseEventHandler = (e): void => {
    e.stopPropagation();
    onSelect?.(key);
  };
  return (
    <Button.Button
      onClick={handleSelect}
      variant="text"
      className={CSS(
        CSS.BE("palette", "item"),
        hovered && CSS.BEM("palette", "item", "hovered"),
        CSS.BEM("palette", "item", "command")
      )}
      sharp
      {...props}
    >
      {name}
    </Button.Button>
  );
};
