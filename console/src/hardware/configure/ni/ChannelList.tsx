import { type ReactElement } from "react";

import { Align, Channel, Form, Header, Status } from "@synnaxlabs/pluto";
import { Button } from "@synnaxlabs/pluto/button";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";

import { CSS } from "@/css";
import { CHANNEL_TYPE_DISPLAY, type NIChannel } from "@/hardware/configure/ni/types";

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
  const { value } = Form.useFieldArray<NIChannel>({ path });
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <Header.Header level="h3">
        <Header.Title weight={500}>Channels</Header.Title>
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
}: List.ItemProps<string, NIChannel> & {
  path: string;
}): ReactElement => {
  const { entry } = props;
  const hasLine = "line" in entry;
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
          <Text.Text level="p" weight={500} shade={6} style={{ width: "3rem" }}>
            {childValues.port} {hasLine && `/${entry.line}`}
          </Text.Text>
          <Text.Text level="p" weight={500} shade={9}>
            {channelName}
          </Text.Text>
        </Align.Space>
        <Text.Text level="p" shade={6}>
          {CHANNEL_TYPE_DISPLAY[entry.type]}
        </Text.Text>
      </Align.Space>
      <Button.Toggle
        checkedVariant="text"
        uncheckedVariant="text"
        value={entry.enabled}
        size="small"
        onClick={(e) => e.stopPropagation()}
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
