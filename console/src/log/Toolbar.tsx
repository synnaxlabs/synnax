import { channel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Channel, Input } from "@synnaxlabs/pluto";
import { ReactElement } from "react";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Layout } from "@/layout";
import { useSelect } from "@/log/selectors";
import { setChannels } from "@/log/slice";

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement => {
  const d = useDispatch();
  const { name } = Layout.useSelectRequired(layoutKey);
  const state = useSelect(layoutKey);
  const handleChannelChange = (v: channel.Key[]) => {
    d(setChannels({ key: layoutKey, channels: v }));
  };
  return (
    <>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Log />}>{name}</ToolbarTitle>
      </ToolbarHeader>
      <Align.Space style={{ padding: "2rem", width: "100%" }}>
        <Input.Item direction="x" label="Channels" grow>
          <Channel.SelectMultiple
            value={state.channels}
            onChange={handleChannelChange}
            searchOptions={{ internal: undefined }}
          />
        </Input.Item>
      </Align.Space>
    </>
  );
};
