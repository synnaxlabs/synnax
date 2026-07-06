// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/app/Mosaic.css";

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
  type Menu,
  Mosaic as Base,
  Nav,
  type Pluto,
  Portal,
  Status,
  Synnax,
  Tabs,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { caseconv, type location } from "@synnaxlabs/x";
import {
  type ComponentType,
  memo,
  type ReactElement,
  useCallback,
  useEffect,
  useLayoutEffect,
} from "react";

import { Nav as AppNav } from "@/app/nav";
import { createSelectorLayout, useSelectorVisible } from "@/app/Selector";
import { LinePlot } from "@/feature/lineplot";
import { Project } from "@/feature/project";
import { ContextMenu as PlatformContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Import } from "@/platform/import";
import { Layout } from "@/platform/layout";
import { Ontology } from "@/platform/ontology";
import { Session } from "@/session";

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

const ContextMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement | null => {
  if (keys.length === 0)
    return (
      <PlatformContextMenu.Menu>
        <PlatformContextMenu.ReloadConsoleItem />
      </PlatformContextMenu.Menu>
    );
  const layoutKey = keys[0];
  const layout = Session.Layout.useSelect(layoutKey);
  if (layout == null) return null;
  const C = Layout.useContextMenuRenderer(layout.type);
  return C == null ? (
    <PlatformContextMenu.Menu>
      <Layout.MenuItems layoutKey={layoutKey} />
    </PlatformContextMenu.Menu>
  ) : (
    <C layoutKey={layoutKey} />
  );
};

interface ModalContentProps extends Tabs.Tab {
  node: Portal.Node;
}

const ModalContent = ({ node, tabKey }: ModalContentProps): ReactElement => {
  const dispatch = Session.useDispatch();
  const layout = Session.Layout.useSelectRequired(tabKey);
  const { windowKey, focused: focusedKey } = Session.Layout.useSelectFocused();
  const focused = tabKey === focusedKey;
  const handleClose = () =>
    windowKey != null && dispatch(Session.Layout.setFocus({ windowKey, key: null }));
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
        <Nav.Bar
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
              <Nav.Bar.Start>
                <Breadcrumb.Breadcrumb>
                  <Breadcrumb.Segment>
                    {Icon.resolve(layout.icon)}
                    {layout.name}
                  </Breadcrumb.Segment>
                </Breadcrumb.Breadcrumb>
              </Nav.Bar.Start>
              <Nav.Bar.End pack>
                {Session.Runtime.ENGINE === "tauri" && (
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
              </Nav.Bar.End>
            </>
          )}
        </Nav.Bar>
        <Portal.Out node={node} />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};

const contextMenu = Component.renderProp(ContextMenu);

interface CustomTabNameProps extends Tabs.NameProps {
  useName: Layout.UseName;
}

const CustomTabName = ({
  useName,
  tabKey,
  name,
  onRename: propsOnRename,
  ...rest
}: CustomTabNameProps): ReactElement => {
  const handleLayoutRename = useCallback(
    (name: string) => propsOnRename?.(tabKey, name),
    [tabKey, propsOnRename],
  );
  const { onRename, retrieve } = useName(tabKey, handleLayoutRename);
  useEffect(() => {
    retrieve();
  }, [retrieve]);
  const handleRename = useCallback(
    (_: string, name: string) => {
      handleLayoutRename(name);
      onRename(name);
    },
    [handleLayoutRename, onRename],
  );
  return (
    <Tabs.DefaultName tabKey={tabKey} name={name} onRename={handleRename} {...rest} />
  );
};

const TabName: ComponentType<Tabs.NameProps> = (props) => {
  const type = Session.Layout.useSelectType(props.tabKey);
  const useName = Layout.useNameHook(type);
  if (useName != null) return <CustomTabName key={type} useName={useName} {...props} />;
  return <Tabs.DefaultName {...props} />;
};

const renderTabName = Component.renderProp(TabName);

interface MosaicProps {
  windowKey: string;
  mosaic: Base.Node;
}

export const Mosaic = memo((): ReactElement | null => {
  const [windowKey, mosaic] = Session.Layout.useSelectMosaic();
  return windowKey == null || mosaic == null ? null : (
    <Internal windowKey={windowKey} mosaic={mosaic} />
  );
});
Mosaic.displayName = "Mosaic";

/** LayoutMosaic renders the central layout mosaic of the application. */
const Internal = ({ windowKey, mosaic }: MosaicProps): ReactElement => {
  const store = Session.useStore();
  const activeTab = Session.Layout.useSelectActiveMosaicTabState();
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const removeLayout = Layout.useRemover();
  const dispatch = Session.useDispatch();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  const handleDrop = useCallback(
    (key: number, tabKey: string, loc: location.Location, index?: number): void => {
      if (windowKey == null) return;
      dispatch(Session.Layout.moveMosaicTab({ key, tabKey, loc, windowKey, index }));
    },
    [dispatch, windowKey],
  );

  const services = Ontology.useServices();
  const fileIngesters = Import.useFileIngesters();

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
            store,
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
    [placeLayout, store, client, addStatus, handleError, removeLayout, services],
  );

  LinePlot.useTriggerHold();

  const handleClose = Layout.useRemover();

  const handleSelect = useCallback(
    (tabKey: string): void => {
      dispatch(Session.Layout.selectMosaicTab({ tabKey }));
    },
    [dispatch],
  );

  const handleRename = useCallback(
    (tabKey: string, name: string): void => {
      dispatch(Session.Layout.rename({ key: tabKey, name }));
    },
    [dispatch],
  );

  const handleResize = useCallback(
    (key: number, size: number) => {
      dispatch(Session.Layout.resizeMosaicTab({ key, size, windowKey }));
    },
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
              fileIngesters,
              ingestDirectory: Project.ingest,
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
    [
      client,
      fileIngesters,
      placeLayout,
      store,
      fluxStore,
      handleError,
      store,
      fluxStore,
    ],
  );

  // Creates a wrapper around the general purpose layout content to create a set of
  // content nodes that are rendered at the top level of the Mosaic and then 'portaled'
  // into their correct location. This means that moving layouts around in the Mosaic
  // or focusing them will not cause them to re-mount. This has considerable impacts
  // on the user experience, as it reduces necessary data fetching and expensive
  const [portalRef, portalNodes] = Base.usePortal({
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
      <Base.Mosaic
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
        addTooltip="Create component"
        className={CSS.B("mosaic")}
        tabName={renderTabName}
      >
        {renderProp}
      </Base.Mosaic>
    </>
  );
};

export const MosaicWindow = memo<Layout.Renderer>(
  ({ layoutKey }: Layout.RendererProps) => {
    const dispatch = Session.useDispatch();
    const [windowKey, mosaic] = Session.Layout.useSelectMosaic();
    useLayoutEffect(() => {
      dispatch(Session.Nav.showBottom({}));
    }, [layoutKey]);
    if (windowKey == null || mosaic == null) return null;
    return (
      <>
        <AppNav.Bar.AuxTop />
        <Flex.Box
          y
          gap="tiny"
          grow
          className={CSS.B("mosaic-window")}
          style={{ padding: "1rem", paddingTop: 0, overflow: "hidden" }}
        >
          <Internal windowKey={windowKey} mosaic={mosaic} />
          <AppNav.Drawer.Bottom />
        </Flex.Box>
      </>
    );
  },
);
MosaicWindow.displayName = "MosaicWindow";
