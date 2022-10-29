import {
  useSelectMosaic,
  moveLayoutMosaicTab,
  deleteLayoutMosaicTab,
  selectLayoutMosaicTab,
  resizeLayoutMosaicTab,
} from "../store";
import { useDispatch } from "react-redux";
import {
  Mosaic as PlutoMosaic,
  debounce,
  Location,
  Space,
  Tab,
} from "@synnaxlabs/pluto";
import { Logo } from "@/components";

import { memo } from "react";
import { LayoutContent } from "./LayoutContent";

const MOSAIC_RESIZE_DEBOUNCE = 100; // ms

export const LayoutMosaic = () => {
  const mosaic = useSelectMosaic();
  const dispatch = useDispatch();

  const handleDrop = (key: number, tabKey: string, loc: Location) => {
    dispatch(moveLayoutMosaicTab({ key, tabKey, loc }));
  };

  const handleClose = (tabKey: string) => {
    dispatch(deleteLayoutMosaicTab({ tabKey }));
  };

  const handleSelect = (tabKey: string) => {
    dispatch(selectLayoutMosaicTab({ tabKey }));
  };

  const onResize = debounce(
    (key: number, size: number) =>
      dispatch(resizeLayoutMosaicTab({ key, size })),
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

const EmptyContent = () => (
  <Space
    style={{ width: "100%", height: "100%" }}
    justify="spaceAround"
    align="center"
  >
    <Logo
      style={{
        height: "10%",
        opacity: 0.5,
      }}
    />
  </Space>
);

const Content = memo(({ tab }: { tab: Tab }) => {
  return <LayoutContent layoutKey={tab.tabKey} />;
});
