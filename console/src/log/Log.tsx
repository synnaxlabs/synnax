// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Access, Icon, Log as Base } from "@synnaxlabs/pluto";
import { deep, primitive, TimeSpan, uuid } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu, EmptyAction } from "@/components";
import { createLoadRemote } from "@/hooks/useLoadRemote";
import { Layout } from "@/layout";
import {
  select,
  useSelect,
  useSelectIsRemoteCreated,
  useSelectVersion,
} from "@/log/selectors";
import {
  internalCreate,
  setActiveToolbarTab,
  setRemoteCreated,
  type State,
  stateFromLog,
  ZERO_STATE,
} from "@/log/slice";
import { Project } from "@/project";
import { Selector } from "@/selector";

export const LAYOUT_TYPE = "log";
export type LayoutType = typeof LAYOUT_TYPE;

export const useSyncComponent = Project.createSyncComponent(
  "Log",
  async ({ key, project, store, fluxStore, client }) => {
    const storeState = store.getState();
    if (!Access.updateGranted({ id: log.ontologyID(key), store: fluxStore, client }))
      return;
    const data = select(storeState, key);
    if (data == null) return;
    const layout = Layout.selectRequired(storeState, key);
    if (!data.remoteCreated) store.dispatch(setRemoteCreated({ key }));
    await client.logs.create(project, {
      key,
      name: layout.name,
      channels: data.channels,
      remoteCreated: data.remoteCreated,
      timestampPrecision: data.timestampPrecision,
      showChannelNames: data.showChannelNames,
      showReceiptTimestamp: data.showReceiptTimestamp,
    });
  },
);

const DEFAULT_RETENTION = TimeSpan.days(1);
const PRELOAD = TimeSpan.seconds(30);
const EXTRA_CONTEXT_MENU_ITEMS = <ContextMenu.ReloadConsoleItem />;

const Loaded: Layout.Renderer = ({ layoutKey, visible, active }) => {
  const log = useSelect(layoutKey);
  const dispatch = useSyncComponent(layoutKey);

  const enableTriggers = useCallback(() => active, [active]);

  const activeChannels = log.channels.filter((e) => !primitive.isZero(e.channel));
  const hasChannels = activeChannels.length > 0;
  // Stable spec — channels are passed separately via the `channels` prop so that
  // adding/removing a channel does not destroy and recreate the telem source.
  const t = Base.streamMultiChannelLog({
    channels: [],
    timeSpan: PRELOAD,
    keepFor: DEFAULT_RETENTION,
  });
  const handleDoubleClick = useCallback(() => {
    dispatch(Layout.setNavDrawerVisible({ key: "visualization", value: true }));
  }, [dispatch]);

  const handleConfigureChannels = useCallback(() => {
    dispatch(setActiveToolbarTab({ key: layoutKey, tab: "channels" }));
    handleDoubleClick();
  }, [dispatch, layoutKey, handleDoubleClick]);

  return (
    <Base.Log
      telem={t}
      channels={activeChannels}
      showChannelNames={log.showChannelNames}
      showReceiptTimestamp={log.showReceiptTimestamp}
      timestampPrecision={log.timestampPrecision}
      onDoubleClick={handleDoubleClick}
      enableTriggers={enableTriggers}
      extraContextMenuItems={EXTRA_CONTEXT_MENU_ITEMS}
      emptyContent={
        <EmptyAction
          message={
            !hasChannels
              ? "No channels configured for this log."
              : "No data received yet."
          }
          action={!hasChannels ? "Configure channels" : ""}
          onClick={hasChannels ? handleDoubleClick : handleConfigureChannels}
        />
      }
      visible={visible}
    />
  );
};

const useLoadRemote = createLoadRemote<log.Log>({
  useRetrieve: Base.useRetrieveObservable,
  targetVersion: ZERO_STATE.version,
  useSelectVersion,
  actionCreator: (v) => internalCreate(stateFromLog(v)),
});

export const Log: Layout.Renderer = ({ layoutKey, ...rest }) => {
  const log = useLoadRemote(layoutKey);
  if (log == null) return null;
  return <Loaded layoutKey={layoutKey} {...rest} />;
};

Log.useName = Layout.createUseFluxName(
  Base.useRename,
  Base.useRetrieveObservableName,
  useSelectIsRemoteCreated,
);

export const Selectable: Selector.Selectable = ({ layoutKey, onPlace, onResolved }) => {
  const hasCreatePermission = Access.useCreateGranted(log.TYPE_ONTOLOGY_ID);
  const dispatch = useDispatch();
  const handleClick = useCallback(() => {
    // In panel mode create the local log state (so it renders and later syncs to
    // the server on edit) and hand its ontology.ID back; otherwise open it as a
    // mosaic tab as before.
    if (onResolved != null) {
      dispatch(internalCreate({ ...deep.copy(ZERO_STATE), key: layoutKey }));
      onResolved(log.ontologyID(layoutKey));
    } else onPlace(create({ key: layoutKey }));
  }, [onResolved, onPlace, layoutKey, dispatch]);

  if (!hasCreatePermission) return null;

  return (
    <Selector.Item
      key={LAYOUT_TYPE}
      title="Log"
      icon={<Icon.Log />}
      onClick={handleClick}
    />
  );
};
Selectable.type = LAYOUT_TYPE;
Selectable.useVisible = () => Access.useCreateGranted(log.TYPE_ONTOLOGY_ID);

export type CreateArg = Partial<State> & Omit<Partial<Layout.BaseState>, "type">;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Log", location = "mosaic", window, tab, ...rest } = initial;
    const key = log.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key }));
    return {
      key,
      name,
      icon: "Log",
      location,
      type: LAYOUT_TYPE,
      windowKey: key,
      window,
      tab,
    };
  };
