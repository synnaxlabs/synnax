import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers } from "@synnaxlabs/lyra/triggers";

import {
  ESCAPE,
  Option,
  Row,
  SAVE,
  SaveHint,
  Section,
  StaticTooltip,
} from "@/options/common";
import { KEYCAP_COMMANDS, OPEN_WINDOW, PaletteList, RENAME, TabMenu } from "@/options/samples";

const SEARCH: Triggers.Trigger = ["Control", "P"];
const COMMAND_TODAY: Triggers.Trigger = ["Control", "Shift", "P"];
// Apple's modifier order: Control, Option, Shift, Command.
const COMMAND_APPLE: Triggers.Trigger = ["Shift", "Control", "P"];

const TipLine = ({ label, trigger }: { label: string; trigger: Triggers.Trigger }) => (
  <Flex.Box x justify="between" align="center" gap="large">
    <Text.Text level="small">{label}</Text.Text>
    <Triggers.Text trigger={trigger} level="small" />
  </Flex.Box>
);

const Footer = ({ today }: { today: boolean }) => (
  <div className="opt-footer-strip">
    <Modal.Footer>
      {today && <SaveHint action="Create" />}
      <Nav.Bar.End x align="center" gap={today ? "small" : "medium"}>
        <Button.Button variant="outlined" triggerIndicator={ESCAPE}>
          Cancel
        </Button.Button>
        <Button.Button variant="filled" triggerIndicator={today ? undefined : SAVE}>
          Create
        </Button.Button>
      </Nav.Bar.End>
    </Modal.Footer>
  </div>
);

const KeycapSample = ({ today }: { today: boolean }) => (
  <Flex.Box y gap="large">
    <Flex.Box x gap="huge" align="start">
      <Row label="Tooltip">
        <StaticTooltip>
          <Flex.Box gap="small">
            <TipLine label="Search" trigger={SEARCH} />
            <TipLine
              label="Command palette"
              trigger={today ? COMMAND_TODAY : COMMAND_APPLE}
            />
          </Flex.Box>
        </StaticTooltip>
      </Row>
      <Row label="Buttons">
        <Flex.Box y gap="medium" align="start">
          <Button.Button variant="outlined" triggerIndicator={OPEN_WINDOW}>
            <Icon.OpenInNewWindow />
            Open new window
          </Button.Button>
          <Button.Button variant="text" triggerIndicator={RENAME}>
            <Icon.Rename />
            Rename
          </Button.Button>
        </Flex.Box>
      </Row>
      <Row label="Context menu (panel tab)">
        <TabMenu />
      </Row>
      <Row label="Palette items">
        <div className="opt-palette-snippet">
          <PaletteList commands={KEYCAP_COMMANDS} />
        </div>
      </Row>
    </Flex.Box>
    <Row label="Modal footer">
      <Footer today={today} />
    </Row>
  </Flex.Box>
);

export const KeycapsSection = () => (
  <Section
    id="keycaps"
    n={4}
    title="Keycap look and placement"
    summary="Sets the keycap size, its one edge cue, and where boxed caps appear against plain glyph text. Menus stay at today's text size so only the caps change. Alternatives also write Esc with a capital and list modifiers in Apple's order (⇧ before ⌘)."
  >
    <Option
      letter="A"
      title="Today"
      desc="15 px cap, 9 px glyph, fill plus border. Boxed caps everywhere; tooltip caps share the tooltip fill. The footer hint sits apart from the button it fires."
    >
      <KeycapSample today />
    </Option>
    <Option
      letter="B"
      title="20 px fill, plain text in lists"
      className="k-b"
      desc="20 px cap, 12 px glyph at 500, fill only (a step darker inside tooltips). Menus and the palette print plain gray glyphs. The shortcut moves into the primary button."
    >
      <KeycapSample today={false} />
    </Option>
    <Option
      letter="C"
      title="20 px thin line, plain text in lists"
      className="k-c"
      desc="20 px cap, 12 px glyph at 500, a hairline and no fill. Menus and the palette print plain gray glyphs. The shortcut moves into the primary button."
    >
      <KeycapSample today={false} />
    </Option>
    <Option
      letter="D"
      title="20 px fill, boxed everywhere"
      className="k-d"
      desc="20 px filled cap, 12 px glyph at 500, boxed in menus and the palette too. The shortcut moves into the primary button."
    >
      <KeycapSample today={false} />
    </Option>
  </Section>
);
