import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";

import { noop, Option, Row, Section, StaticTooltip } from "@/options/common";
import {
  ChannelMenu,
  ConfirmDelete,
  DataTypeSelect,
  StaticDropdown,
  Toast,
} from "@/options/samples";

export const RadiusSample = () => (
  <Flex.Box y gap="large">
    <Row label="Small 24">
      <div style={{ width: 120 }}>
        <Input.Text size="small" value="limit" onChange={noop} full="x" />
      </div>
      <Button.Button size="small" variant="outlined">
        Cancel
      </Button.Button>
      <Button.Button size="small" variant="filled">
        Save
      </Button.Button>
      <Button.Button size="small" variant="text">
        <Icon.Add />
        Add row
      </Button.Button>
    </Row>
    <Row label="Medium 28">
      <div style={{ width: 240 }}>
        <Input.Text value="ox_tank_pressure_transducer_01" onChange={noop} full="x" />
      </div>
      <div style={{ width: 150 }}>
        <DataTypeSelect />
      </div>
      <Button.Button variant="outlined">Cancel</Button.Button>
      <Button.Button variant="filled" status="error">
        Delete
      </Button.Button>
    </Row>
    <Row label="Large 36">
      <Button.Button size="large" variant="outlined">
        Cancel
      </Button.Button>
      <Button.Button size="large" variant="filled">
        Create channel
      </Button.Button>
    </Row>
    <Row label="Floating surfaces" align="start">
      <Flex.Box y gap="large">
        <StaticTooltip>Create channel</StaticTooltip>
        <StaticDropdown />
      </Flex.Box>
      <ChannelMenu />
      <Flex.Box y gap="large">
        <Toast
          name="ox_tank_pressure_transducer_01"
          message="Copied link to clipboard"
        />
        <ConfirmDelete />
      </Flex.Box>
    </Row>
  </Flex.Box>
);

export const RadiusSection = () => (
  <Section
    id="radius"
    n={1}
    title="Corner radius ladder"
    summary="Sets one corner radius per control size and one per overlay class, so a Cancel and a Delete in one footer match."
  >
    <Option
      letter="A"
      title="Today"
      desc="Outlined 4 px, filled 6 px at every size. Tooltip 6, dropdown 6, toast 9, menu 12 (items 4), modal 12."
    >
      <RadiusSample />
    </Option>
    <Option
      letter="B"
      title="Tight"
      className="r-b"
      desc="Controls 4 px at every size and variant. Tooltip 4. Menus, dropdowns, toasts, palette, modals 10, menu items 4."
    >
      <RadiusSample />
    </Option>
    <Option
      letter="C"
      title="Stepped"
      className="r-c"
      desc="Controls 4 / 6 / 8 px at small / medium / large. Tooltip 6. Large overlays 12, items 6 (12 minus 6 padding)."
    >
      <RadiusSample />
    </Option>
    <Option
      letter="D"
      title="Soft"
      className="r-d"
      desc="Controls 6 / 8 / 10 px at small / medium / large. Tooltip 8. Large overlays 14, items 8 (14 minus 6 padding)."
    >
      <RadiusSample />
    </Option>
  </Section>
);
