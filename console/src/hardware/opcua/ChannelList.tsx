import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Align, Channel, Form, Header, Status } from "@synnaxlabs/pluto";
import { Button } from "@synnaxlabs/pluto/button";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { xy } from "@synnaxlabs/x";
import { nanoid } from "nanoid";

import { CSS } from "@/css";
import { type NIChannel } from "@/hardware/ni/types";
import { type ReadChannel } from "@/hardware/opcua/types";

export interface ChannelListProps {
  path: string;
  onSelect: (keys: string[], index: number) => void;
  selected: string[];
}

export const ChannelList = ({
  path,
  selected,
  onSelect,
}: ChannelListProps): ReactElement => {
  const { value, push } = Form.useFieldArray<ReadChannel>({ path });

  const handleAdd = (): void => {
    push({
      key: nanoid(),
      channel: 0,
      nodeId: 0,
    });
  };

  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <Header.Header level="h3">
        <Header.Title weight={500}>Channels</Header.Title>
        <Header.Actions>
          {[
            {
              key: "add",
              onClick: handleAdd,
              children: <Icon.Add />,
            },
          ]}
        </Header.Actions>
      </Header.Header>
      <List.List<string, NIChannel> data={value}>
        <List.Selector<string, NIChannel>
          value={selected}
          allowNone={false}
          allowMultiple={true}
          onChange={(keys, { clickedIndex }) =>
            clickedIndex != null && onSelect(keys, clickedIndex)
          }
          replaceOnSingle
        >
          <List.Core<string, NIChannel> grow>
            {(props) => <ChannelListItem {...props} path={path} />}
          </List.Core>
        </List.Selector>
      </List.List>
    </Align.Space>
  );
};

export const ChannelListItem = ({
  path,
  ...props
}: List.ItemProps<string, ReadChannel> & {
  path: string;
}): ReactElement => {
  const { entry } = props;
  const ctx = Form.useContext();
  const childValues = Form.useChildFieldValues<NIChannel>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  if (childValues == null) return <></>;
  const channelName = Channel.useName(entry.channel);
  return (
    <List.ItemFrame
      {...props}
      entry={childValues}
      justify="spaceBetween"
      align="center"
    >
      <Align.Space direction="y" size="small">
        <Align.Space direction="x">
          <Text.Text level="p" weight={500} shade={9}>
            {channelName}
          </Text.Text>
        </Align.Space>
      </Align.Space>
      <Button.Toggle
        checkedVariant="outlined"
        uncheckedVariant="outlined"
        value={entry.enabled}
        size="small"
        onClick={(e) => e.stopPropagation()}
        onChange={(v) => {
          ctx.set({ path: `${path}.${props.index}.enabled`, value: v });
        }}
        tooltip={
          <Text.Text level="small" style={{ maxWidth: 300 }}>
            Data acquisition for this channel is{" "}
            {entry.enabled ? "enabled" : "disabled"}. Click to
            {entry.enabled ? " disable" : " enable"} it.
          </Text.Text>
        }
      >
        <Status.Text
          variant={entry.enabled ? "success" : "disabled"}
          level="small"
          align="center"
        >
          {entry.enabled ? "Enabled" : "Disabled"}
        </Status.Text>
      </Button.Toggle>
    </List.ItemFrame>
  );
};
