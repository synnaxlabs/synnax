import { Space } from "@synnaxlabs/pluto";
import BottomNavbar from "./BottomNavbar";
import "./index.css";
import LeftNavbar from "./LeftNavbar";
import RightNavbar from "./RightNavbar";
import TopNavbar from "./TopNavbar";
import { LayoutMosaic } from "@/features/layout";

export const MainLayout = () => {
  return (
    <Space
      direction="vertical"
      size="large"
      className="void-main__container"
      empty
    >
      <TopNavbar />
      <Space direction="horizontal" size="large" grow empty>
        <LeftNavbar />
        <div className="void-main-layout__content">
          <LayoutMosaic />
        </div>
        <RightNavbar />
      </Space>
      <BottomNavbar />
    </Space>
  );
};
