import { Flex } from "@synnaxlabs/lyra/flex";

import { Option, Row, Section } from "@/options/common";
import { RadiusSample } from "@/options/sections/Radius";
import { Panel } from "@/options/sections/TabStyle";

export const BaselineSection = () => (
  <Section
    id="baseline"
    title="Baseline: today with font smoothing off"
    summary="Today's components as the Console ships them, with the font smoothing typo still in place. Every section below turns smoothing on, its option A included."
    className="opt-smoothing-off"
  >
    <Option
      letter="0"
      title="Today, smoothing off"
      desc="The only render on this page without antialiased smoothing."
    >
      <Flex.Box y gap="huge">
        <RadiusSample />
        <Row label="Document tabs">
          <Panel />
        </Row>
      </Flex.Box>
    </Option>
  </Section>
);
