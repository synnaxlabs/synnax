import { Space, Tabs } from "@synnaxlabs/pluto";

import { VisToolbarTitle } from "../../../components";

import { LinePlotDataControls } from "./LinePlotDataControls";
import { ControlledLineVisProps } from "./types";

import { ToolbarHeader } from "@/components";

export const LinePlotToolBar = (props: ControlledLineVisProps): JSX.Element => {
  const tabProps = Tabs.useStatic({
    tabs: [
      {
        tabKey: "data",
        title: "Data",
        content: <LinePlotDataControls {...props} />,
      },
    ],
  });

  return (
    <Space>
      <Tabs.Provider value={tabProps}>
        <ToolbarHeader>
          <VisToolbarTitle />
          <Tabs.Selector style={{ borderBottom: "none" }} />
        </ToolbarHeader>
        <Tabs.Content />
      </Tabs.Provider>
    </Space>
  );
};
