import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";

import { noop, Option, Row, Section } from "@/options/common";
import { Panel } from "@/options/sections/TabStyle";

const WEIGHT_TABS = [
  { key: "plot", name: "Hotfire 14 pressures", icon: <Icon.LinePlot /> },
  { key: "schematic", name: "Feed system", icon: <Icon.Schematic /> },
  { key: "status", name: "Status", icon: <Icon.Status /> },
  { key: "http", name: "HTTP read", icon: <Icon.Task /> },
];

const WeightSample = () => (
  <Flex.Box y gap="large">
    <Row label="Buttons: small, medium, large">
      <Button.Button size="small" variant="outlined">
        Configure
      </Button.Button>
      <Button.Button size="small" variant="text" textColor={9}>
        <Icon.Add />
        Add row
      </Button.Button>
      <Button.Button variant="outlined">Cancel</Button.Button>
      <Button.Button variant="filled">Create channel</Button.Button>
      <Button.Button variant="text">
        <Icon.Range />
        Open range explorer
      </Button.Button>
      <Button.Button size="large" variant="filled">
        Sign in
      </Button.Button>
    </Row>
    <Row label="Document tabs">
      <Panel tabs={WEIGHT_TABS} selected="plot" width={560} content={false} />
    </Row>
    <Row label="Segmented control (line plot toolbar)">
      <Select.Buttons value="lines" onChange={noop}>
        <Select.Item itemKey="data">Data</Select.Item>
        <Select.Item itemKey="lines">Lines</Select.Item>
        <Select.Item itemKey="axes">Axes</Select.Item>
        <Select.Item itemKey="properties">Properties</Select.Item>
        <Select.Item itemKey="rules">Rules</Select.Item>
      </Select.Buttons>
      <Select.Buttons value="x" onChange={noop} variant="outlined">
        <Select.Item itemKey="x">Horizontal</Select.Item>
        <Select.Item itemKey="y">Vertical</Select.Item>
      </Select.Buttons>
    </Row>
  </Flex.Box>
);

export const WeightSection = () => (
  <Section
    id="weight"
    n={5}
    title="Label weight"
    summary="Sets one weight for every control label: buttons, tabs, and segmented controls, at every size and variant."
  >
    <Option
      letter="A"
      title="Today"
      desc="Buttons at 400 (large at 450). Tab names at 400, the focused tab at 500. Segmented control at 400."
    >
      <WeightSample />
    </Option>
    <Option letter="B" title="400" className="w-400" desc="Every control label at 400.">
      <WeightSample />
    </Option>
    <Option letter="C" title="450" className="w-450" desc="Every control label at 450.">
      <WeightSample />
    </Option>
    <Option letter="D" title="500" className="w-500" desc="Every control label at 500.">
      <WeightSample />
    </Option>
  </Section>
);
