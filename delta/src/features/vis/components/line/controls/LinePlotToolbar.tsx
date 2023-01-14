import { Space, Tab, Tabs } from "@synnaxlabs/pluto";

import { VisToolbarTitle } from "../..";

import { LinePlotChannelControls } from "./LinePlotChannelControls";
import { ControlledLineVisProps } from "./types";

import { ToolbarHeader } from "@/components";

export const LinePlotToolBar = (props: ControlledLineVisProps): JSX.Element => {
  const tabContent = ({ tabKey }: Tab): JSX.Element => {
    switch (tabKey) {
      default:
        return <LinePlotChannelControls {...props} />;
    }
  };

  const tabProps = Tabs.useStatic({
    tabs: [
      {
        tabKey: "channels",
        title: "Channels",
      },
    ],
    content: tabContent,
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
