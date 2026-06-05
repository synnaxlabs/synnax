// Copyright 2026 Synnax Labs, Inc.
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
  type Flux,
  Icon,
  Input,
  List,
  Menu,
  Select,
  state,
  Synnax,
  Text,
  Workspace,
} from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";
import { useDispatch, useStore } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { type RootState } from "@/store";
import { ContextMenu } from "@/workspace/ContextMenu";
import { CREATE_LAYOUT } from "@/workspace/Create";
import { selectActiveName, useSelectActive } from "@/workspace/selectors";
import { maybeRename, setActive } from "@/workspace/slice";

const useRename = () => {
  const store = useStore<RootState>();
  return Workspace.useRename({
    beforeUpdate: useCallback(
      async ({ data, rollbacks }: Flux.BeforeUpdateParams<Workspace.RenameParams>) => {
        const { key, name } = data;
        const oldName = selectActiveName(store.getState());
        store.dispatch(maybeRename({ key, name }));
        rollbacks.push(() => {
          if (oldName != null) store.dispatch(maybeRename({ key, name: oldName }));
        });
        return data;
      },
      [store],
    ),
  });
};

const listItem = Component.renderProp(
  (props: List.ItemProps<workspace.Key>): ReactElement | null => {
    const { itemKey } = props;
    const ws = List.useItem<workspace.Key, workspace.Workspace>(itemKey);
    const onRename = useRename();
    const hasUpdatePermission = Access.useUpdateGranted(workspace.ontologyID(itemKey));
    if (ws == null) return null;
    return (
      <Select.ListItem {...props}>
        <Text.MaybeEditable
          id={`text-${ws.key}`}
          value={ws.name}
          onChange={
            hasUpdatePermission
              ? (name) => onRename.update({ key: ws.key, name })
              : undefined
          }
          allowDoubleClick={false}
        />
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
  const handleVisibleChange = useCallback(
    (next: state.SetArg<boolean>) => {
      const visible = state.executeSetter(next, dialogVisible);
      setDialogVisible(visible);
      // The list seeds from cached workspaces, so refetch on open to surface workspaces
      // created since the cache was last populated.
      if (!visible) return;
      setSearch("");
      retrieve({});
    },
    [dialogVisible, retrieve],
  );
  const hasCreatePermission = Access.useCreateGranted(workspace.TYPE_ONTOLOGY_ID);
  const hasRetrievePermission = Access.useRetrieveGranted(workspace.TYPE_ONTOLOGY_ID);
  const menuProps = Menu.useContextMenu();
  const contextMenu = useCallback(
    (props: Menu.ContextMenuMenuProps) => <ContextMenu {...props} getItem={getItem} />,
    [getItem],
  );
  const handleFetchMore = useCallback(() => {
    retrieve({});
  }, [retrieve]);
  if (!hasRetrievePermission) return null;
  return (
    <Dialog.Frame visible={dialogVisible} onVisibleChange={handleVisibleChange}>
      <Select.Frame
        data={data}
        value={active?.key}
        onChange={handleChange}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={handleFetchMore}
        allowNone
      >
        <Dialog.Trigger
          size="medium"
          className={CSS.B("trigger")}
          contrast={2}
          weight={400}
        >
          <Icon.Workspace key="workspace" />
          {active?.name ?? "No workspace"}
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
            {hasCreatePermission && (
              <Button.Button
                size="large"
                variant="outlined"
                onClick={() => {
                  setDialogVisible(false);
                  placeLayout(CREATE_LAYOUT);
                }}
                gap="small"
                tooltip="Create a new workspace"
                tooltipLocation={location.BOTTOM_CENTER}
                borderColor={6}
              >
                <Icon.Add />
                New
              </Button.Button>
            )}
          </Flex.Box>
          <Menu.ContextMenu menu={contextMenu} {...menuProps} />
          <List.Items bordered borderColor={6} grow onContextMenu={menuProps.open}>
            {listItem}
          </List.Items>
        </Dialog.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
