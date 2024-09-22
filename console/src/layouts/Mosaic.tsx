// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/Mosaic.css";

import { ontology } from "@synnaxlabs/client";
import { Icon, Logo } from "@synnaxlabs/media";
import {
  Breadcrumb,
  Button,
  componentRenderProp,
  Eraser,
  Menu,
  Modal,
  Mosaic as Core,
  Nav,
  Portal,
  Status,
  Synnax,
  Tabs,
  useDebouncedCallback,
} from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback, useLayoutEffect } from "react";
import { useDispatch, useStore } from "react-redux";

import { NAV_DRAWERS, NavDrawer, NavMenu } from "@/components/nav/Nav";
import { Confirm } from "@/confirm";
import { Layout } from "@/layout";
import { Content } from "@/layout/Content";
import { usePlacer } from "@/layout/hooks";
import { useSelectActiveMosaicTabKey, useSelectMosaic } from "@/layout/selectors";
import {
  moveMosaicTab,
  remove,
  rename,
  resizeMosaicTab,
  selectMosaicTab,
  setNavDrawer,
} from "@/layout/slice";
import { createSelector } from "@/layouts/Selector";
import { LinePlot } from "@/lineplot";
import { Schematic } from "@/schematic";
import { SERVICES } from "@/services";
import { type RootStore } from "@/store";
import { Workspace } from "@/workspace";

const EmptyContent = (): ReactElement => (
  <Eraser.Eraser>
    <Logo.Watermark />;
  </Eraser.Eraser>
);

const emptyContent = <EmptyContent />;

export const MOSAIC_TYPE = "mosaic";

const FILE_HANDLERS = [Schematic.fileHandler, LinePlot.fileHandler];

export const ContextMenu = ({
  keys,
}: Menu.ContextMenuMenuProps): ReactElement | null => {
  if (keys.length === 0) return null;
  const layoutKey = keys[0];
  const layout = Layout.useSelect(layoutKey);
  if (layout == null) return null;
  const Menu = Layout.useContextMenuRenderer(layout?.type);
  if (Menu == null) return null;
  return <Menu layoutKey={layoutKey} />;
};

interface ContentCProps extends Tabs.Tab {
  node: Portal.Node;
}

const ContentC = ({ node, tabKey }: ContentCProps) => {
  const d = useDispatch();
  const layout = Layout.useSelectRequired(tabKey);
  const [windowKey, focusedKey] = Layout.useSelectFocused();
  const focused = tabKey === focusedKey;
  const handleClose = () => d(Layout.setFocus({ windowKey, key: null }));
  return (
    <Modal.Base visible close={handleClose} centered enabled={focused}>
      <Nav.Bar
        style={{ display: focused ? "flex" : "none" }}
        location="top"
        size="4.5rem"
      >
        <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
          <Breadcrumb.Breadcrumb icon={layout.icon}>
            {layout.name}
          </Breadcrumb.Breadcrumb>
        </Nav.Bar.Start>
        <Nav.Bar.End style={{ paddingRight: "1rem" }}>
          <Button.Icon onClick={handleClose} size="small">
            <Icon.Subtract style={{ color: "var(--pluto-gray-l8)" }} />
          </Button.Icon>
        </Nav.Bar.End>
      </Nav.Bar>
      <Portal.Out node={node} />
    </Modal.Base>
  );
};

const contextMenu = componentRenderProp(ContextMenu);

/** LayoutMosaic renders the central layout mosaic of the application. */
export const Mosaic = memo((): ReactElement => {
  const [windowKey, mosaic] = useSelectMosaic();
  const store = useStore();
  const activeTab = useSelectActiveMosaicTabKey();
  const client = Synnax.use();
  const placer = usePlacer();
  const dispatch = useDispatch();

  const handleDrop = useCallback(
    (key: number, tabKey: string, loc: location.Location): void => {
      dispatch(moveMosaicTab({ key, tabKey, loc, windowKey }));
    },
    [dispatch, windowKey],
  );

  const handleCreate = useCallback(
    (mosaicKey: number, location: location.Location, tabKeys?: string[]) => {
      if (tabKeys == null) {
        placer(
          createSelector({
            tab: { mosaicKey, location },
            location: "mosaic",
          }),
        );
        return;
      }
      tabKeys.forEach((tabKey) => {
        const res = ontology.stringIDZ.safeParse(tabKey);
        if (res.success) {
          const id = new ontology.ID(res.data);
          if (client == null) return;
          SERVICES[id.type].onMosaicDrop?.({
            client,
            store: store as RootStore,
            id,
            nodeKey: mosaicKey,
            location,
            placeLayout: placer,
          });
        } else
          placer(
            createSelector({
              tab: { mosaicKey, location },
              location: "mosaic",
            }),
          );
      });
    },
    [placer, store, client],
  );

  LinePlot.useTriggerHold({
    defaultMode: "hold",
    hold: [["H"]],
    toggle: [["H", "H"]],
  });

  const handleClose = useCallback(
    (tabKey: string): void => {
      dispatch(remove({ keys: [tabKey] }));
    },
    [dispatch],
  );

  const handleSelect = useCallback(
    (tabKey: string): void => {
      dispatch(selectMosaicTab({ tabKey }));
    },
    [dispatch],
  );

  const handleRename = useCallback(
    (tabKey: string, name: string): void => {
      dispatch(rename({ key: tabKey, name }));
    },
    [dispatch],
  );

  const handleResize = useDebouncedCallback(
    (key, size) => {
      dispatch(resizeMosaicTab({ key, size, windowKey }));
    },
    100,
    [dispatch, windowKey],
  );

  const workspaceKey = Workspace.useSelectActiveKey();
  const confirm = Confirm.useModal();
  const addStatus = Status.useAggregator();

  const handleFileDrop = useCallback(
    (nodeKey: number, loc: location.Location, event: React.DragEvent) => {
      void (async () => {
        const files = Array.from(event.dataTransfer.files);
        for (const file of files) {
          const name = file.name;
          try {
            if (file.type !== "application/json")
              throw Error(`${name} is not a JSON file`);
            const buffer = await file.arrayBuffer();
            const fileAsJSON = JSON.parse(new TextDecoder().decode(buffer));

            let handlerFound = false;
            for (const fileHandler of FILE_HANDLERS) {
              if (
                await fileHandler({
                  file: fileAsJSON,
                  placer,
                  name,
                  store,
                  confirm,
                  client,
                  workspaceKey: workspaceKey ?? undefined,
                  dispatch,
                  tab: { mosaicKey: nodeKey, location: loc },
                })
              ) {
                handlerFound = true;
                break;
              }
            }
            if (!handlerFound)
              throw Error(`${name} is not recognized as a Synnax object`);
          } catch (e) {
            if (e instanceof Error)
              addStatus({
                variant: "error",
                message: `Failed to read ${name}`,
                description: e.message,
              });
          }
        }
      })();
    },
    [dispatch],
  );

  const [portalRef, portalNodes] = Core.usePortal({
    root: mosaic,
    onSelect: handleSelect,
    children: ({ tabKey, visible }) => (
      <Content key={tabKey} layoutKey={tabKey} forceHidden={visible === false} />
    ),
  });

  const renderProp = useCallback<Tabs.RenderProp>(
    (props) => (
      <ContentC
        key={props.tabKey}
        node={portalRef.current.get(props.tabKey) as Portal.Node}
        {...props}
      />
    ),
    [],
  );

  return (
    <>
      {portalNodes}
      <Core.Mosaic
        root={mosaic}
        onDrop={handleDrop}
        onClose={handleClose}
        onSelect={handleSelect}
        contextMenu={contextMenu}
        onResize={handleResize}
        emptyContent={emptyContent}
        onRename={handleRename}
        onCreate={handleCreate}
        activeTab={activeTab ?? undefined}
        onFileDrop={handleFileDrop}
      >
        {renderProp}
      </Core.Mosaic>
    </>
  );
});
Mosaic.displayName = "Mosaic";

export const Window = memo(({ layoutKey }: Layout.RendererProps): ReactElement => {
  const { menuItems, onSelect } = Layout.useNavDrawer("bottom", NAV_DRAWERS);
  const d = useDispatch();
  useLayoutEffect(() => {
    d(
      setNavDrawer({
        windowKey: layoutKey,
        location: "bottom",
        menuItems: ["visualization"],
        activeItem: "visualization",
      }),
    );
  }, [layoutKey]);
  return (
    <>
      <Mosaic />
      <NavDrawer location="bottom" />
      <Nav.Bar
        className="console-main-nav"
        location="bottom"
        style={{ paddingRight: "1.5rem" }}
        size={7 * 6}
      >
        <Nav.Bar.End>
          <NavMenu onChange={onSelect}>{menuItems}</NavMenu>
        </Nav.Bar.End>
      </Nav.Bar>
    </>
  );
});
Window.displayName = "MosaicWindow";
