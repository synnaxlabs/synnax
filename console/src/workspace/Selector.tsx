// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/workspace/Selector.css";

import { type workspace } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Caret,
  componentRenderProp,
  Dropdown,
  Input,
  List,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { type MouseEventHandler, type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { CREATE_LAYOUT } from "@/workspace/Create";
import { useSelectActive } from "@/workspace/selectors";
import { add, setActive } from "@/workspace/slice";

export const Selector = (): ReactElement => {
  const client = Synnax.use();
  const dispatch = useDispatch();
  const placeLayout = Layout.usePlacer();
  const active = useSelectActive();
  const dProps = Dropdown.use();
  const handleException = Status.useExceptionHandler();
  const handleChange = useCallback(
    (v: string | null) => {
      dProps.close();
      if (v === null) {
        dispatch(setActive(null));
        dispatch(Layout.clearWorkspace());
        return;
      }
      if (client == null) return;
      client.workspaces
        .retrieve(v)
        .then((ws) => {
          dispatch(add(ws));
          dispatch(
            Layout.setWorkspace({
              slice: ws.layout as Layout.SliceState,
              keepNav: false,
            }),
          );
        })
        .catch((e) => handleException(e, "Failed to switch workspace"));
    },
    [active, client, dispatch, dProps.close, handleException],
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
        endIcon={
          <Caret.Animated
            enabledLoc="bottom"
            disabledLoc="left"
            enabled={dProps.visible}
          />
        }
        variant="text"
        onClick={() => dProps.toggle()}
        size="medium"
        className={CSS.B("trigger")}
        shade={8}
        weight={400}
      >
        {active?.name ?? "No Workspace"}
      </Button.Button>
      <Align.Pack direction="y" borderShade={4} style={{ width: 500, height: 200 }}>
        <Cluster.NoneConnectedBoundary>
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
                      onClick={() => {
                        dProps.close();
                        placeLayout(CREATE_LAYOUT);
                      }}
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
        </Cluster.NoneConnectedBoundary>
      </Align.Pack>
    </Dropdown.Dialog>
  );
};

export const SelectorListItem = ({
  onSelect,
  ...rest
}: List.ItemProps<string, workspace.Workspace>): ReactElement => {
  const { entry } = rest;
  const handleSelect: MouseEventHandler = (e): void => {
    e.stopPropagation();
    onSelect?.(entry.key);
  };
  return (
    <List.ItemFrame {...rest} onClick={handleSelect}>
      <Text.Text level="p">{entry.name}</Text.Text>
    </List.ItemFrame>
  );
};
