import { Mosaic, useMosaic, MosaicNode, Space } from "@synnaxlabs/pluto";
import { useClientConnector } from "../../cluster/useActiveClient";
import Plot from "../Plot/Plot";
import BottomNavbar from "./BottomNavbar";
import "./index.css";
import LeftNavbar from "./LeftNavbar";
import RightNavbar from "./RightNavbar";
import TopNavbar from "./TopNavbar";

const initialTree: MosaicNode = {
  key: 0,
  level: 0,
  direction: "horizontal",
  first: {
    level: 1,
    key: 1,
    tabs: [
      {
        tabKey: "1",
        title: "Tab 1",
        content: <Plot />,
      },
    ],
  },
  last: {
    level: 1,
    key: 2,
    tabs: [
      {
        tabKey: "2",
        title: "Tab 2",
        content: <Plot />,
      },
      {
        tabKey: "3",
        title: "Tab 3",
        content: <Plot />,
      },
    ],
  },
};

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
  const { insertTab, ...props } = useMosaic({ initialTree });
  return (
    <div className="main__content">
      <Mosaic {...props} />
    </div>
  );
};
