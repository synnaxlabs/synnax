import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree } from "@synnaxlabs/lyra/tree";
import { type ReactNode } from "react";

import { CHANNELS, noop, Option, Row, Section, StaticTooltip } from "@/options/common";
import { Toast } from "@/options/samples";
import { LONG_TAB, Panel } from "@/options/sections/TabStyle";

export type Mark = "today" | "fade" | "ellipsis" | "middle";

/** Middle truncation: a head that ellipsizes and a tail that never shrinks. */
export const MiddleText = ({
  children,
  tail = 10,
}: {
  children: string;
  tail?: number;
}) => (
  <span className="opt-mid">
    <span className="opt-mid-head">{children.slice(0, -tail)}</span>
    <span className="opt-mid-tail">{children.slice(-tail)}</span>
  </span>
);

const TREE_CHANNELS = CHANNELS.slice(0, 5);
const NODES = TREE_CHANNELS.map((key) => ({ key }));
const getItem = List.createGetItem(
  (key: string) => ({ key, name: key }),
  (keys: string[]) => keys.map((key) => ({ key, name: key })),
);

const ITEM_HEIGHT = 27;
const HOVERED = 1;

const ChannelTree = ({ mark }: { mark: Mark }) => {
  const props = Tree.use({ nodes: NODES, selected: [TREE_CHANNELS[0]] });
  return (
    <div className="opt-tree" style={{ width: 196, height: 5 * ITEM_HEIGHT }}>
      <Tree.Tree {...props} getItem={getItem} virtual={false}>
        {({ key, ...rest }) => (
          <Tree.Item key={key} {...rest} hovered={rest.index === HOVERED}>
            <Icon.Channel />
            {mark === "middle" ? (
              <MiddleText>{key}</MiddleText>
            ) : (
              <Text.Text overflow="fade" className="opt-tree-name">
                {key}
              </Text.Text>
            )}
          </Tree.Item>
        )}
      </Tree.Tree>
    </div>
  );
};

const STRIP_TABS = [
  { key: "schematic", name: "Feed system", icon: <Icon.Schematic /> },
  { key: "long", name: LONG_TAB, icon: <Icon.LinePlot /> },
  { key: "status", name: "Status", icon: <Icon.Status /> },
];

const ChannelSelect = ({ mark }: { mark: Mark }) => (
  <div style={{ width: 196 }}>
    <Select.Simple<string>
      resourceName="channel"
      value={CHANNELS[0]}
      onChange={noop}
      icon={<Icon.Channel />}
      full="x"
    >
      {CHANNELS.map((c) => (
        <Select.Item key={c} itemKey={c} textValue={c}>
          {mark === "middle" ? <MiddleText>{c}</MiddleText> : c}
        </Select.Item>
      ))}
    </Select.Simple>
  </div>
);

const TOAST_NAME = "hotfire_14_ox_tank_pressure_transducer_01";

/** Status names are strings; the middle option renders its name through MiddleText. */
const toastName = (mark: Mark): string =>
  mark === "middle"
    ? (<MiddleText>{TOAST_NAME}</MiddleText> as unknown as string)
    : TOAST_NAME;

const Tip = ({ children }: { children: ReactNode }) => (
  <StaticTooltip
    className="opt-hover-tip"
    style={{ top: (HOVERED + 1) * ITEM_HEIGHT + 4, left: 4 }}
  >
    {children}
  </StaticTooltip>
);

const OverflowSample = ({ mark }: { mark: Mark }) => (
  <Flex.Box x gap="huge" align="start">
    <Row label="Tree (pointer on the second row)">
      <div className="opt-hover-anchor" style={{ height: 5 * ITEM_HEIGHT + 12 }}>
        <ChannelTree mark={mark} />
        {mark !== "today" && <Tip>{TREE_CHANNELS[HOVERED]}</Tip>}
      </div>
    </Row>
    <Flex.Box y gap="large">
      <Row label="Tabs">
        <Panel tabs={STRIP_TABS} selected="schematic" width={440} content={false} />
      </Row>
      <Flex.Box x gap="huge" align="start">
        <Row label="Select value">
          <ChannelSelect mark={mark} />
        </Row>
        <Row label="Toast name">
          <Toast variant="success" name={toastName(mark)} message="Copied link to clipboard" />
        </Row>
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);

export const OverflowSection = () => (
  <Section
    id="overflow"
    n={2}
    title="Fade versus ellipsis for cut text"
    summary="Sets the mark that shows a name is cut. Every alternative also shows the full name in a tooltip, only when the name is cut; one hovered tree row shows it."
  >
    <Option
      letter="A"
      title="Today"
      desc="Tree, tabs, and select fade; the toast name ends in an ellipsis. No tooltip shows the full name."
    >
      <OverflowSample mark="today" />
    </Option>
    <Option
      letter="B"
      title="Fade everywhere"
      className="f-b"
      desc="Every cut name fades at its end, the toast name too. Tooltip when cut."
    >
      <OverflowSample mark="fade" />
    </Option>
    <Option
      letter="C"
      title="End ellipsis everywhere"
      className="f-c"
      desc="Every cut name ends in an ellipsis. Tooltip when cut."
    >
      <OverflowSample mark="ellipsis" />
    </Option>
    <Option
      letter="D"
      title="Middle ellipsis for channel names"
      className="f-d"
      desc="Channel names keep their last 10 characters; other names (the tab) end in an ellipsis. Tooltip when cut."
    >
      <OverflowSample mark="middle" />
    </Option>
  </Section>
);
