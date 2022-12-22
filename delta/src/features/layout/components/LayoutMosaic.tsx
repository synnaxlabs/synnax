import { memo } from "react";

import { Mosaic as PlutoMosaic, debounce, Space } from "@synnaxlabs/pluto";
import type { Location, Tab } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import {
  useSelectMosaic,
  moveLayoutMosaicTab,
  deleteLayoutMosaicTab,
  selectLayoutMosaicTab,
  resizeLayoutMosaicTab,
  renameLayoutMosaicTab,
} from "../store";

import { LayoutContent } from "./LayoutContent";

import { Logo } from "@/components";
import { Loader } from "@/components/Loader/Loader";

export const LayoutMosaic = (): JSX.Element => {
  const mosaic = useSelectMosaic();
  const dispatch = useDispatch();

  const handleDrop = (key: number, tabKey: string, loc: Location): void => {
    dispatch(moveLayoutMosaicTab({ key, tabKey, loc }));
  };

  const handleClose = (tabKey: string): void => {
    dispatch(deleteLayoutMosaicTab({ tabKey }));
  };

  const handleSelect = (tabKey: string): void => {
    dispatch(selectLayoutMosaicTab({ tabKey }));
  };

  const handleTitleChange = (tabKey: string, title: string): void => {
    dispatch(renameLayoutMosaicTab({ tabKey, title }));
  };

  const onResize = debounce(
    (key: number, size: number) => dispatch(resizeLayoutMosaicTab({ key, size })),
    0
  );

  return (
    <PlutoMosaic
      root={mosaic}
      onDrop={handleDrop}
      onClose={handleClose}
      onSelect={handleSelect}
      onResize={onResize}
      emptyContent={EmptyContent}
      onTitleChange={handleTitleChange}
    >
      {Content}
    </PlutoMosaic>
  );
};

const EmptyContent = (): JSX.Element => (
  <Space style={{ width: "100%", height: "100%" }} justify="spaceAround" align="center">
    <Logo
      style={{
        height: "10%",
        opacity: 0.5,
      }}
    />
  </Space>
);

export const LoadingContent = (): JSX.Element => (
  <Space style={{ width: "100%", height: "100%" }} justify="spaceAround" align="center">
    <Loader />
  </Space>
);

const Content = memo(
  ({ tab }: { tab: Tab }): JSX.Element => <LayoutContent layoutKey={tab.tabKey} />
);
Content.displayName = "Content";
