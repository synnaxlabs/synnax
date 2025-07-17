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
import {
  Button,
  Component,
  Dialog,
  Icon,
  Input,
  List,
  Select,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { CREATE_LAYOUT } from "@/workspace/Create";
import { useSelectActive } from "@/workspace/selectors";
import { add, setActive } from "@/workspace/slice";

export const selectorListItem = Component.renderProp(
  (props: List.ItemProps<workspace.Key>): ReactElement | null => {
    const { itemKey } = props;
    const ws = List.useItem<workspace.Key, workspace.Workspace>(itemKey);
    const selectProps = Select.useItemState(itemKey);
    if (ws == null) return null;
    return (
      <List.Item {...props} {...selectProps}>
        <Text.Text level="p">{ws.name}</Text.Text>
      </List.Item>
    );
  },
);

export const Selector = (): ReactElement => {
  const client = Synnax.use();
  const dispatch = useDispatch();
  const active = useSelectActive();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  const { data, retrieve, getItem, subscribe } = Workspace.useList();
  const [search, setSearch] = useState("");
  const handleChange = useCallback(
    (v: string | null) => {
      if (v === null) {
        dispatch(setActive(null));
        dispatch(Layout.clearWorkspace());
        return;
      }
      if (client == null) return;
      handleError(async () => {
        const ws = await client.workspaces.retrieve(v);
        dispatch(add(ws));
        dispatch(
          Layout.setWorkspace({
            slice: ws.layout as Layout.SliceState,
            keepNav: false,
          }),
        );
      }, "Failed to switch workspace");
    },
    [active, client, dispatch, close, handleError],
  );

  return (
    <Dialog.Frame>
      <Select.Frame
        data={data}
        value={active?.key}
        onChange={handleChange}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={() => retrieve({})}
        allowNone
      >
        <Dialog.Trigger
          startIcon={<Icon.Workspace key="workspace" />}
          variant="text"
          size="medium"
          className={CSS.B("trigger")}
          shade={2}
          weight={400}
        >
          {active?.name ?? "No Workspace"}
        </Dialog.Trigger>
        <Dialog.Dialog style={{ minHeight: 200, minWidth: 400 }}>
          <Cluster.NoneConnectedBoundary>
            <Input.Text
              size="large"
              placeholder={
                <Text.WithIcon level="p" startIcon={<Icon.Search key="search" />}>
                  Search Workspaces
                </Text.WithIcon>
              }
              shade={0}
              value={search}
              onChange={(v) => {
                setSearch(v);
                retrieve((p) => ({ ...p, search: v }));
              }}
            >
              <Button.Button
                startIcon={<Icon.Close />}
                size="large"
                variant="outlined"
                onClick={() => handleChange(null)}
                iconSpacing="small"
                tooltip="Switch to no workspace"
              >
                Clear
              </Button.Button>
              <Button.Button
                size="large"
                startIcon={<Icon.Add />}
                variant="outlined"
                onClick={() => {
                  close();
                  placeLayout(CREATE_LAYOUT);
                }}
                iconSpacing="small"
                tooltip="Create a new workspace"
                tooltipLocation={{ y: "bottom" }}
              >
                New
              </Button.Button>
            </Input.Text>
            <List.Items bordered borderShade={6} grow>
              {selectorListItem}
            </List.Items>
          </Cluster.NoneConnectedBoundary>
        </Dialog.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
