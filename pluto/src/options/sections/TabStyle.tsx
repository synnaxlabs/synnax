import "@/panel/Mosaic.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";

import { cls, noop, Option, Section } from "@/options/common";

export const LONG_TAB =
  "Hot fire 14 chamber and feed system pressures overlay with thermal limits";

interface DocTab {
  key: string;
  name: string;
  icon: Icon.ReactElement;
}

export const DOC_TABS: DocTab[] = [
  { key: "long", name: LONG_TAB, icon: <Icon.LinePlot /> },
  { key: "schematic", name: "Feed system", icon: <Icon.Schematic /> },
  { key: "status", name: "Status", icon: <Icon.Status /> },
  { key: "log", name: "Stand log", icon: <Icon.Log /> },
  { key: "http", name: "HTTP read", icon: <Icon.Task /> },
];

const PlotStub = () => (
  <svg className="opt-plot" viewBox="0 0 600 150" preserveAspectRatio="none">
    <line x1="40" y1="10" x2="40" y2="130" />
    <line x1="40" y1="130" x2="590" y2="130" />
    <polyline
      className="opt-plot-a"
      points="40,110 90,104 140,96 190,70 240,52 290,48 340,50 390,46 440,44 490,47 540,45 590,46"
    />
    <polyline
      className="opt-plot-b"
      points="40,120 90,118 140,112 190,98 240,86 290,82 340,84 390,80 440,79 490,81 540,80 590,80"
    />
  </svg>
);

interface PanelProps {
  tabs?: DocTab[];
  selected?: string;
  width?: number;
  content?: boolean;
  className?: string;
}

/**
 * A mosaic leaf as the Console draws it: the same Tabs parts, Mosaic.css, a name in a
 * Text, a close button on each tab, and the create button in the sticky cap.
 */
export const Panel = ({
  tabs = DOC_TABS,
  selected = "long",
  width,
  content = true,
  className,
}: PanelProps) => (
  <div
    className={cls("opt-panel", !content && "opt-strip-only", className)}
    style={width != null ? { width } : undefined}
  >
    <div className="pluto-mosaic__leaf">
      <Tabs.Frame value={selected} onChange={noop} grow>
        <Tabs.Selector align="start">
          {tabs.map((t) => (
            <Tabs.Tab
              key={t.key}
              itemKey={t.key}
              className={t.key === selected ? "pluto--alt-color" : undefined}
            >
              {t.icon}
              <Text.Text>{t.name}</Text.Text>
              <Tabs.Close onClick={noop} />
            </Tabs.Tab>
          ))}
          <Flex.Box grow />
          <Flex.Box align="center" justify="center" className="pluto-panel-mosaic__cap">
            <Button.Button
              size="small"
              variant="text"
              className="pluto-panel-mosaic__create"
              textColor={9}
            >
              <Icon.Add />
            </Button.Button>
          </Flex.Box>
        </Tabs.Selector>
        {content && (
          <Tabs.Content itemKey={selected} grow>
            <PlotStub />
          </Tabs.Content>
        )}
      </Tabs.Frame>
    </div>
  </div>
);

export const TabStyleSection = () => (
  <Section
    id="tabs"
    n={7}
    title="Tab style"
    summary="Sets one look for document tabs and whether the active tab joins its panel. The long name is the active tab in an 820 px panel."
  >
    <Option
      letter="A"
      title="Today"
      desc="Rounded tabs; the active one tinted with primary and detached from the panel. No width cap on the active tab; the others shrink to 66 px, then the strip scrolls."
    >
      <Panel />
    </Option>
    <Option
      letter="B"
      title="Joined card"
      className="tabs-b"
      desc="The strip sits one step darker. The active tab takes the panel fill and opens into it. Active tab up to 240 px, others at least 84 px."
    >
      <Panel />
    </Option>
    <Option
      letter="C"
      title="Joined card with accent"
      className="tabs-c"
      desc="As B, plus a 2 px primary line on the active tab's top edge."
    >
      <Panel />
    </Option>
    <Option
      letter="D"
      title="Underline"
      className="tabs-d"
      desc="A flat strip on the panel fill with one hairline. The active tab carries a 2 px primary underline. Same 240 / 84 px widths."
    >
      <Panel />
    </Option>
  </Section>
);
