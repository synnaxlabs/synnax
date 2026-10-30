import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers } from "@synnaxlabs/lyra/triggers";

import { ESCAPE, Option, Scrim, Section } from "@/options/common";
import { CREATE_COMMANDS, Palette } from "@/options/samples";

const ENTER: Triggers.Trigger = ["Enter"];

const FooterBar = () => (
  <Flex.Box x align="center" justify="between" className="opt-palette-footer">
    <Text.Text color={9} gap="small">
      <Icon.Bolt />
      Commands
    </Text.Text>
    <Flex.Box x align="center" gap="large">
      <Text.Text color={11} gap="small" weight={500}>
        Create channel
        <Triggers.Text trigger={ENTER} level="small" />
      </Text.Text>
      <span className="opt-palette-footer-sep" />
      <Text.Text color={9} gap="small">
        Close
        <Triggers.Text trigger={ESCAPE} level="small" />
      </Text.Text>
    </Flex.Box>
  </Flex.Box>
);

export const PaletteSection = () => (
  <Section
    id="palette"
    n={9}
    title="Palette size and footer"
    summary="Sets the command palette width and whether a 40 px footer bar names the selected command's action and key."
  >
    <Option
      letter="A"
      title="Today"
      desc="95vw up to 1200 px (1200 px on a 1440 px window), 6 px radius, 27 px input, no footer."
    >
      <Scrim>
        <Palette commands={CREATE_COMMANDS} className="pal-today" />
      </Scrim>
    </Option>
    <Option
      letter="B"
      title="660 px, no footer"
      className="pal-b"
      desc="660 px wide, the modal width. Input, items, and radius as today."
    >
      <Scrim>
        <Palette commands={CREATE_COMMANDS} />
      </Scrim>
    </Option>
    <Option
      letter="C"
      title="660 px with footer"
      className="pal-c"
      desc="660 px wide plus a 40 px footer bar that names the selected command's action and key."
    >
      <Scrim>
        <Palette commands={CREATE_COMMANDS} footer={<FooterBar />} />
      </Scrim>
    </Option>
    <Option
      letter="D"
      title="660 px with footer, report frame"
      className="pal-d"
      desc="As C, with the report's frame rule: 16 px input and the 12 px overlay radius."
    >
      <Scrim>
        <Palette commands={CREATE_COMMANDS} footer={<FooterBar />} />
      </Scrim>
    </Option>
  </Section>
);
