// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Avatar } from "@synnaxlabs/lyra/avatar";
import { Breadcrumb } from "@synnaxlabs/lyra/breadcrumb";
import { Button } from "@synnaxlabs/lyra/button";
import { Caret } from "@synnaxlabs/lyra/caret";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Status } from "@synnaxlabs/lyra/status";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Text } from "@synnaxlabs/lyra/text";
import { type CSSProperties, type ReactElement, type ReactNode } from "react";

import { current, LINES, plot, run } from "@/components/console/tour/mock/sim";

// The tank after it has run long enough to fill the plot and start a pressurization.
const INITIAL = run(34);
const PLOT_SIZE = { width: 380, height: 220 };

const TREE_OFFSET = "--pluto-tree-item-offset";

const CreateCalculatedIcon = Icon.createComposite(Icon.Calculation, {
  topRight: Icon.Add,
});

interface ToolbarItem {
  key: string;
  name: string;
  icon: ReactElement;
  trigger: string;
}

const TOOLBARS: ToolbarItem[] = [
  { key: "channel", name: "Channels", icon: <Icon.Channel />, trigger: "C" },
  { key: "range", name: "Ranges", icon: <Icon.Range />, trigger: "R" },
  { key: "workspace", name: "Projects", icon: <Icon.Project />, trigger: "W" },
  { key: "device", name: "Devices", icon: <Icon.Device />, trigger: "D" },
  { key: "task", name: "Tasks", icon: <Icon.Task />, trigger: "T" },
  { key: "arc", name: "Arcs", icon: <Icon.Arc />, trigger: "A" },
  { key: "status", name: "Statuses", icon: <Icon.Status />, trigger: "S" },
];

interface ActionProps {
  tooltip: string;
  filled?: boolean;
  children: ReactElement;
}

const Action = ({ tooltip, filled = false, children }: ActionProps): ReactElement => (
  <Button.Button
    size="small"
    rounded="small"
    variant={filled ? "filled" : "outlined"}
    className="console-toolbar__action"
    aria-label={tooltip}
    tabIndex={-1}
  >
    {children}
  </Button.Button>
);

interface ToolbarHeaderProps {
  children: ReactNode;
  actions?: ReactNode;
}

const ToolbarHeader = ({ children, actions }: ToolbarHeaderProps): ReactElement => (
  <Header.Header
    className="console-toolbar__header"
    level="h5"
    shrink={false}
    background={1}
  >
    {children}
    {actions != null && <Header.Actions>{actions}</Header.Actions>}
  </Header.Header>
);

const ToolbarTitle = ({ children }: { children: ReactNode }): ReactElement => (
  <Header.Title color={10} weight={500}>
    {children}
  </Header.Title>
);

interface DrawerProps {
  toolbar: ToolbarItem;
  actions?: ReactNode;
  children?: ReactNode;
}

const Drawer = ({ toolbar, actions, children }: DrawerProps): ReactElement => (
  <Flex.Box
    y
    empty
    className="console-nav__drawer docs-console__drawer"
    background={0}
    rounded="large"
    bordered
    borderColor={6}
    data-drawer={toolbar.key}
    hidden
  >
    <ToolbarHeader actions={actions}>
      <ToolbarTitle>
        {toolbar.icon}
        {toolbar.name}
      </ToolbarTitle>
    </ToolbarHeader>
    <Flex.Box y empty grow className="docs-console__toolbar-body">
      {children}
    </Flex.Box>
  </Flex.Box>
);

interface EmptyProps {
  message: string;
  action?: string;
}

const Empty = ({ message, action }: EmptyProps): ReactElement => (
  <Flex.Box center>
    <Text.Text el="span" y center status="disabled" gap="tiny">
      {message}
      {action != null && (
        <Text.Text el="span" variant="link">
          {action}
        </Text.Text>
      )}
    </Text.Text>
  </Flex.Box>
);

interface TreeItemProps {
  icon: Icon.ReactElement;
  name: string;
  expandable?: boolean;
  virtual?: boolean;
}

const TreeItem = ({
  icon,
  name,
  expandable = false,
  virtual = false,
}: TreeItemProps): ReactElement => (
  <List.Item
    itemKey={name}
    className="pluto-tree__item"
    style={{ [TREE_OFFSET]: "1.5rem" } as CSSProperties}
    gap="small"
    align="center"
    role="treeitem"
    aria-expanded={expandable ? false : undefined}
    tabIndex={-1}
  >
    {expandable && (
      <Caret.Animated
        className="pluto-tree__expansion-indicator"
        enabled={false}
        enabledLoc="bottom"
        disabledLoc="right"
      />
    )}
    {icon}
    <Text.Text el="span" overflow="fade" grow={virtual}>
      {name}
    </Text.Text>
    {virtual && <Icon.Virtual color={9} />}
  </List.Item>
);

const Tree = ({ children }: { children: ReactNode }): ReactElement => (
  <div className="docs-console__tree" role="tree">
    {children}
  </div>
);

const CHANNELS: Array<{ name: string; icon: Icon.ReactElement; virtual?: boolean }> = [
  { name: "sample_abort_cmd", icon: <Icon.Binary color={10} />, virtual: true },
  { name: "sample_press_vlv_cmd", icon: <Icon.Binary color={10} />, virtual: true },
  { name: "sample_press_vlv_state", icon: <Icon.Binary color={10} /> },
  { name: "sample_start_cmd", icon: <Icon.Binary color={10} />, virtual: true },
  { name: "sample_tank_pt", icon: <Icon.Decimal color={10} /> },
  { name: "sample_time", icon: <Icon.Time color={10} /> },
  { name: "sample_vent_vlv_cmd", icon: <Icon.Binary color={10} />, virtual: true },
  { name: "sample_vent_vlv_state", icon: <Icon.Binary color={10} /> },
];

const toolbar = (key: string): ToolbarItem => {
  const item = TOOLBARS.find((t) => t.key === key);
  if (item == null) throw new Error(`unknown toolbar: ${key}`);
  return item;
};

const ArcItem = ({ name }: { name: string }): ReactElement => (
  <List.Item
    itemKey={name}
    role="option"
    justify="between"
    align="center"
    tabIndex={-1}
  >
    <Flex.Box y gap="small" grow className="console-arc__metadata">
      <Flex.Box x align="center" gap="small">
        <Status.Indicator variant="success" className="console-arc-list-item__status" />
        <Text.Text el="span" overflow="ellipsis" weight={500}>
          {name}
        </Text.Text>
      </Flex.Box>
      <Text.Text el="span" level="small" status="success">
        Running
      </Text.Text>
    </Flex.Box>
    <Button.Button
      variant="outlined"
      size="small"
      status="error"
      aria-label={`Stop ${name}`}
      tabIndex={-1}
    >
      <Icon.Stop />
    </Button.Button>
  </List.Item>
);

const Drawers = (): ReactElement => (
  <>
    <Drawer
      toolbar={toolbar("channel")}
      actions={
        <>
          <Action tooltip="Create calculated channel" filled>
            <CreateCalculatedIcon />
          </Action>
          <Action tooltip="Create channel" filled>
            <Icon.Add />
          </Action>
        </>
      }
    >
      <Tree>
        <TreeItem icon={<Icon.Group color={10} />} name="Metrics" expandable />
        {CHANNELS.map((c) => (
          <TreeItem key={c.name} icon={c.icon} name={c.name} virtual={c.virtual} />
        ))}
      </Tree>
    </Drawer>
    <Drawer
      toolbar={toolbar("range")}
      actions={
        <>
          <Action tooltip="Open range explorer">
            <Icon.Explore />
          </Action>
          <Action tooltip="Create range" filled>
            <Icon.Add />
          </Action>
        </>
      }
    >
      <Empty message="No favorited ranges" action="Open range explorer" />
    </Drawer>
    <Drawer
      toolbar={toolbar("workspace")}
      actions={
        <Action tooltip="Create project" filled>
          <Icon.Add />
        </Action>
      }
    >
      <Tree>
        <TreeItem icon={<Icon.Project color={10} />} name="Sample project" expandable />
      </Tree>
    </Drawer>
    <Drawer toolbar={toolbar("device")} />
    <Drawer
      toolbar={toolbar("task")}
      actions={
        <Action tooltip="Create task" filled>
          <Icon.Add />
        </Action>
      }
    >
      <Empty message="No tasks" action="Create task" />
    </Drawer>
    <Drawer
      toolbar={toolbar("arc")}
      actions={
        <>
          <Action tooltip="Open Arc explorer">
            <Icon.Explore />
          </Action>
          <Action tooltip="Create Arc automation" filled>
            <Icon.Add />
          </Action>
        </>
      }
    >
      <div className="docs-console__list">
        <ArcItem name="Press sequence" />
        <ArcItem name="Tank simulator" />
      </div>
    </Drawer>
    <Drawer
      toolbar={toolbar("status")}
      actions={
        <>
          <Action tooltip="Open status explorer">
            <Icon.Explore />
          </Action>
          <Action tooltip="Create status" filled>
            <Icon.Add />
          </Action>
        </>
      }
    >
      <Empty message="No favorited statuses" action="Open status explorer" />
    </Drawer>
  </>
);

const Rail = (): ReactElement => (
  <Nav.Bar
    location="left"
    size="7.5rem"
    className="console-nav__bar docs-console__rail"
  >
    <Nav.Bar.Start bordered align="center">
      <Button.Button
        variant="text"
        size="large"
        className="console-project-selector__trigger"
        data-part="project"
        aria-label="Project selector"
        tabIndex={-1}
      >
        <Avatar.Avatar name="Sample project" />
      </Button.Button>
    </Nav.Bar.Start>
    <Nav.Bar.Content align="center">
      <Button.Button
        variant="outlined"
        align="center"
        justify="center"
        textColor={9}
        full="x"
        className="console-palette__btn"
        data-part="search"
        aria-label="Search and command palette"
        tabIndex={-1}
      >
        <Icon.Search />
      </Button.Button>
      <Menu.Menu>
        {TOOLBARS.map((t) => (
          <Menu.Item
            key={t.key}
            itemKey={t.key}
            size="medium"
            className="console-main-nav__item"
            data-part={t.key}
            data-toolbar={t.key}
            aria-label={t.name}
            tabIndex={-1}
          >
            {t.icon}
          </Menu.Item>
        ))}
      </Menu.Menu>
    </Nav.Bar.Content>
    <Nav.Bar.End bordered>
      <Menu.Menu>
        <Menu.Item
          itemKey="component"
          size="medium"
          className="console-main-nav__item"
          data-part="component"
          data-toolbar="component"
          aria-label="Component toolbar"
          tabIndex={-1}
        >
          <span
            className="docs-console__component-icon"
            data-component-icon="schematic"
            hidden
          >
            <Icon.Schematic />
          </span>
          <span className="docs-console__component-icon" data-component-icon="lineplot">
            <Icon.LinePlot />
          </span>
          <span
            className="docs-console__component-icon"
            data-component-icon="editor"
            hidden
          >
            <Icon.Arc />
          </span>
        </Menu.Item>
      </Menu.Menu>
    </Nav.Bar.End>
  </Nav.Bar>
);

const Top = (): ReactElement => (
  <Nav.Bar location="top" size="7rem" className="console-nav__bar docs-console__top">
    <Nav.Bar.Start gap="large">
      <span className="docs-console__traffic" aria-hidden>
        <span />
        <span />
        <span />
      </span>
      <Tabs.Frame
        className="console-panel-selector"
        value="sample"
        x
        align="center"
        empty={false}
        gap="small"
        data-part="panels"
      >
        <Tabs.Selector size="medium" variant="pill">
          <Tabs.Tab itemKey="sample" tabIndex={-1}>
            <Icon.Panel />
            <Text.Text el="span">Sample project</Text.Text>
          </Tabs.Tab>
        </Tabs.Selector>
        <Button.Button
          variant="text"
          textColor={9}
          aria-label="Create panel"
          tabIndex={-1}
        >
          <Icon.Add />
        </Button.Button>
      </Tabs.Frame>
    </Nav.Bar.Start>
    <Nav.Bar.Content grow />
    <Nav.Bar.End justify="end" align="center" gap="small">
      <Button.Button
        size="small"
        variant="text"
        data-part="help"
        aria-label="Open documentation"
        tabIndex={-1}
      >
        <Icon.QuestionMark />
      </Button.Button>
    </Nav.Bar.End>
  </Nav.Bar>
);

interface LeafTab {
  key: string;
  name: string;
  icon: ReactElement;
}

interface LeafProps {
  part: string;
  tabs: LeafTab[];
  focused?: boolean;
  children: ReactNode;
}

const Leaf = ({ part, tabs, focused = false, children }: LeafProps): ReactElement => (
  <div
    className="pluto-mosaic__leaf docs-console__leaf"
    data-part={part}
    data-leaf={part}
  >
    <Tabs.Frame value={tabs[0].key}>
      <Tabs.Selector align="start">
        {tabs.map((t, i) => (
          <Tabs.Tab
            key={t.key}
            itemKey={t.key}
            className={focused && i === 0 ? "pluto--alt-color" : undefined}
            data-leaf-tab={t.key}
            tabIndex={-1}
          >
            {t.icon}
            <Text.Text el="span">{t.name}</Text.Text>
          </Tabs.Tab>
        ))}
        <Flex.Box grow />
        <Flex.Box align="center" justify="center" className="pluto-panel-mosaic__cap">
          <Button.Button
            variant="text"
            size="small"
            className="pluto-panel-mosaic__create"
            aria-label="Add a tab"
            tabIndex={-1}
          >
            <Icon.Add color={9} />
          </Button.Button>
        </Flex.Box>
      </Tabs.Selector>
      <div className="docs-console__leaf-body">{children}</div>
    </Tabs.Frame>
  </div>
);

// Symbol geometry at the Console's zoom 1: each symbol sits 1px inside its node
// position, and SVG symbols draw at 0.8 of their view box.
const SVG_SCALE = 0.8;
const LABEL_GAP = 18.5;

type Point = [number, number];

const CORNER_RADIUS = 6;
const END_EXTENSION = 4;

const pipePath = (points: Point[]): string => {
  const pts = points.map(([x, y]): Point => [x, y]);
  const extend = (end: Point, from: Point): void => {
    const dx = Math.sign(end[0] - from[0]);
    const dy = Math.sign(end[1] - from[1]);
    end[0] += dx * END_EXTENSION;
    end[1] += dy * END_EXTENSION;
  };
  extend(pts[0], pts[1]);
  extend(pts[pts.length - 1], pts[pts.length - 2]);
  const length = (p: Point, q: Point): number => Math.hypot(q[0] - p[0], q[1] - p[1]);
  const toward = (p: Point, q: Point, d: number): Point => {
    const l = length(p, q);
    return [p[0] + ((q[0] - p[0]) / l) * d, p[1] + ((q[1] - p[1]) / l) * d];
  };
  let d = `M${pts[0][0]} ${pts[0][1]}`;
  for (let i = 1; i < pts.length - 1; i++) {
    const [prev, corner, next] = [pts[i - 1], pts[i], pts[i + 1]];
    const r = Math.min(
      CORNER_RADIUS,
      length(prev, corner) / 2,
      length(corner, next) / 2,
    );
    const [ax, ay] = toward(corner, prev, r);
    const [bx, by] = toward(corner, next, r);
    d += `L${ax} ${ay}Q${corner[0]} ${corner[1]} ${bx} ${by}`;
  }
  const last = pts[pts.length - 1];
  return `${d}L${last[0]} ${last[1]}`;
};

// The press valve feeds the bottom of the tank, and the vent valve drains its top.
const PIPES: Point[][] = [
  [
    [227.55, 160],
    [299.5, 160],
  ],
  [
    [427.5, 40],
    [500.1, 40],
  ],
];

const VALVE = { width: 87 * SVG_SCALE, height: 42 * SVG_SCALE };

const Label = ({ x, top, children }: { x: number; top: number; children: string }) => (
  <text
    className="docs-console__symbol-label"
    x={x}
    y={top - LABEL_GAP}
    textAnchor="middle"
    dominantBaseline="central"
  >
    {children}
  </text>
);

interface ValveProps {
  x: number;
  y: number;
  id: string;
  label: string;
  enabled: boolean;
}

const Valve = ({ x, y, id, label, enabled }: ValveProps): ReactElement => (
  <g className={enabled ? "docs-console--enabled" : undefined} data-valve-group={id}>
    <Label x={x + 1 + VALVE.width / 2} top={y + 1}>
      {label}
    </Label>
    <path
      className="docs-console__symbol"
      transform={`translate(${x + 1} ${y + 1}) scale(${SVG_SCALE})`}
      d="M43.5 21L6.35453 2.20349C4.35901 1.19372 2 2.64384 2 4.88029V37.1197C2 39.3562 4.35901 40.8063 6.35453 39.7965L43.5 21ZM43.5 21L80.6455 2.20349C82.641 1.19372 85 2.64384 85 4.8803V37.1197C85 39.3562 82.641 40.8063 80.6455 39.7965L43.5 21Z"
    />
  </g>
);

interface SymbolButtonProps {
  x: number;
  y: number;
  width: number;
  label: string;
  command: string;
  danger?: boolean;
}

const BUTTON_CLASS = "docs-console__symbol-button";

const SymbolButton = ({ x, y, width, label, command, danger }: SymbolButtonProps) => (
  <g
    transform={`translate(${x + 1} ${y + 1})`}
    className={danger ? `${BUTTON_CLASS} ${BUTTON_CLASS}--danger` : BUTTON_CLASS}
    data-command={command}
  >
    <rect x={0.5} y={0.5} width={width - 1} height={27} rx={6} />
    <text x={width / 2} y={14} textAnchor="middle" dominantBaseline="central">
      {label}
    </text>
  </g>
);

// The tank's wall is a 2px CSS border with 50% / 10% corner radii on a 125x200 box.
const TANK = { x: 302, y: 2, width: 123, height: 198, rx: 61.5, ry: 19 };
const VALUE = { x: 310, y: 241, width: 107, height: 28, content: 70 };

const Schematic = (): ReactElement => {
  const { pressure, press, vent } = current(INITIAL);
  const fill = (pressure / 500) * TANK.height;
  return (
    <svg
      className="docs-console__schematic pluto-symbol-colored"
      viewBox="-7.5 -70 742 400"
      aria-hidden
    >
      <defs>
        <clipPath id="docs-console-tank-clip">
          <rect {...TANK} />
        </clipPath>
      </defs>
      <g className="docs-console__pipes">
        {PIPES.map((points, i) => (
          <path key={i} d={pipePath(points)} />
        ))}
      </g>
      <Valve x={156} y={142.2} id="press" label="Press valve" enabled={press === 1} />
      <Label x={TANK.x + TANK.width / 2} top={1}>
        Tank
      </Label>
      <rect
        className="docs-console__tank-fill"
        clipPath="url(#docs-console-tank-clip)"
        x={TANK.x}
        y={TANK.y + TANK.height - fill}
        width={TANK.width}
        height={fill}
        data-tank-fill
      />
      <rect className="docs-console__symbol" {...TANK} />
      <Valve x={500} y={22.2} id="vent" label="Vent valve" enabled={vent === 1} />
      <Label x={VALUE.x + VALUE.width / 2} top={VALUE.y}>
        Tank pressure
      </Label>
      <g transform={`translate(${VALUE.x} ${VALUE.y})`}>
        <rect
          className="docs-console__value-box"
          x={1}
          y={1}
          width={VALUE.width - 2}
          height={VALUE.height - 2}
          rx={3}
        />
        <text
          className="docs-console__value"
          x={17}
          y={VALUE.height / 2}
          dominantBaseline="central"
          data-tank-value
        >
          {pressure.toFixed(1)}
        </text>
        <line
          className="docs-console__value-divider"
          x1={VALUE.content + 3}
          x2={VALUE.content + 3}
          y1={0}
          y2={VALUE.height}
        />
        <text
          className="docs-console__value-units"
          x={(VALUE.content + 4 + VALUE.width - 2) / 2}
          y={VALUE.height / 2}
          textAnchor="middle"
          dominantBaseline="central"
        >
          psi
        </text>
      </g>
      <SymbolButton x={426.5} y={223.5} width={60} label="Start" command="start" />
      <SymbolButton
        x={426.5}
        y={256.5}
        width={60}
        label="Abort"
        command="abort"
        danger
      />
    </svg>
  );
};

const LinePlot = (): ReactElement => (
  <div className="docs-console__plot" data-plot>
    <svg
      viewBox={`0 0 ${PLOT_SIZE.width} ${PLOT_SIZE.height}`}
      width={PLOT_SIZE.width}
      height={PLOT_SIZE.height}
      aria-hidden
      dangerouslySetInnerHTML={{
        __html: plot(INITIAL, PLOT_SIZE.width, PLOT_SIZE.height),
      }}
    />
    <div className="docs-console__legend">
      {LINES.map((l) => (
        <Flex.Box key={l.key} x align="center" gap="small">
          <span
            className="docs-console__legend-swatch"
            style={{ background: l.color }}
          />
          <Text.Text el="span" level="small">
            {l.name}
          </Text.Text>
        </Flex.Box>
      ))}
    </div>
  </div>
);

export interface WindowProps {
  /** The Press sequence code, passed by the page as a named slot. */
  sequence?: ReactNode;
  /** The Tank simulator code, passed by the page as a named slot. */
  simulator?: ReactNode;
}

const Editor = ({ sequence, simulator }: WindowProps): ReactElement => (
  <div className="docs-console__editor">
    <div className="docs-console__code" data-leaf-panel="sequence">
      {sequence}
    </div>
    <div className="docs-console__code" data-leaf-panel="simulator" hidden>
      {simulator}
    </div>
    <Flex.Box
      x
      justify="between"
      align="stretch"
      empty
      bordered
      pack
      rounded="large"
      borderColor={7}
      background={1}
      className="console-task-controls console-arc-editor__controls"
    >
      <Button.Button
        el="div"
        variant="text"
        className="console-task-status"
        tabIndex={-1}
      >
        <Flex.Box
          x
          align="start"
          gap="small"
          justify="between"
          grow
          className="console-task-status__message"
        >
          <Flex.Box x gap="small" className="console-task-status__message-content">
            <Text.Text el="span" level="p">
              <Status.Indicator variant="success" />
            </Text.Text>
            <Text.Text
              el="span"
              level="p"
              status="success"
              weight={500}
              overflow="wrap"
              className="console-task-status__message-text"
            >
              Task started successfully
            </Text.Text>
          </Flex.Box>
        </Flex.Box>
        <Icon.Expand className="console-task-status__expand-indicator" />
      </Button.Button>
      <Flex.Box
        x
        align="center"
        justify="end"
        className="console-task-controls__actions"
      >
        <SelectTrigger icon={<Icon.Rack />} className="console-rack-select">
          Node 1 Embedded Driver
        </SelectTrigger>
        <Button.Button
          status="error"
          size="medium"
          variant="filled"
          aria-label="Stop"
          tabIndex={-1}
        >
          <Icon.Stop />
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  </div>
);

const Mosaic = (props: WindowProps): ReactElement => (
  <Flex.Box
    x
    className="pluto-mosaic docs-console__mosaic"
    rounded="large"
    bordered
    borderColor={6}
    background={0}
    data-part="mosaic"
  >
    <div className="docs-console__mosaic-column">
      <Leaf
        part="schematic"
        tabs={[{ key: "tank", name: "Tank", icon: <Icon.Schematic /> }]}
      >
        <Schematic />
      </Leaf>
      <Leaf
        part="lineplot"
        focused
        tabs={[{ key: "pressure", name: "Tank pressure", icon: <Icon.LinePlot /> }]}
      >
        <LinePlot />
      </Leaf>
    </div>
    <Leaf
      part="editor"
      tabs={[
        { key: "sequence", name: "Press sequence", icon: <Icon.Arc /> },
        { key: "simulator", name: "Tank simulator", icon: <Icon.Arc /> },
      ]}
    >
      <Editor {...props} />
    </Leaf>
  </Flex.Box>
);

interface SelectTriggerProps {
  icon?: Icon.ReactElement;
  className?: string;
  children: ReactNode;
}

const SelectTrigger = ({
  icon,
  className,
  children,
}: SelectTriggerProps): ReactElement => (
  <Button.Button
    variant="outlined"
    gap="small"
    full="x"
    className={`pluto-dialog__trigger docs-console__select-trigger ${className ?? ""}`}
    tabIndex={-1}
  >
    {icon}
    <Text.Text el="span" overflow="ellipsis">
      {children}
    </Text.Text>
    <Caret.Animated enabled={false} enabledLoc="bottom" disabledLoc="left" />
  </Button.Button>
);

interface FieldProps {
  label: string;
  values: Array<{ name: string; icon: Icon.ReactElement }>;
  grow?: boolean;
}

const MultipleField = ({ label, values, grow = false }: FieldProps): ReactElement => (
  <Input.Item x label={label} grow={grow} align="center">
    <Tag.Tags
      full="x"
      variant="outlined"
      className="pluto-dialog__trigger pluto--variant-outlined"
      grow
    >
      {values.map((v) => (
        <Tag.Tag key={v.name} size="small" icon={v.icon} onClose={() => {}}>
          {v.name}
        </Tag.Tag>
      ))}
      <Caret.Animated
        className="pluto-text--p"
        enabled={false}
        enabledLoc="bottom"
        disabledLoc="left"
        color={9}
      />
    </Tag.Tags>
  </Input.Item>
);

const BottomHeader = ({ children }: { children: ReactNode }): ReactElement => (
  <Header.Header
    className="console-toolbar__header docs-console__bottom-header"
    level="h5"
    shrink={false}
    background={1}
  >
    {children}
  </Header.Header>
);

const ToolbarButtons = ({ csv = false }: { csv?: boolean }): ReactElement => (
  <Flex.Box x gap="small" className="docs-console__toolbar-buttons">
    {csv && (
      <Button.Button
        size="medium"
        variant="text"
        tabIndex={-1}
        aria-label="Download as CSV"
      >
        <Icon.CSV />
      </Button.Button>
    )}
    <Button.Button size="medium" variant="text" tabIndex={-1} aria-label="Export">
      <Icon.Export />
    </Button.Button>
    <Button.Button size="medium" variant="text" tabIndex={-1} aria-label="Copy link">
      <Icon.Link />
    </Button.Button>
  </Flex.Box>
);

const ToolbarTabs = ({ tabs, value }: { tabs: string[]; value: string }) => (
  <Tabs.Frame value={value} x align="center">
    <Tabs.Selector>
      {tabs.map((name) => (
        <Tabs.Tab key={name} itemKey={name} tabIndex={-1}>
          <Text.Text el="span">{name}</Text.Text>
        </Tabs.Tab>
      ))}
    </Tabs.Selector>
  </Tabs.Frame>
);

const ComponentDrawer = (): ReactElement => (
  <Flex.Box
    y
    empty
    className="console-nav__drawer docs-console__bottom"
    background={0}
    rounded="large"
    bordered
    borderColor={6}
    data-drawer="component"
    hidden
  >
    <div data-component-panel="lineplot">
      <BottomHeader>
        <ToolbarTitle>
          <Icon.LinePlot />
          Tank pressure
        </ToolbarTitle>
        <Flex.Box x align="center" empty>
          <ToolbarButtons csv />
          <ToolbarTabs
            tabs={["Data", "Lines", "Axes", "Properties", "Rules"]}
            value="Data"
          />
        </Flex.Box>
      </BottomHeader>
      <Flex.Box y full="x" className="docs-console__fields">
        <MultipleField
          label="X1"
          values={[{ name: "Rolling 30s", icon: <Icon.Range /> }]}
          grow
        />
        <MultipleField
          label="Y1"
          values={[{ name: "sample_tank_pt", icon: <Icon.Decimal color={10} /> }]}
          grow
        />
        <MultipleField
          label="Y2"
          values={[
            { name: "sample_press_vlv_state", icon: <Icon.Binary color={10} /> },
            { name: "sample_vent_vlv_state", icon: <Icon.Binary color={10} /> },
          ]}
          grow
        />
        <Input.Item x label="X1" align="center" className="docs-console__field-x">
          <SelectTrigger icon={<Icon.Time />}>sample_time</SelectTrigger>
        </Input.Item>
      </Flex.Box>
    </div>
    <div data-component-panel="schematic" hidden>
      <BottomHeader>
        <Breadcrumb.Breadcrumb level="h5">
          <Breadcrumb.Segment weight={500} color={10} level="h5">
            <Icon.Schematic />
            Tank
          </Breadcrumb.Segment>
        </Breadcrumb.Breadcrumb>
        <Flex.Box x align="center" empty>
          <ToolbarButtons />
          <ToolbarTabs tabs={["Symbols", "Properties", "Control"]} value="Properties" />
        </Flex.Box>
      </BottomHeader>
      <Flex.Box center className="docs-console__bottom-empty">
        <Text.Text el="span" status="disabled" center>
          Select a schematic element to configure its properties
        </Text.Text>
      </Flex.Box>
    </div>
    <div data-component-panel="editor" hidden>
      <BottomHeader>
        <ToolbarTitle>
          <Icon.Arc />
          Press sequence
        </ToolbarTitle>
      </BottomHeader>
      <Flex.Box center className="docs-console__bottom-empty">
        <Empty message="No properties to configure" />
      </Flex.Box>
    </div>
  </Flex.Box>
);

const COMMANDS: Array<{ name: string; icon: ReactElement }> = [
  { name: "Create channel", icon: <Icon.Channel /> },
  { name: "Create line plot", icon: <Icon.LinePlot /> },
  { name: "Create schematic", icon: <Icon.Schematic /> },
  { name: "Create Arc automation", icon: <Icon.Arc /> },
  { name: "Create range", icon: <Icon.Range /> },
  { name: "Create project", icon: <Icon.Project /> },
];

const Palette = (): ReactElement => (
  <div className="docs-console__palette" data-palette hidden>
    <div className="docs-console__palette-input">
      <Icon.Search />
      <Text.Text el="span" color={10}>
        &gt;
      </Text.Text>
      <span className="docs-console__palette-caret" />
    </div>
    <div className="docs-console__palette-list">
      {COMMANDS.map((c, i) => (
        <div
          key={c.name}
          className={`docs-console__palette-item${i === 0 ? " docs-console__palette-item--selected" : ""}`}
        >
          {c.icon}
          <Text.Text el="span">{c.name}</Text.Text>
        </div>
      ))}
    </div>
  </div>
);

/** A copy of the Synnax Console with the sample project open. */
export const Window = (props: WindowProps): ReactElement => (
  <div className="docs-console__window">
    <div className="console-main docs-console__workspace">
      <Top />
      <Rail />
      <div className="docs-console__content">
        <div className="docs-console__main">
          <Drawers />
          <Mosaic {...props} />
        </div>
        <ComponentDrawer />
      </div>
    </div>
    <Palette />
  </div>
);
