import { Dispatch, PayloadAction } from "@reduxjs/toolkit";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon } from "@synnaxlabs/media";
import { Log as Core, telem, usePrevious } from "@synnaxlabs/pluto";
import { deep, primitiveIsZero, TimeSpan, UnknownRecord } from "@synnaxlabs/x";
import { ReactElement, useCallback, useEffect } from "react";
import { v4 as uuidv4 } from "uuid";

import { useLoadRemote } from "@/hooks/useLoadRemote";
import { Layout } from "@/layout";
import { select, useSelect } from "@/log/selectors";
import { internalCreate, setRemoteCreated, State, ZERO_STATE } from "@/log/slice";
import { Workspace } from "@/workspace";

export type LayoutType = "log";
export const LAYOUT_TYPE = "log";

interface SyncPayload {
  key?: string;
}

const useSyncComponent = (layoutKey: string): Dispatch<PayloadAction<SyncPayload>> =>
  Workspace.useSyncComponent<SyncPayload>(
    "Log",
    layoutKey,
    async (ws, store, client) => {
      const storeState = store.getState();
      const data = select(storeState, layoutKey);
      if (data == null) return;
      const layout = Layout.selectRequired(storeState, layoutKey);
      const setData = {
        ...data,
        key: undefined,
        snapshot: undefined,
      } as unknown as UnknownRecord;
      if (!data.remoteCreated) store.dispatch(setRemoteCreated({ key: layoutKey }));
      await new Promise((r) => setTimeout(r, 1000));
      await client.workspaces.log.create(ws, {
        key: layoutKey,
        name: layout.name,
        data: setData,
      });
    },
  );

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
  if (primitiveIsZero(ch)) t = telem.noopSeriesSourceSpec;
  else
    t = telem.streamChannelData({
      channel: ch,
      timeSpan: TimeSpan.seconds(log.retention),
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

  return <Core.Log telem={t} onDoubleClick={handleDoubleClick} visible={visible} />;
};

export const Log: Layout.Renderer = ({ layoutKey, ...props }): ReactElement | null => {
  const log = useLoadRemote({
    name: "Log",
    targetVersion: ZERO_STATE.version,
    layoutKey,
    useSelect,
    fetcher: async (client, layoutKey) => {
      const { key, data } = await client.workspaces.log.retrieve(layoutKey);
      return { key, ...data } as State;
    },
    actionCreator: internalCreate,
  });
  if (log == null) return null;
  return <Loaded layoutKey={layoutKey} {...props} />;
};

export const SELECTABLE: Layout.Selectable = {
  key: LAYOUT_TYPE,
  title: "Log",
  icon: <Icon.Log />,
  create: (key) => create({ key }),
};

export const create =
  (initial: Partial<State> & Omit<Partial<Layout.State>, "type">): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Log", location = "mosaic", window, tab, ...rest } = initial;
    const key = initial.key ?? uuidv4();
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