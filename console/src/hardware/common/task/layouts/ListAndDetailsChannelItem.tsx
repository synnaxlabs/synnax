import { Align, List, Text } from "@synnaxlabs/pluto";
import { type Key, type Keyed } from "@synnaxlabs/x";

import { Common } from "@/hardware/common";

export interface ListAndDetailsChannelItemProps<K extends Key, E extends Keyed<K>>
  extends List.ItemProps<K, E> {
  port: string | number;
  portMaxChars: number;
  canTare: boolean;
  channel: number;
  onTare: (channel: number) => void;
  isSnapshot: boolean;
  path: string;
  hasTareButton: boolean;
  name?: string;
}
export const ListAndDetailsChannelItem = <K extends Key, E extends Keyed<K>>({
  port,
  portMaxChars,
  canTare,
  onTare,
  isSnapshot,
  path,
  hasTareButton,
  channel,
  name,
  ...rest
}: ListAndDetailsChannelItemProps<K, E>) => (
  <List.ItemFrame
    {...rest}
    justify="spaceBetween"
    align="center"
    style={{ padding: "1.25rem 2rem" }}
  >
    <Align.Space direction="x" size="small">
      <Text.Text
        level="p"
        shade={6}
        weight={500}
        style={{ width: `${portMaxChars * 1.25}rem` }}
      >
        {port}
      </Text.Text>
      {name != null ? (
        <Text.Text
          level="p"
          shade={7}
          weight={450}
          style={{
            maxWidth: 150,
            flexGrow: 1,
            textOverflow: "ellipsis",
            overflow: "hidden",
          }}
          noWrap
        >
          {name}
        </Text.Text>
      ) : (
        <Common.Task.ChannelName channel={channel} />
      )}
    </Align.Space>
    <Align.Pack direction="x" align="center" size="small">
      {hasTareButton && (
        <Common.Task.TareButton disabled={!canTare} onTare={() => onTare(channel)} />
      )}
      <Common.Task.EnableDisableButton
        path={`${path}.enabled`}
        isSnapshot={isSnapshot}
      />
    </Align.Pack>
  </List.ItemFrame>
);
