// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/Mosaic.css";

import { ontology } from "@synnaxlabs/client";
import { Logo } from "@synnaxlabs/media";
import {
  Breadcrumb,
  Button,
  Component,
  Dialog,
  Eraser,
  Flex,
  Flux,
  Icon,
  Menu as PMenu,
  Mosaic as Core,
  Nav as PNav,
  OS,
  type Pluto,
  Portal,
  Status,
  Synnax,
  type Tabs,
  Text,
  Triggers,
  useDebouncedCallback,
} from "@synnaxlabs/pluto";
import { caseconv, type location, TimeSpan } from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback, useLayoutEffect } from "react";
import { useDispatch, useStore } from "react-redux";

import { Menu } from "@/components";
import { CSS } from "@/css";
import { Import } from "@/import";
import { FILE_INGESTORS } from "@/ingestors";
import { Layout } from "@/layout";
import { Controls } from "@/layout/Controls";
import { Nav } from "@/layouts/nav";
import { createSelectorLayout, useSelectorVisible } from "@/layouts/Selector";
import { LinePlot } from "@/lineplot";
import { Ontology } from "@/ontology";
import { Runtime } from "@/runtime";
import { type RootState, type RootStore } from "@/store";
import { Workspace } from "@/workspace";
import { WorkspaceServices } from "@/workspace/services";

const EmptyContent = (): ReactElement => {
  const createComponentEnabled = useSelectorVisible();
  return (
    <Eraser.Eraser>
      <Flex.Box gap={5} center>
        <Logo className="synnax-logo-watermark" />
        {createComponentEnabled && (
          <Flex.Box x gap="small">
            <Text.Text level="h5" weight={450} color={9}>
              New Component
            </Text.Text>
            <Flex.Box x empty>
              <Triggers.Text level="h5" trigger={["Control", "T"]} />
            </Flex.Box>
          </Flex.Box>
        )}
      </Flex.Box>
    </Eraser.Eraser>
  );
};
export const MOSAIC_LAYOUT_TYPE = "mosaic";

const ContextMenu = ({ keys }: PMenu.ContextMenuMenuProps): ReactElement | null => {
  if (keys.length === 0)
    return (
      <PMenu.Menu level="small" gap="small">
        <Menu.ReloadConsoleItem />
      </PMenu.Menu>
    );
  const layoutKey = keys[0];
  const layout = Layout.useSelect(layoutKey);
  if (layout == null) return null;
  const C = Layout.useContextMenuRenderer(layout.type);
  return C == null ? (
    <PMenu.Menu level="small" gap="small">
      <Layout.MenuItems layoutKey={layoutKey} />
    </PMenu.Menu>
  ) : (
    <C layoutKey={layoutKey} />
  );
};

interface ModalContentProps extends Tabs.Tab {
  node: Portal.Node;
}

const ModalContent = ({ node, tabKey }: ModalContentProps): ReactElement => {
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
    <Dialog.Frame
      onVisibleChange={handleClose}
      visible={focused}
      full
      modalPosition="slammed"
      variant="modal"
      background={focused ? 0 : undefined}
    >
      <Dialog.Dialog
        passthrough
        full
        className={CSS(CSS.B(caseconv.toKebab(layout.type)), CSS.B("mosaic-modal"))}
      >
        <PNav.Bar
          location="top"
          size="5rem"
          className={CSS(
            CSS.B("mosaic-modal-bar"),
            focused && CSS.BM("mosaic-modal-bar", "focused"),
          )}
          bordered
        >
          {/*
           * We do this to reduce the number of mounted DOM nodes. For some reason removing
           * the entire bar causes react to crash, so we just hide its children.
           */}
          {focused && (
            <>
              <PNav.Bar.Start>
                <Breadcrumb.Breadcrumb>
                  <Breadcrumb.Segment>
                    {Icon.resolve(layout.icon)}
                    {layout.name}
                  </Breadcrumb.Segment>
                </Breadcrumb.Breadcrumb>
              </PNav.Bar.Start>
              <PNav.Bar.End pack>
                {Runtime.ENGINE === "tauri" && (
                  <Button.Button
                    onClick={handleOpenInNewWindow}
                    size="small"
                    textColor={9}
                  >
                    <Icon.OpenInNewWindow />
                  </Button.Button>
                )}
                <Button.Button onClick={handleClose} size="small" textColor={9}>
                  <Icon.Subtract />
                </Button.Button>
              </PNav.Bar.End>
            </>
          )}
        </PNav.Bar>
        <Portal.Out node={node} />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};

const contextMenu = Component.renderProp(ContextMenu);

interface MosaicProps {
  windowKey: string;
  mosaic: Core.Node;
}

const RESIZE_DEBOUNCE = TimeSpan.milliseconds(100).milliseconds;

export const Mosaic = memo((): ReactElement | null => {
  const [windowKey, mosaic] = Layout.useSelectMosaic();
  return windowKey == null || mosaic == null ? null : (
    <Internal windowKey={windowKey} mosaic={mosaic} />
  );
});
Mosaic.displayName = "Mosaic";

/** LayoutMosaic renders the central layout mosaic of the application. */
const Internal = ({ windowKey, mosaic }: MosaicProps): ReactElement => {
  const store = useStore<RootState>();
  const activeTab = Layout.useSelectActiveMosaicTabState();
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const removeLayout = Layout.useRemover();
  const dispatch = useDispatch();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  const handleDrop = useCallback(
    (key: number, tabKey: string, loc: location.Location, index?: number): void => {
      if (windowKey == null) return;
      dispatch(Layout.moveMosaicTab({ key, tabKey, loc, windowKey, index }));
    },
    [dispatch, windowKey],
  );

  const services = Ontology.useServices();

  const handleCreate = useCallback(
    (mosaicKey: number, location: location.Location, tabKeys?: string[]) => {
      if (tabKeys == null) {
        placeLayout(createSelectorLayout({ tab: { mosaicKey, location } }));
        return;
      }
      tabKeys.forEach((tabKey) => {
        const res = ontology.idZ.safeParse(tabKey);
        if (res.success) {
          const id = res.data;
          if (client == null) return;
          services[id.type].onMosaicDrop?.({
            client,
            store: store as RootStore,
            id,
            nodeKey: mosaicKey,
            location,
            placeLayout,
            addStatus,
            handleError,
            removeLayout,
            services,
          });
        } else placeLayout(createSelectorLayout({ tab: { mosaicKey, location } }));
      });
    },
    [placeLayout, store, client, addStatus],
  );

  LinePlot.useTriggerHold({ defaultMode: "toggle", toggle: [["H"]] });

  const handleClose = Layout.useRemover();

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
    RESIZE_DEBOUNCE,
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
              fileIngestors: FILE_INGESTORS,
              ingestDirectory: WorkspaceServices.ingest,
              layout: { tab: { mosaicKey: nodeKey, location: loc } },
              placeLayout,
              store,
              fluxStore,
            });
          } catch (e) {
            handleError(e, `Failed to read ${item.getAsFile()?.name ?? "file"}`);
          }
        }),
      );
    },
    [client, placeLayout, store, fluxStore],
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
  const selectorVisible = useSelectorVisible();

  return (
    <>
      {portalNodes}
      <Core.Mosaic
        rounded={1}
        bordered
        borderColor={5}
        background={0}
        root={mosaic}
        onDrop={handleDrop}
        onClose={handleClose}
        onSelect={handleSelect}
        contextMenu={contextMenu}
        onResize={handleResize}
        emptyContent={<EmptyContent />}
        onRename={handleRename}
        onCreate={selectorVisible ? handleCreate : undefined}
        activeTab={activeTab.layoutKey ?? undefined}
        onFileDrop={handleFileDrop}
        addTooltip="Create Component"
        className={CSS.B("mosaic")}
      >
        {renderProp}
      </Core.Mosaic>
    </>
  );
};

const NavTop = (): ReactElement | null => {
  const os = OS.use();
  const isWindowsOS = os === "Windows";
  const { onSelect } = Layout.useNavDrawer("bottom", Nav.DRAWER_ITEMS);
  const activeName = Layout.useSelectActiveMosaicTabName();
  const activeWorkspaceName = Workspace.useSelectActiveName();
  const button = (
    <Button.Button
      variant="outlined"
      className={CSS.BE("mosaic", "controls-button")}
      onClick={() => onSelect("visualization")}
      justify="center"
      size="small"
      contrast={2}
      color={9}
      weight={450}
      triggerIndicator={["V"]}
    >
      <Icon.Visualize />
      Controls
    </Button.Button>
  );
  return (
    <Layout.Nav.Bar
      location="top"
      size="6rem"
      data-tauri-drag-region
      bordered={false}
      className={CSS.BE("mosaic", "bar")}
    >
      <PNav.Bar.Start data-tauri-drag-region align="center">
        <Controls visibleIfOS="macOS" forceOS={os} />
        {isWindowsOS && <Logo />}
        {isWindowsOS && button}
      </PNav.Bar.Start>
      <PNav.Bar.AbsoluteCenter>
        <Text.Text
          level="small"
          weight={500}
          color={10}
          data-tauri-drag-region
          style={{ cursor: "default" }}
        >
          {activeName} {activeWorkspaceName && `- ${activeWorkspaceName}`}
        </Text.Text>
      </PNav.Bar.AbsoluteCenter>
      <PNav.Bar.End data-tauri-drag-region align="center" justify="end">
        {isWindowsOS ? <Controls visibleIfOS="Windows" forceOS={os} /> : button}
      </PNav.Bar.End>
    </Layout.Nav.Bar>
  );
};

export const MosaicWindow = memo<Layout.Renderer>(
  ({ layoutKey }: Layout.RendererProps) => {
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
        <Flex.Box
          y
          gap="tiny"
          grow
          className={CSS.B("mosaic-window")}
          style={{ padding: "1rem", paddingTop: 0, overflow: "hidden" }}
        >
          <Internal windowKey={windowKey} mosaic={mosaic} />
          <Layout.Nav.Drawer location="bottom" menuItems={Nav.DRAWER_ITEMS} />
        </Flex.Box>
      </>
    );
  },
);
MosaicWindow.displayName = "MosaicWindow";
