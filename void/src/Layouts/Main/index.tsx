import { MultiResizable, Space } from "@synnaxlabs/pluto";
import Plot from "../Plot/Plot";
import BottomNavbar from "./BottomNavbar";
import "./index.css";
import LeftNavbar from "./LeftNavbar";
import RightNavbar from "./RightNavbar";
import TopNavbar from "./TopNavbar";

export default function Layout() {
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
  return (
    <div className="main__content">
      <MultiResizable direction="vertical">
        <MultiResizable direction="horizontal">
          <Plot />
          <Plot />
        </MultiResizable>
        <Plot />
      </MultiResizable>
    </div>
  );
};
