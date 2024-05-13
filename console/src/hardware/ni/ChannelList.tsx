import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Align, Channel, Form, Header, Menu, Status } from "@synnaxlabs/pluto";
import { Button } from "@synnaxlabs/pluto/button";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { deep, xy } from "@synnaxlabs/x";
import { nanoid } from "nanoid";

import { CSS } from "@/css";
import {
  AI_CHANNEL_TYPE_NAMES,
  CHANNEL_TYPE_DISPLAY,
  ZERO_AI_CHANNELS,
  type NIChannel,
} from "@/hardware/ni/types";

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
  const { value, push } = Form.useFieldArray<NIChannel>({ path });

  const handleAdd = (): void => {
    const availablePort = Math.max(0, ...value.map((v) => v.port)) + 1;
    push({
      ...deep.copy(ZERO_AI_CHANNELS["ai_accel"]),
      port: availablePort,
      key: nanoid(),
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
              size: "large",
            },
          ]}
        </Header.Actions>
      </Header.Header>
      <Menu.ContextMenu>
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
      </Menu.ContextMenu>
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
  const ctx = Form.useContext();
  const childValues = Form.useChildFieldValues<NIChannel>({
    path: `${path}.${props.index}`,
    optional: true,
  });
  const channelName = Channel.useName(childValues.channel, "No Synnax Channel");
  const channelValid =
    Form.useField<number>({
      path: `${path}.${props.index}.channel`,
    }).status.variant === "success";
  const portValid =
    Form.useField<number>({
      path: `${path}.${props.index}.port`,
    }).status.variant === "success";
  if (childValues == null) return <></>;
  return (
    <List.ItemFrame
      {...props}
      entry={childValues}
      justify="spaceBetween"
      align="center"
    >
      <Align.Space direction="y" size="small">
        <Align.Space direction="x">
          <Text.Text
            level="p"
            weight={500}
            shade={6}
            style={{ width: "3rem" }}
            color={portValid ? undefined : "var(--pluto-error-z)"}
          >
            {childValues.port} {hasLine && `/${entry.line}`}
          </Text.Text>
          <Text.Text
            level="p"
            weight={500}
            shade={9}
            color={(() => {
              if (channelName === "No Synnax Channel") return "var(--pluto-warning-z)";
              else if (channelValid) return undefined;
              return "var(--pluto-error-z)";
            })()}
          >
            {channelName}
          </Text.Text>
        </Align.Space>
        <Text.Text level="p" shade={6}>
          {AI_CHANNEL_TYPE_NAMES[childValues.type]}
        </Text.Text>
      </Align.Space>
      <Button.Toggle
        checkedVariant="outlined"
        uncheckedVariant="outlined"
        value={childValues.enabled}
        size="small"
        onClick={(e) => e.stopPropagation()}
        onChange={(v) => {
          console.log("setting enabled", v);
          ctx.set({ path: `${path}.${props.index}.enabled`, value: v });
        }}
        tooltip={
          <Text.Text level="small" style={{ maxWidth: 300 }}>
            Data acquisition for this channel is{" "}
            {childValues.enabled ? "enabled" : "disabled"}. Click to
            {childValues.enabled ? " disable" : " enable"} it.
          </Text.Text>
        }
      >
        <Status.Text
          variant={childValues.enabled ? "success" : "disabled"}
          level="small"
          align="center"
        >
          {childValues.enabled ? "Enabled" : "Disabled"}
        </Status.Text>
      </Button.Toggle>
    </List.ItemFrame>
  );
};
