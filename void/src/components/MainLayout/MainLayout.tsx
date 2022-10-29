import { Space } from "@synnaxlabs/pluto";
import { LayoutMosaic } from "@/features/layout";
import { NavBottom } from "./NavBottom";
import { NavLeft } from "./NavLeft";
import { NavRight } from "./NavRight";
import { NavTop } from "./NavTop";
import "./MainLayout.css";
import { ClusterProvider } from "@/features/cluster";

export const MainLayout = () => {
  return (
    <ClusterProvider>
      <Space
        direction="vertical"
        size="large"
        className="void-main__container"
        empty
      >
        <NavTop />
        <Space direction="horizontal" size="large" grow empty>
          <NavLeft />
          <div className="void-main-layout__content">
            <LayoutMosaic />
          </div>
          <NavRight />
        </Space>
        <NavBottom />
      </Space>
    </ClusterProvider>
  );
};
