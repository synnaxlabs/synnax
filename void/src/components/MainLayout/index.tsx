import {
  Mosaic,
  MosaicNode,
  Space,
  Location,
  debounce,
} from "@synnaxlabs/pluto";
import { useDispatch, useSelector } from "react-redux";
import { useClientConnector } from "../../features/cluster/components/useActiveClient";
import Plot from "../Plot/Plot";
import BottomNavbar from "./BottomNavbar";
import "./index.css";
import LeftNavbar from "./LeftNavbar";
import RightNavbar from "./RightNavbar";
import TopNavbar from "./TopNavbar";
import {
  insertTab,
  removeTab,
  moveTab,
  resize,
  MosaicSliceStoreState,
  selectTab,
} from "../../mosaic/slice";
import Logo from "../../components/Logo/Logo";

export default function Layout() {
  useClientConnector();
  return (
    <Space direction="vertical" size="large" className="main__container" empty>
      <TopNavbar />
      <Space
        direction="horizontal"
        size="large"
        style={{ overflow: "hidden" }}
        grow
        empty
      >
        <LeftNavbar />
        <Content />
        <RightNavbar />
      </Space>
      <BottomNavbar />
    </Space>
  );
}

const Content = () => {
  const mosaic = useSelector(
    (state: MosaicSliceStoreState) => state.mosaic.mosaics["main"]
  );
  const dispatch = useDispatch();

  const onDrop = (key: number, tabKey: string, loc: Location) => {
    dispatch(moveTab({ key, tabKey, loc }));
  };

  const onClose = (tabKey: string) => {
    dispatch(removeTab({ tabKey }));
  };

  const onSelect = (tabKey: string) => {
    dispatch(selectTab({ tabKey }));
  };

  const onResize = debounce(
    (key: number, size: number) => dispatch(resize({ key, size })),
    100
  );

  return (
    <div className="main__content">
      <Mosaic
        tree={mosaic}
        onDrop={onDrop}
        onClose={onClose}
        onSelect={onSelect}
        onResize={onResize}
        emptyContent={
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
        }
      >
        {Plot}
      </Mosaic>
    </div>
  );
};
