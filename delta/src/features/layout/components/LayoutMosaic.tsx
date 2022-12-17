import { memo } from "react";

import {
  Mosaic as PlutoMosaic,
  debounce,
  Location,
  Space,
  Tab,
} from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import {
  useSelectMosaic,
  moveLayoutMosaicTab,
  deleteLayoutMosaicTab,
  selectLayoutMosaicTab,
  resizeLayoutMosaicTab,
} from "../store";

import { LayoutContent } from "./LayoutContent";

import { Logo } from "@/components";

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

const Content = memo(
  ({ tab }: { tab: Tab }): JSX.Element => <LayoutContent layoutKey={tab.tabKey} />
);
Content.displayName = "Content";
