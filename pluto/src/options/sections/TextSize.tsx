import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers } from "@synnaxlabs/lyra/triggers";

import { noop, Option, Row, SAVE, Section } from "@/options/common";
import { ChannelMenu, Field, Toast } from "@/options/samples";

const ROWS = [
  ["start_time", "2026-10-27T13:32:20Z"],
  ["limit", "100"],
];

interface SampleProps {
  menuLevel: "small" | "p";
  menuGap?: "small";
}

const TextSizeSample = ({ menuLevel, menuGap }: SampleProps) => (
  <Flex.Box x gap="huge" align="start">
    <Row label="Context menu">
      <ChannelMenu level={menuLevel} gap={menuGap} />
    </Row>
    <Flex.Box y gap="large" style={{ width: 300 }}>
      <Row label="Field label, help, and error text">
        <Flex.Box y gap="medium" style={{ width: 300 }}>
          <Field label="Name" required helpText="Name is required" status="error">
            <Input.Text value="" onChange={noop} placeholder="Name" status="error" />
          </Field>
          <Field label="Sample rate" helpText="Samples per second, from 1 to 50 kHz.">
            <Input.Numeric value={1000} onChange={noop} endContent="Hz" />
          </Field>
        </Flex.Box>
      </Row>
      <Row label="Table headings">
        <div style={{ width: 300 }}>
          <Input.Table value={ROWS} onChange={noop}>
            <Input.TableColumn name="Parameter" type="string" />
            <Input.TableColumn name="Value" type="string" />
          </Input.Table>
        </div>
      </Row>
    </Flex.Box>
    <Flex.Box y gap="large">
      <Row label="Toast description">
        <Toast
          variant="error"
          name="ox_tank_pressure_transducer_01"
          message="Failed to delete channel"
          description="The channel is the index of 4 other channels. Delete or re-index them first, then try again."
        />
      </Row>
      <Row label="Small button, keycap, small text">
        <Flex.Box y gap="medium">
          <Flex.Box x gap="medium" align="center">
            <Button.Button size="small" variant="text" textColor={9}>
              <Icon.Add />
              Add row
            </Button.Button>
            <Button.Button size="small" variant="outlined">
              Configure
            </Button.Button>
            <Triggers.Text level="small" trigger={SAVE}>
              Save
            </Triggers.Text>
          </Flex.Box>
          <Text.Text level="small" color={9}>
            Task not deployed
          </Text.Text>
        </Flex.Box>
      </Row>
    </Flex.Box>
  </Flex.Box>
);

export const TextSizeSection = () => (
  <Section
    id="text-size"
    n={3}
    title="Smallest text size"
    summary="Sets the floor for small text (labels, help and error text, table headings, toast descriptions, small buttons) and the size of context menu items."
  >
    <Option
      letter="A"
      title="Today"
      desc="Small text at 11 px. Context menus at 11 px in 24 px items."
    >
      <TextSizeSample menuLevel="small" menuGap="small" />
    </Option>
    <Option
      letter="B"
      title="11 px floor, Lyra menus"
      className="t-b"
      desc="Small text stays 11 px. Context menus return to Lyra's own 13 px in 28 px items."
    >
      <TextSizeSample menuLevel="p" />
    </Option>
    <Option
      letter="C"
      title="12 px floor, Lyra menus"
      className="t-c"
      desc="Small text rises to 12 px (16 px line). Context menus at 13 px in 28 px items."
    >
      <TextSizeSample menuLevel="p" />
    </Option>
    <Option
      letter="D"
      title="12 px floor, compact menus"
      className="t-d"
      desc="Small text rises to 12 px (16 px line). Context menus at 12 px in 24 px items."
    >
      <TextSizeSample menuLevel="small" menuGap="small" />
    </Option>
  </Section>
);
