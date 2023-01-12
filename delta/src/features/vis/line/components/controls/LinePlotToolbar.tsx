import { Space, Tabs } from "@synnaxlabs/pluto";

import { VisToolbarTitle } from "../../../components";

import { LinePlotChannelControls } from "./LinePlotChannelControls";
import { ControlledLineVisProps } from "./types";

import { ToolbarHeader } from "@/components";

export const LinePlotToolBar = (props: ControlledLineVisProps): JSX.Element => {
  const tabProps = Tabs.useStatic({
    tabs: [
      {
        tabKey: "channels",
        title: "Channels",
        content: <LinePlotChannelControls {...props} />,
      },
    ],
  });

  return (
    <Space empty>
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
