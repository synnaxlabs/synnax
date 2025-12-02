// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/workspace/Selector.css";

import { UnexpectedError, workspace } from "@synnaxlabs/client";
import {
  Access,
  Button,
  Component,
  Dialog,
  Flex,
  Icon,
  Input,
  List,
  Select,
  Synnax,
  Text,
  Workspace,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { CREATE_LAYOUT } from "@/workspace/Create";
import { useSelectActive } from "@/workspace/selectors";
import { setActive } from "@/workspace/slice";

const listItem = Component.renderProp(
  (props: List.ItemProps<workspace.Key>): ReactElement | null => {
    const { itemKey } = props;
    const ws = List.useItem<workspace.Key, workspace.Workspace>(itemKey);
    if (ws == null) return null;
    return (
      <Select.ListItem {...props}>
        <Text.Text>{ws.name}</Text.Text>
      </Select.ListItem>
    );
  },
);

const DIALOG_STYLE = { minHeight: 200, minWidth: 400 };

export const Selector = (): ReactElement | null => {
  const client = Synnax.use();
  const dispatch = useDispatch();
  const active = useSelectActive();
  const placeLayout = Layout.usePlacer();
  const [dialogVisible, setDialogVisible] = useState(false);
  const { data, retrieve, getItem, subscribe } = Workspace.useList();
  const [search, setSearch] = useState("");
  const handleChange = useCallback(
    (v: string | null) => {
      if (v === null) {
        dispatch(setActive(null));
        dispatch(Layout.clearWorkspace());
        return;
      }
      const ws = getItem(v);
      if (ws == null) throw new UnexpectedError(`Workspace ${v} not found`);
      dispatch(setActive(ws));
      dispatch(
        Layout.setWorkspace({ slice: ws.layout as Layout.SliceState, keepNav: false }),
      );
      setDialogVisible(false);
    },
    [dispatch, getItem],
  );
  const allowCreateWorkspace = Access.useEditGranted(workspace.ontologyID(""));
  const allowViewWorkspace = Access.useViewGranted(workspace.ontologyID(""));
  if (!allowViewWorkspace) return null;
  return (
    <Dialog.Frame visible={dialogVisible} onVisibleChange={setDialogVisible}>
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
          size="medium"
          className={CSS.B("trigger")}
          contrast={2}
          weight={400}
        >
          <Icon.Workspace key="workspace" />
          {active?.name ?? "No Workspace"}
        </Dialog.Trigger>
        <Dialog.Dialog style={DIALOG_STYLE} bordered={client == null} borderColor={6}>
          <Flex.Box pack rounded>
            <Input.Text
              size="large"
              rounded
              placeholder={
                <>
                  <Icon.Search key="search" />
                  Search workspaces
                </>
              }
              contrast={0}
              value={search}
              onChange={(v) => {
                setSearch(v);
                retrieve((p) => ({ ...p, search: v }));
              }}
              full="x"
              style={{ borderBottomLeftRadius: 0 }}
              borderColor={6}
            />
            <Button.Button
              size="large"
              variant="outlined"
              onClick={() => {
                handleChange(null);
                setDialogVisible(false);
              }}
              gap="small"
              tooltip="Switch to no workspace"
              borderColor={6}
            >
              <Icon.Close />
              Clear
            </Button.Button>
            {allowCreateWorkspace && (
              <Button.Button
                size="large"
                variant="outlined"
                onClick={() => {
                  setDialogVisible(false);
                  placeLayout(CREATE_LAYOUT);
                }}
                gap="small"
                tooltip="Create a new workspace"
                tooltipLocation={{ y: "bottom" }}
                borderColor={6}
              >
                <Icon.Add />
                New
              </Button.Button>
            )}
          </Flex.Box>
          <List.Items bordered borderColor={6} grow>
            {listItem}
          </List.Items>
        </Dialog.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
