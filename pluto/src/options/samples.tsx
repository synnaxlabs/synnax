// Scratch page for visual options. Not for merge. Shared Synnax samples built from real
// Lyra components.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement, type ReactNode } from "react";

import {
  cls,
  ESCAPE,
  noop,
  now,
  SAVE,
  SaveHint,
  StaticMenu,
  StaticModal,
} from "@/options/common";

export const DATA_TYPES = [
  "float64",
  "float32",
  "int64",
  "int32",
  "int16",
  "uint8",
  "timestamp",
  "string",
];

/** A closed select trigger holding a value, as in the channel create modal. */
export const DataTypeSelect = ({
  value = "float32",
  style,
}: {
  value?: string;
  style?: React.CSSProperties;
}) => (
  <Select.Simple<string>
    resourceName="data type"
    value={value}
    onChange={noop}
    icon={<Icon.Binary />}
    style={style}
    triggerProps={{ style }}
  >
    {DATA_TYPES.map((d) => (
      <Select.Item key={d} itemKey={d}>
        {d}
      </Select.Item>
    ))}
  </Select.Simple>
);

/** A select dropdown frozen open, built from the Select parts the live one uses. */
export const StaticDropdown = ({
  value = "float32",
  className,
}: {
  value?: string;
  className?: string;
}) => (
  <Select.Frame<string> value={value} onChange={noop}>
    <Flex.Box
      y
      pack
      empty
      background={0}
      rounded="small"
      className={cls(
        "pluto-dialog__dialog",
        "pluto-select__dialog",
        "pluto--visible",
        "opt-static",
        "opt-dropdown",
        className,
      )}
      style={{ width: 200 }}
    >
      <Select.Search placeholder="Search data types..." />
      <Select.List bordered borderColor={6} grow rounded full="x" displayItems={5}>
        {DATA_TYPES.slice(0, 5).map((d) => (
          <Select.Item key={d} itemKey={d}>
            {d}
          </Select.Item>
        ))}
      </Select.List>
    </Flex.Box>
  </Select.Frame>
);

/** The channel tree context menu, item for item. */
export const ChannelMenuItems = (): ReactElement => (
  <>
    <Menu.Item itemKey="rename">
      <Icon.Rename />
      Rename
    </Menu.Item>
    <Menu.Item itemKey="group">
      <Icon.Group />
      Group selection
    </Menu.Item>
    <Menu.Divider />
    <Menu.Item itemKey="alias">
      <Icon.Rename />
      Set alias under Hotfire 14
    </Menu.Item>
    <Menu.Divider />
    <Menu.Item itemKey="link">
      <Icon.Link />
      Copy link
    </Menu.Item>
    <Menu.Item itemKey="props">
      <Icon.Copy />
      Copy properties
    </Menu.Item>
    <Menu.Divider />
    <Menu.Item itemKey="delete">
      <Icon.Delete />
      Delete
    </Menu.Item>
    <Menu.Divider />
    <Menu.Item itemKey="hardReload">
      <Icon.Refresh />
      Reload Console
    </Menu.Item>
  </>
);

export const ChannelMenu = ({
  level = "small",
  gap = "small",
  className,
}: {
  level?: "small" | "p";
  gap?: "small" | "medium";
  className?: string;
}) => (
  <StaticMenu level={level} gap={gap} className={className}>
    <ChannelMenuItems />
  </StaticMenu>
);

export const RENAME: Triggers.Trigger = ["Control", "E"];
export const FOCUS: Triggers.Trigger = ["Control", "L"];
export const CLOSE: Triggers.Trigger = ["Control", "W"];

/** The panel tab context menu, which carries shortcuts on a focused tab. */
export const TabMenu = ({ className }: { className?: string }) => (
  <StaticMenu className={className}>
    <Menu.Item itemKey="rename" trigger={RENAME} triggerIndicator>
      <Icon.Rename />
      Rename
    </Menu.Item>
    <Menu.Item itemKey="focus" triggerIndicator={FOCUS}>
      <Icon.Focus />
      Focus
    </Menu.Item>
    <Menu.Divider />
    <Menu.Item itemKey="splitX">
      <Icon.SplitX />
      Split horizontally
    </Menu.Item>
    <Menu.Item itemKey="splitY">
      <Icon.SplitY />
      Split vertically
    </Menu.Item>
    <Menu.Divider />
    <Menu.Item itemKey="close" triggerIndicator={CLOSE}>
      <Icon.Close />
      Close
    </Menu.Item>
  </StaticMenu>
);

interface ToastProps {
  variant?: Status.Variant;
  name?: string;
  message: string;
  description?: string;
  className?: string;
}

export const Toast = ({
  variant = "success",
  name = "",
  message,
  description = "",
  className,
}: ToastProps) => (
  <Status.Notification
    className={cls("opt-toast", className)}
    status={{
      key: message,
      variant,
      name,
      message,
      description,
      time: now,
      count: 1,
    }}
    silence={noop}
  />
);

/** The delete-channel confirmation as the Console renders it today. */
export const ConfirmDelete = ({
  className,
  name = "ox_tank_pressure_transducer_01",
}: {
  className?: string;
  name?: string;
}) => (
  <StaticModal className={cls("opt-confirm", className)}>
    <Modal.Header icon={<Icon.Channel />}>{["Channel", "Delete"]}</Modal.Header>
    <Modal.Body>
      <Text.Text level="h3" weight={450}>
        Are you sure you want to delete {name}?
      </Text.Text>
      <Text.Text weight={450}>This action cannot be undone.</Text.Text>
    </Modal.Body>
    <Modal.Footer>
      <SaveHint action="Delete" />
      <Nav.Bar.End x align="center">
        <Button.Button variant="outlined" triggerIndicator={ESCAPE}>
          Cancel
        </Button.Button>
        <Button.Button variant="filled" status="error">
          Delete
        </Button.Button>
      </Nav.Bar.End>
    </Modal.Footer>
  </StaticModal>
);

/** A labeled form field, as Form.Field renders it. */
export const Field = ({
  label,
  required,
  children,
  helpText,
  status,
  grow,
}: {
  label: string;
  required?: boolean;
  children: ReactNode;
  helpText?: string;
  status?: Status.Variant;
  grow?: boolean;
}) => (
  <Input.Item
    label={label}
    required={required}
    helpText={helpText}
    status={status}
    padHelpText
    grow={grow}
  >
    {children}
  </Input.Item>
);

export { SAVE };

export interface Command {
  key: string;
  name: string;
  icon: Icon.ReactElement;
  trigger?: Triggers.Trigger;
}

export const OPEN_WINDOW: Triggers.Trigger = ["Control", "O"];

export const CREATE_COMMANDS: Command[] = [
  { key: "channel", name: "Create channel", icon: <Icon.Channel /> },
  { key: "calc", name: "Create calculated channel", icon: <Icon.Calculation /> },
  { key: "lineplot", name: "Create line plot", icon: <Icon.LinePlot /> },
  { key: "schematic", name: "Create schematic", icon: <Icon.Schematic /> },
  { key: "range", name: "Create range", icon: <Icon.Range /> },
  { key: "table", name: "Create table", icon: <Icon.Table /> },
  { key: "log", name: "Create log", icon: <Icon.Log /> },
  { key: "arc", name: "Create Arc automation", icon: <Icon.Arc /> },
  { key: "http", name: "Create HTTP read task", icon: <Icon.Logo.HTTP /> },
  { key: "status", name: "Create status", icon: <Icon.Status /> },
];

export const KEYCAP_COMMANDS: Command[] = [
  { key: "window", name: "Open new window", icon: <Icon.OpenInNewWindow />, trigger: OPEN_WINDOW },
  { key: "channel", name: "Create channel", icon: <Icon.Channel /> },
  { key: "theme", name: "Change color theme", icon: <Icon.DarkMode /> },
];

export const PaletteItem = ({ cmd, hovered }: { cmd: Command; hovered?: boolean }) => (
  <Select.Item
    itemKey={cmd.key}
    justify="between"
    align="center"
    hovered={hovered}
    className="opt-palette-item"
  >
    <Text.Text gap="medium">
      {cmd.icon}
      {cmd.name}
    </Text.Text>
    {cmd.trigger != null && (
      <Flex.Box x>
        <Triggers.Text trigger={cmd.trigger} level="small" color={9} />
      </Flex.Box>
    )}
  </Select.Item>
);

/** The command list of the palette, frozen with its first item under the pointer. */
export const PaletteList = ({ commands }: { commands: Command[] }) => (
  <Select.Frame<string> value="" onChange={noop}>
    <Flex.Box y empty className="opt-palette-list">
      {commands.map((c, i) => (
        <PaletteItem key={c.key} cmd={c} hovered={i === 0} />
      ))}
    </Flex.Box>
  </Select.Frame>
);

const PALETTE_PLACEHOLDER = (
  <>
    <Icon.Search />
    Type to search or {">"} to view commands
  </>
);

interface PaletteProps {
  value?: string;
  commands: Command[];
  className?: string;
  footer?: ReactNode;
}

export const Palette = ({ value = ">Create", commands, className, footer }: PaletteProps) => (
  <StaticModal className={cls("opt-palette", className)}>
    <Input.Search
      className="opt-palette-input"
      placeholder={PALETTE_PLACEHOLDER}
      size="huge"
      flush
      onChange={noop}
      borderColor={8}
      value={value}
      full="x"
    />
    <div className="opt-palette-scroll">
      <PaletteList commands={commands} />
    </div>
    {footer}
  </StaticModal>
);
