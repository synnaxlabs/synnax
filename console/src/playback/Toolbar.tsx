import { Icon } from "@synnaxlabs/media";
import { Align } from "@synnaxlabs/pluto";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";

export const Content = () => {
  return (
    <Align.Space className={CSS.B("playback-toolbar")}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Playback />}>Playback</ToolbarTitle>
      </ToolbarHeader>
    </Align.Space>
  );
};

export const Toolbar: Layout.NavDrawerItem = {
  key: "playback",
  icon: <Icon.Playback />,
  content: <Content />,
  tooltip: "Playback",
  initialSize: 150,
  minSize: 100,
  maxSize: 300,
};
