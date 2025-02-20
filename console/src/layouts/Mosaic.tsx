// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon, Logo } from "@synnaxlabs/media";
import {
  Breadcrumb,
  Button,
  componentRenderProp,
  Eraser,
  Menu as PMenu,
  Modal,
  Mosaic as Core,
  Nav as PNav,
  Portal,
  Status,
  Synnax,
  type Tabs,
  useDebouncedCallback,
} from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import { memo, useCallback, useLayoutEffect } from "react";
import { useDispatch, useStore } from "react-redux";

import { Menu } from "@/components";
import { Import } from "@/import";
import { INGESTORS } from "@/ingestors";
import { Layout } from "@/layout";
import { Nav } from "@/layouts/nav";
import { SELECTOR_LAYOUT } from "@/layouts/Selector";
import { LinePlot } from "@/lineplot";
import { SERVICES } from "@/services";
import { type RootStore } from "@/store";
import { WorkspaceServices } from "@/workspace/services";

const EmptyContent = () => (
  <Eraser.Eraser>
    <Logo.Watermark />;
  </Eraser.Eraser>
);

const EMPTY_CONTENT = <EmptyContent />;

export const MOSAIC_LAYOUT_TYPE = "mosaic";

const ContextMenu = ({ keys }: PMenu.ContextMenuMenuProps) => {
  if (keys.length === 0)
    return (
      <PMenu.Menu level="small" iconSpacing="small">
        <Menu.HardReloadItem />
      </PMenu.Menu>
    );
  const layoutKey = keys[0];
  const layout = Layout.useSelect(layoutKey);
  if (layout == null) return null;
  const C = Layout.useContextMenuRenderer(layout.type);
  return C == null ? (
    <PMenu.Menu level="small" iconSpacing="small">
      <Layout.MenuItems layoutKey={layoutKey} />
    </PMenu.Menu>
  ) : (
    <C layoutKey={layoutKey} />
  );
};

interface ModalContentProps extends Tabs.Tab {
  node: Portal.Node;
}

const ModalContent = ({ node, tabKey }: ModalContentProps) => {
  const dispatch = useDispatch();
  const layout = Layout.useSelectRequired(tabKey);
  const { windowKey, focused: focusedKey } = Layout.useSelectFocused();
  const focused = tabKey === focusedKey;
  const handleClose = () =>
    windowKey != null && dispatch(Layout.setFocus({ windowKey, key: null }));
  const openInNewWindow = Layout.useOpenInNewWindow();
  const handleOpenInNewWindow = () => {
    openInNewWindow(tabKey);
    handleClose();
  };
  return (
    <Modal.Dialog visible close={handleClose} centered enabled={focused}>
      <PNav.Bar
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
            <PNav.Bar.Start style={{ paddingLeft: "2rem" }}>
              <Breadcrumb.Breadcrumb icon={layout.icon}>
                {layout.name}
              </Breadcrumb.Breadcrumb>
            </PNav.Bar.Start>
            <PNav.Bar.End style={{ paddingRight: "1rem" }} empty>
              <Button.Icon onClick={handleOpenInNewWindow} size="small">
                <Icon.OpenInNewWindow style={{ color: "var(--pluto-gray-l8)" }} />
              </Button.Icon>
              <Button.Icon onClick={handleClose} size="small">
                <Icon.Subtract style={{ color: "var(--pluto-gray-l8)" }} />
              </Button.Icon>
            </PNav.Bar.End>
          </>
        )}
      </PNav.Bar>
      <Portal.Out node={node} />
    </Modal.Dialog>
  );
};

const contextMenu = componentRenderProp(ContextMenu);

interface MosaicProps {
  windowKey: string;
  mosaic: Core.Node;
}

export const Mosaic = memo(() => {
  const [windowKey, mosaic] = Layout.useSelectMosaic();
  return windowKey == null || mosaic == null ? null : (
    <Internal windowKey={windowKey} mosaic={mosaic} />
  );
});
Mosaic.displayName = "Mosaic";

/** LayoutMosaic renders the central layout mosaic of the application. */
const Internal = ({ windowKey, mosaic }: MosaicProps) => {
  const store = useStore();
  const activeTab = Layout.useSelectActiveMosaicTabKey();
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const dispatch = useDispatch();
  const addStatus = Status.useAdder();
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
        placeLayout({ ...SELECTOR_LAYOUT, tab: { mosaicKey, location } });
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
            placeLayout,
            addStatus,
            handleException,
          });
        } else placeLayout({ ...SELECTOR_LAYOUT, tab: { mosaicKey, location } });
      });
    },
    [placeLayout, store, client, addStatus],
  );

  LinePlot.useTriggerHold({ defaultMode: "toggle", toggle: [["H"]] });

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
              placeLayout,
              store,
            });
          } catch (e) {
            handleException(e, `Failed to read ${item.getAsFile()?.name ?? "file"}`);
          }
        }),
      );
    },
    [client, placeLayout, store],
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
        emptyContent={EMPTY_CONTENT}
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

export const MosaicWindow = memo(({ layoutKey }: Layout.RendererProps) => {
  const { menuItems, onSelect } = Layout.useNavDrawer("bottom", Nav.DRAWER_ITEMS);
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
  return windowKey == null || mosaic == null ? null : (
    <>
      <Nav.Top />
      <Internal windowKey={windowKey} mosaic={mosaic} />
      <Nav.Drawer location="bottom" />
      <PNav.Bar
        className="console-main-nav"
        location="bottom"
        style={{ paddingRight: "1.5rem", zIndex: 8 }}
        size="6rem"
      >
        <PNav.Bar.End>
          <Nav.Menu onChange={onSelect}>{menuItems}</Nav.Menu>
        </PNav.Bar.End>
      </PNav.Bar>
    </>
  );
});
MosaicWindow.displayName = "MosaicWindow";
