import "@/hardware/task/common/common.css";

import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Form,
  Header,
  Menu,
  Status,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { Key, Keyed } from "@synnaxlabs/x";

import { Menu as CMenu } from "@/components/menu";
import { CSS } from "@/css";

export interface ControlsProps {
  onStartStop: () => void;
  startingOrStopping: boolean;
  onConfigure: () => void;
  configuring: boolean;
  state?: task.State<{ running?: boolean; message?: string }>;
}

const CONFIGURE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export interface ChannelListEmptyContentProps {
  onAdd: () => void;
}

export const ChannelListEmptyContent = ({ onAdd }: ChannelListEmptyContentProps) => (
  <Align.Space direction="y" style={{ height: "100%" }}>
    <Align.Center direction="y">
      <Text.Text level="p">No channels in task.</Text.Text>
      <Text.Link level="p" onClick={onAdd}>
        Add a channel
      </Text.Link>
    </Align.Center>
  </Align.Space>
);

interface ChannelListContextMenuProps<T> {
  keys: string[];
  value: T[];
  onSelect: (keys: string[], index: number) => void;
  remove: (indices: number[]) => void;
  onDuplicate: (indices: number[]) => void;
  path: string;
}

export const ChannelListContextMenu = <
  K extends Key,
  T extends Keyed<K> & { enabled: boolean },
>({
  keys,
  value,
  onSelect,
  remove,
  onDuplicate,
  path,
}: ChannelListContextMenuProps<T>) => {
  const methods = Form.useContext();
  const indices = keys.map((k) => value.findIndex((v) => v.key === k));
  const handleRemove = () => {
    remove(indices);
    onSelect([], -1);
  };
  const handleDuplicate = () => onDuplicate(indices);
  const handleDisable = () =>
    value.forEach((_, i) => {
      if (!indices.includes(i)) return;
      methods.set(`${path}.${i}.enabled`, false);
    });
  const handleEnable = () =>
    value.forEach((_, i) => {
      if (!indices.includes(i)) return;
      methods.set(`${path}.${i}.enabled`, true);
    });
  const allowDisable = indices.some((i) => value[i].enabled);
  const allowEnable = indices.some((i) => !value[i].enabled);
  return (
    <Menu.Menu
      onChange={{
        remove: handleRemove,
        duplicate: handleDuplicate,
        disable: handleDisable,
        enable: handleEnable,
      }}
      level="small"
    >
      <Menu.Item itemKey="remove" startIcon={<Icon.Close />}>
        Remove
      </Menu.Item>
      <Menu.Item itemKey="duplicate" startIcon={<Icon.Copy />}>
        Duplicate
      </Menu.Item>
      <Menu.Divider />
      {allowDisable && (
        <Menu.Item itemKey="disable" startIcon={<Icon.Disable />}>
          Disable
        </Menu.Item>
      )}
      {allowEnable && (
        <Menu.Item itemKey="enable" startIcon={<Icon.Enable />}>
          Enable
        </Menu.Item>
      )}
      {(allowEnable || allowDisable) && <Menu.Divider />}
      <CMenu.HardReloadItem />
    </Menu.Menu>
  );
};

export const Controls = ({
  state,
  onStartStop,
  startingOrStopping,
  onConfigure,
  configuring,
}: ControlsProps) => (
  <Align.Space direction="x" className={CSS.B("task-controls")} justify="spaceBetween">
    <Align.Space
      className={CSS.B("task-state")}
      direction="x"
      style={{
        borderRadius: "1rem",
        border: "var(--pluto-border)",
        padding: "2rem",
        width: "100%",
      }}
    >
      {state?.details?.message != null && (
        <Status.Text variant={state?.variant as Status.Variant}>
          {state?.details?.message}
        </Status.Text>
      )}
    </Align.Space>
    <Align.Space
      direction="x"
      style={{
        borderRadius: "1rem",
        border: "var(--pluto-border)",
        padding: "2rem",
      }}
      justify="end"
    >
      <Align.Space direction="y">
        <Button.Icon
          loading={startingOrStopping}
          disabled={startingOrStopping || state == null}
          onClick={onStartStop}
          variant="outlined"
        >
          {state?.details?.running === true ? <Icon.Pause /> : <Icon.Play />}
        </Button.Icon>
      </Align.Space>
      <Align.Space direction="y">
        <Button.Button
          loading={configuring}
          disabled={configuring}
          onClick={onConfigure}
          triggers={[CONFIGURE_TRIGGER]}
          tooltip={
            <Align.Space direction="x" align="center" size="small">
              <Triggers.Text shade={7} level="small" trigger={CONFIGURE_TRIGGER} />
              <Text.Text shade={7} level="small">
                To Configure
              </Text.Text>
            </Align.Space>
          }
        >
          Configure
        </Button.Button>
      </Align.Space>
    </Align.Space>
  </Align.Space>
);

export const ChannelListHeader = ({ onAdd }: ChannelListEmptyContentProps) => (
  <Header.Header level="h4">
    <Header.Title weight={500}>Channels</Header.Title>
    <Header.Actions>
      {[
        {
          key: "add",
          onClick: onAdd,
          children: <Icon.Add />,
          size: "large",
        },
      ]}
    </Header.Actions>
  </Header.Header>
);

export interface EnableDisableButtonProps {
  value: boolean;
  onChange: (v: boolean) => void;
}

export const EnableDisableButton = ({
  value,
  onChange: onClick,
}: EnableDisableButtonProps) => (
  <Button.Toggle
    checkedVariant="outlined"
    uncheckedVariant="outlined"
    value={value}
    size="small"
    onClick={(e) => e.stopPropagation()}
    onChange={onClick}
    tooltip={
      <Text.Text level="small" style={{ maxWidth: 300 }}>
        Data acquisition for this channel is {value ? "enabled" : "disabled"}. Click to
        {value ? " disable" : " enable"} it.
      </Text.Text>
    }
  >
    <Status.Text variant={value ? "success" : "disabled"} level="small" align="center">
      {value ? "Enabled" : "Disabled"}
    </Status.Text>
  </Button.Toggle>
);
