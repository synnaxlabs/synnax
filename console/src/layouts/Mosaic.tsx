// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { selectWindowKey } from "@synnaxlabs/drift";
import { Icon, Logo } from "@synnaxlabs/media";
import {
  Breadcrumb,
  Button,
  componentRenderProp,
  Eraser,
  Menu as PMenu,
  Modal,
  Mosaic as Core,
  Nav,
  OS,
  Portal,
  Status,
  Synnax,
  type Tabs,
  Text,
  useDebouncedCallback,
} from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback, useLayoutEffect } from "react";
import { useDispatch, useStore } from "react-redux";

import { Controls } from "@/components";
import { Menu } from "@/components/menu";
import { NAV_DRAWERS, NavDrawer, NavMenu } from "@/components/nav/Nav";
import { Import } from "@/import";
import { INGESTORS } from "@/ingestors";
import { Layout } from "@/layout";
import { createSelector } from "@/layouts/Selector";
import { LinePlot } from "@/lineplot";
import { SERVICES } from "@/services";
import { type RootState, type RootStore } from "@/store";
import { Workspace } from "@/workspace";
import { WorkspaceServices } from "@/workspace/services";

const EmptyContent = (): ReactElement => (
  <Eraser.Eraser>
    <Logo.Watermark />;
  </Eraser.Eraser>
);

const emptyContent = <EmptyContent />;

export const MOSAIC_TYPE = "mosaic";

export const ContextMenu = ({
  keys,
}: PMenu.ContextMenuMenuProps): ReactElement | null => {
  if (keys.length === 0)
    return (
      <PMenu.Menu level="small" iconSpacing="small">
        <Menu.HardReloadItem />
      </PMenu.Menu>
    );
  const layoutKey = keys[0];
  const layout = Layout.useSelect(layoutKey);
  if (layout == null) return null;
  const C = Layout.useContextMenuRenderer(layout?.type);
  if (C == null)
    return (
      <PMenu.Menu level="small" iconSpacing="small">
        <Layout.MenuItems layoutKey={layoutKey} />
      </PMenu.Menu>
    );
  const res = <C layoutKey={layoutKey} />;
  return res;
};

interface ContentCProps extends Tabs.Tab {
  node: Portal.Node;
}

const ModalContent = ({ node, tabKey }: ContentCProps) => {
  const d = useDispatch();
  const layout = Layout.useSelectRequired(tabKey);
  const { windowKey, focused: focusedKey } = Layout.useSelectFocused();
  const focused = tabKey === focusedKey;
  const handleClose = () =>
    windowKey != null && d(Layout.setFocus({ windowKey, key: null }));
  const openInNewWindow = Layout.useOpenInNewWindow();
  const handleOpenInNewWindow = () => {
    openInNewWindow(tabKey);
    handleClose();
  };
  return (
    <Modal.Dialog visible close={handleClose} centered enabled={focused}>
      <Nav.Bar
        location="top"
        size="5rem"
        style={{ display: focused ? "flex" : "none" }}
      >
        {/*
         * We do this to reduce the number of mounted DOM nodes. For some reason removing
         * the entire bar causes react to crash, so we just hide its children.
         */}
        {focused && (
          <>
            <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
              <Breadcrumb.Breadcrumb icon={layout.icon}>
                {layout.name}
              </Breadcrumb.Breadcrumb>
            </Nav.Bar.Start>
            <Nav.Bar.End style={{ paddingRight: "1rem" }} empty>
              <Button.Icon onClick={handleOpenInNewWindow} size="small">
                <Icon.OpenInNewWindow style={{ color: "var(--pluto-gray-l8)" }} />
              </Button.Icon>
              <Button.Icon onClick={handleClose} size="small">
                <Icon.Subtract style={{ color: "var(--pluto-gray-l8)" }} />
              </Button.Icon>
            </Nav.Bar.End>
          </>
        )}
      </Nav.Bar>
      <Portal.Out node={node} />
    </Modal.Dialog>
  );
};

const contextMenu = componentRenderProp(ContextMenu);

interface MosaicProps {
  windowKey: string;
  mosaic: Core.Node;
}

export const Mosaic = memo((): ReactElement | null => {
  const [windowKey, mosaic] = Layout.useSelectMosaic();
  if (windowKey == null || mosaic == null) return null;
  return <Internal windowKey={windowKey} mosaic={mosaic} />;
});
Mosaic.displayName = "Mosaic";

/** LayoutMosaic renders the central layout mosaic of the application. */
const Internal = ({ windowKey, mosaic }: MosaicProps): ReactElement => {
  const store = useStore();
  const activeTab = Layout.useSelectActiveMosaicTabKey();
  const client = Synnax.use();
  const place = Layout.usePlacer();
  const dispatch = useDispatch();
  const addStatus = Status.useAggregator();
  const handleException = Status.useExceptionHandler();

  const handleDrop = useCallback(
    (key: number, tabKey: string, loc: location.Location): void => {
      if (windowKey == null) return;
      dispatch(Layout.moveMosaicTab({ key, tabKey, loc, windowKey }));
    },
    [dispatch, windowKey],
  );

  const handleCreate = useCallback(
    (mosaicKey: number, location: location.Location, tabKeys?: string[]) => {
      if (tabKeys == null) {
        place(
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
            placeLayout: place,
            addStatus,
            handleException,
          });
        } else
          place(
            createSelector({
              tab: { mosaicKey, location },
              location: "mosaic",
            }),
          );
      });
    },
    [place, store, client, addStatus],
  );

  LinePlot.useTriggerHold({
    defaultMode: "toggle",
    toggle: [["H"]],
  });

  const handleClose = useCallback(
    (tabKey: string): void => {
      dispatch(Layout.remove({ keys: [tabKey] }));
    },
    [dispatch],
  );

  const handleSelect = useCallback(
    (tabKey: string): void => {
      dispatch(Layout.selectMosaicTab({ tabKey }));
    },
    [dispatch],
  );

  const handleRename = useCallback(
    (tabKey: string, name: string): void => {
      dispatch(Layout.rename({ key: tabKey, name }));
    },
    [dispatch],
  );

  const handleResize = useDebouncedCallback(
    (key, size) => {
      dispatch(Layout.resizeMosaicTab({ key, size, windowKey }));
    },
    100,
    [dispatch, windowKey],
  );
  const handleFileDrop = useCallback(
    (nodeKey: number, loc: location.Location, event: React.DragEvent) => {
      const items = Array.from(event.dataTransfer.items);
      void Promise.all(
        items.map(async (item) => {
          try {
            await Import.dataTransferItem(item, {
              client,
              fileIngestors: INGESTORS,
              ingestDirectory: WorkspaceServices.ingest,
              layout: { tab: { mosaicKey: nodeKey, location: loc } },
              placeLayout: place,
              store,
            });
          } catch (e) {
            handleException(e, `Failed to read ${item.getAsFile()?.name ?? "file"}`);
          }
        }),
      );
    },
    [client, place, store],
  );

  // Creates a wrapper around the general purpose layout content to create a set of
  // content nodes that are rendered at the top level of the Mosaic and then 'portaled'
  // into their correct location. This means that moving layouts around in the Mosaic
  // or focusing them will not cause them to re-mount. This has considerable impacts
  // on the user experience, as it reduces necessary data fetching and expensive
  const [portalRef, portalNodes] = Core.usePortal({
    root: mosaic,
    onSelect: handleSelect,
    children: ({ tabKey, visible }) => (
      <Layout.Content key={tabKey} layoutKey={tabKey} forceHidden={visible === false} />
    ),
  });

  const renderProp = useCallback<Tabs.RenderProp>(
    (props) => (
      <ModalContent
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
};

export const NavTop = (): ReactElement | null => {
  const os = OS.use();
  const active = Layout.useSelectActiveMosaicLayout();
  const ws = Workspace.useSelectActive();
  const store = useStore<RootState>();
  const moveToMain = Layout.useMoveIntoMainWindow();
  const collapseButton = (
    <Button.Icon
      onClick={() => {
        const state = store.getState();
        const winKey = selectWindowKey(state);
        Object.values(state.layout.layouts)
          .filter((l) => l.windowKey === winKey && l.location === "mosaic")
          .forEach((l) => moveToMain(l.key));
      }}
    >
      <Icon.MoveToMainWindow />
    </Button.Icon>
  );
  return (
    <Nav.Bar
      className="console-main-nav-top"
      location="top"
      size={"5rem"}
      data-tauri-drag-region
    >
      <Nav.Bar.Start className="console-main-nav-top__start" data-tauri-drag-region>
        <Controls
          className="console-controls--macos"
          visibleIfOS="macOS"
          forceOS={os}
        />
        {os === "Windows" && <Logo className="console-main-nav-top__logo" />}
      </Nav.Bar.Start>
      <Nav.Bar.AbsoluteCenter data-tauri-drag-region>
        <Text.Text
          level="p"
          shade={6}
          style={{ transform: "scale(0.9)", cursor: "default" }}
          data-tauri-drag-region
        >
          {active?.name} {ws?.name != null ? ` - ${ws.name}` : ""}
        </Text.Text>
      </Nav.Bar.AbsoluteCenter>
      <Nav.Bar.End
        data-tauri-drag-region
        style={{ paddingRight: os == "Windows" ? "0" : "1.5rem" }}
      >
        {collapseButton}
        {os === "Windows" && (
          <Controls
            className="console-controls--windows"
            visibleIfOS="Windows"
            forceOS={os}
          />
        )}
      </Nav.Bar.End>
    </Nav.Bar>
  );
};

export const MosaicWindow = memo(
  ({ layoutKey }: Layout.RendererProps): ReactElement | null => {
    const { menuItems, onSelect } = Layout.useNavDrawer("bottom", NAV_DRAWERS);
    const dispatch = useDispatch();
    const [windowKey, mosaic] = Layout.useSelectMosaic();
    useLayoutEffect(() => {
      dispatch(
        Layout.setNavDrawer({
          windowKey: layoutKey,
          location: "bottom",
          menuItems: ["visualization"],
          activeItem: "visualization",
        }),
      );
    }, [layoutKey]);
    if (windowKey == null || mosaic == null) return null;
    return (
      <>
        <NavTop />
        <Internal windowKey={windowKey} mosaic={mosaic} />
        <NavDrawer location="bottom" />
        <Nav.Bar
          className="console-main-nav"
          location="bottom"
          style={{ paddingRight: "1.5rem", zIndex: 8 }}
          size="6rem"
        >
          <Nav.Bar.End>
            <NavMenu onChange={onSelect}>{menuItems}</NavMenu>
          </Nav.Bar.End>
        </Nav.Bar>
      </>
    );
  },
);
MosaicWindow.displayName = "MosaicWindow";
