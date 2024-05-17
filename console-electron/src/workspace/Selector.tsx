// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, type ReactElement, type MouseEventHandler } from "react";

import { type workspace } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Synnax,
  Dropdown,
  Button,
  Input,
  Align,
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
    (v: string | null) => {
      dProps.close();
      if (v === null) {
        d(setActive(null));
        d(Layout.clearWorkspace());
        return;
      }
      void (async () => {
        if (v == null) {
          d(setActive(null));
          return;
        } else if (client == null) return;
        const ws = await client.workspaces.retrieve(v);
        d(add({ workspaces: [ws] }));
        console.log(ws.layout);
        d(
          Layout.setWorkspace({
            slice: ws.layout as unknown as Layout.SliceState,
            keepNav: false,
          }),
        );
      })();
    },
    [active, client, d, dProps.close],
  );

  return (
    <Dropdown.Dialog
      {...dProps}
      keepMounted={false}
      variant="floating"
      className={CSS(CSS.BE("workspace", "selector"))}
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
      <Align.Pack direction="y" style={{ width: 500, height: 200 }}>
        <List.List>
          <List.Selector
            value={active?.key ?? null}
            onChange={handleChange}
            allowMultiple={false}
            allowNone={true}
          >
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
                    startIcon={<Icon.Close />}
                    variant="outlined"
                    onClick={() => handleChange(null)}
                    iconSpacing="small"
                    tooltip="Switch to no workspace"
                  >
                    Clear
                  </Button.Button>
                  <Button.Button
                    startIcon={<Icon.Add />}
                    variant="outlined"
                    onClick={() => place(createWindowLayout())}
                    iconSpacing="small"
                    tooltip="Create a new workspace"
                    tooltipLocation={{ y: "bottom" }}
                  >
                    New
                  </Button.Button>
                </Input.Text>
              )}
            </List.Search>
            <List.Core>{componentRenderProp(SelectorListItem)}</List.Core>
          </List.Selector>
        </List.List>
      </Align.Pack>
    </Dropdown.Dialog>
  );
};

export const SelectorListItem = ({
  onSelect,
  ...props
}: List.ItemProps<string, workspace.Workspace>): ReactElement => {
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
