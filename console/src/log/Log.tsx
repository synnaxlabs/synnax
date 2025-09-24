// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon, Log as Core, telem, usePrevious } from "@synnaxlabs/pluto";
import { deep, primitive, TimeSpan, uuid } from "@synnaxlabs/x";
import { useCallback, useEffect } from "react";

import { EmptyAction } from "@/components";
import { createLoadRemote } from "@/hooks/useLoadRemote";
import { Layout } from "@/layout";
import { select, useSelect, useSelectVersion } from "@/log/selectors";
import { internalCreate, setRemoteCreated, type State, ZERO_STATE } from "@/log/slice";
import { type Selector } from "@/selector";
import { Workspace } from "@/workspace";

export const LAYOUT_TYPE = "log";
export type LayoutType = typeof LAYOUT_TYPE;

export const useSyncComponent = Workspace.createSyncComponent(
  "Log",
  async ({ key, workspace, store, client }) => {
    const storeState = store.getState();
    const data = select(storeState, key);
    if (data == null) return;
    const layout = Layout.selectRequired(storeState, key);
    const setData = { ...data, key: undefined };
    if (!data.remoteCreated) store.dispatch(setRemoteCreated({ key }));
    await client.workspaces.logs.create(workspace, {
      key,
      name: layout.name,
      data: setData,
    });
  },
);

const DEFAULT_RETENTION = TimeSpan.days(1);
const PRELOAD = TimeSpan.seconds(30);

const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const winKey = useSelectWindowKey() as string;
  const log = useSelect(layoutKey);
  const dispatch = useSyncComponent(layoutKey);

  const { name } = Layout.useSelectRequired(layoutKey);

  const prevName = usePrevious(name);
  useEffect(() => {
    if (prevName !== name) dispatch(Layout.rename({ key: layoutKey, name }));
  }, [name, prevName, layoutKey]);

  let t: telem.SeriesSourceSpec;
  const ch = log.channels[0];
  const zeroChannel = primitive.isZero(ch);
  if (zeroChannel) t = telem.noopSeriesSourceSpec;
  else
    t = telem.streamChannelData({
      channel: ch,
      timeSpan: PRELOAD,
      keepFor: DEFAULT_RETENTION,
    });
  const handleDoubleClick = useCallback(() => {
    dispatch(
      Layout.setNavDrawerVisible({
        windowKey: winKey,
        key: "visualization",
        value: true,
      }),
    );
  }, [winKey, dispatch]);

  return (
    <Core.Log
      telem={t}
      onDoubleClick={handleDoubleClick}
      emptyContent={
        <EmptyAction
          message={
            zeroChannel
              ? "No channel configured for this log."
              : "No data received yet."
          }
          action={zeroChannel ? "Configure channel" : ""}
          onClick={handleDoubleClick}
        />
      }
      visible={visible}
    />
  );
};

const useLoadRemote = createLoadRemote<log.Log>({
  useRetrieve: Core.useRetrieveObservable,
  targetVersion: ZERO_STATE.version,
  useSelectVersion,
  actionCreator: (v) => internalCreate({ ...(v.data as State), key: v.key }),
});

export const Log: Layout.Renderer = ({ layoutKey, ...rest }) => {
  const log = useLoadRemote(layoutKey);
  if (log == null) return null;
  return <Loaded layoutKey={layoutKey} {...rest} />;
};

export const SELECTABLE: Selector.Selectable = {
  key: LAYOUT_TYPE,
  title: "Log",
  icon: <Icon.Log />,
  create: async ({ layoutKey }) => create({ key: layoutKey }),
};

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
