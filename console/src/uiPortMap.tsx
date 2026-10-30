// SCRATCH: never commit. Prototype of the NI analog read port map.

import "@/uiPortMap.css";

import { type device } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Device, Synnax } from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import {
  createContext,
  type CSSProperties,
  type MouseEvent,
  type ReactElement,
  type ReactNode,
  use,
} from "react";

import { PortField } from "@/feature/ni/device/PortField";
import { useByKeys } from "@/feature/ni/device/queries";
import { createNextAIChannel } from "@/feature/ni/task/createChannel";
import { type AIChannel, channelPort } from "@/feature/ni/task/types";
import { ContextMenu as PlatformContextMenu } from "@/platform/context-menu";
import { Task } from "@/platform/task";

// Stand-in for what the Driver would read from DAQmx at scan time. `half` marks a
// device whose port n pairs with port n + half, in blocks of 2 * half.
interface Layout {
  count: number;
  half?: number;
}

const LAYOUTS: Record<string, Layout> = {
  "USB-6343": { count: 32, half: 8 },
  "USB-6008": { count: 8, half: 4 },
  "NI 9205": { count: 32, half: 8 },
  "NI 9213": { count: 16 },
  "NI 9237": { count: 4 },
  "NI 9375": { count: 0 },
};

const layoutOf = (model?: string): Layout | undefined =>
  model == null ? undefined : LAYOUTS[model];

const isPositive = (l: Layout, p: number): boolean =>
  l.half != null && p % (2 * l.half) < l.half;

const terminalConfig = (ch: AIChannel): string =>
  (ch as { terminalConfig?: string }).terminalConfig ?? "Cfg_Default";

const usesNegative = (l: Layout, ch: AIChannel): boolean => {
  const port = channelPort(ch);
  if (port == null || !isPositive(l, port)) return false;
  const tc = terminalConfig(ch);
  return tc === "Diff" || tc === "Cfg_Default";
};

interface Claim {
  key: string;
  role: "own" | "negative";
}

const claimsFor = (
  l: Layout,
  channels: AIChannel[],
  deviceKey: string,
): Map<number, Claim[]> => {
  const claims = new Map<number, Claim[]>();
  const add = (p: number, c: Claim) => claims.set(p, [...(claims.get(p) ?? []), c]);
  for (const ch of channels) {
    if (ch.device !== deviceKey) continue;
    const port = channelPort(ch);
    if (port == null) continue;
    add(port, { key: ch.key, role: "own" });
    if (l.half != null && usesNegative(l, ch))
      add(port + l.half, { key: ch.key, role: "negative" });
  }
  return claims;
};

const firstFree = (l: Layout, claims: Map<number, Claim[]>): number | undefined => {
  for (let p = 0; p < l.count; p++) if (!claims.has(p)) return p;
  return undefined;
};

export interface PortMapContextValue {
  hovered?: string;
  setHovered: (key?: string) => void;
  selected: string[];
  onSelect: (keys: string[]) => void;
}

const Context = createContext<PortMapContextValue | null>(null);
Context.displayName = "Context";
export const PortMapContext = Context;
export const usePortMap = (): PortMapContextValue | null => use(Context);

// Each channel keeps one hue across the list, the port map, and the port field.
const useHueOf = (): ((key: string) => string | undefined) => {
  const channels = Form.useFieldValue<AIChannel[]>("config.channels");
  const palette = Theming.use().colors.visualization.palettes.default ?? [];
  return (key) => {
    const i = channels.findIndex((c) => c.key === key);
    if (i < 0 || palette.length === 0) return undefined;
    return color.cssString(palette[i % palette.length]);
  };
};

const hueStyle = (hue?: string): CSSProperties | undefined =>
  hue == null ? undefined : ({ "--ni-hue": hue } as CSSProperties);

const TC_NAMES: Record<string, string> = {
  Cfg_Default: "Default, which is differential here",
  Diff: "differential",
  RSE: "referenced single-ended",
  NRSE: "non-referenced single-ended",
  PseudoDiff: "pseudodifferential",
};

const scrollToChannel = (key: string) =>
  document
    .getElementById(`ni-port-item-${key}`)
    ?.scrollIntoView({ block: "nearest", behavior: "smooth" });

interface CellProps {
  deviceKey: string;
  port: number;
  layout: Layout;
  claims: Map<number, Claim[]>;
  channels: AIChannel[];
  ctx: PortMapContextValue;
  hueOf: (key: string) => string | undefined;
  onAdd: (port: number) => void;
  onPick?: (port: number) => void;
  pill: boolean;
  row: number;
  column: number;
}

const Cell = ({
  deviceKey,
  port,
  layout,
  claims,
  channels,
  ctx,
  hueOf,
  onAdd,
  onPick,
  pill,
  row,
  column,
}: CellProps): ReactElement => {
  const list = claims.get(port) ?? [];
  const own = list.find((c) => c.role === "own");
  const negative = pill ? (claims.get(port + (layout.half ?? 0)) ?? []) : [];
  const conflict = list.length > 1 || negative.length > 1;
  const owner = own ?? list[0];
  const keys = [...list, ...negative].map((c) => c.key);
  const hovered = ctx.hovered != null && keys.includes(ctx.hovered);
  const selected = keys.some((k) => ctx.selected.includes(k));
  const ch = owner != null ? channels.find((c) => c.key === owner.key) : undefined;
  let title: string;
  if (owner == null)
    title =
      layout.half != null && !isPositive(layout, port)
        ? `Port ${port} is free. It is the negative input of port ${port - layout.half} when that port is differential. Click to add a channel.`
        : `Port ${port} is free. Click to add a channel.`;
  else if (pill)
    title = `Port ${port}, ${TC_NAMES[terminalConfig(ch!)]}. Uses port ${port + layout.half!} as its negative input.`;
  else if (owner.role === "negative")
    title = `Port ${port} is the negative input of port ${port - layout.half!}.`;
  else title = `Port ${port}, ${TC_NAMES[terminalConfig(ch!)] ?? "single-ended"}.`;
  if (conflict)
    title += ` Conflict: another channel also reads port ${pill ? port + layout.half! : port}.`;
  if (onPick != null)
    title = title.replace("Click to add a channel.", "Click to use it.");
  const handleClick = () => {
    if (onPick != null) return onPick(port);
    if (owner == null) return onAdd(port);
    ctx.onSelect([owner.key]);
    scrollToChannel(owner.key);
  };
  return (
    <button
      type="button"
      title={title}
      data-menu-key={`${deviceKey}|${port}`}
      className={[
        Menu.CONTEXT_TARGET,
        "ni-port-map__cell",
        owner == null ? "ni-port-map__cell--free" : "ni-port-map__cell--used",
        pill && "ni-port-map__cell--pill",
        conflict && "ni-port-map__cell--conflict",
        hovered && "ni-port-map__cell--hovered",
        selected && "ni-port-map__cell--selected",
        ch?.disabled === true && "ni-port-map__cell--disabled",
      ]
        .filter(Boolean)
        .join(" ")}
      style={{
        ...hueStyle(owner != null ? hueOf(owner.key) : undefined),
        gridColumn: column,
        gridRow: pill ? `${row} / span 2` : row,
      }}
      onClick={handleClick}
      onPointerEnter={() => owner != null && ctx.setHovered(owner.key)}
      onPointerLeave={() => ctx.setHovered()}
    >
      <span>{port}</span>
      {pill && <span className="ni-port-map__negative">{port + layout.half!}</span>}
    </button>
  );
};

interface DeviceBlockProps {
  dev: device.Device;
  layout: Layout;
  channels: AIChannel[];
  ctx: PortMapContextValue;
  hueOf: (key: string) => string | undefined;
  onAdd: (deviceKey: string, port: number) => void;
  onPick?: (port: number) => void;
  hideHeader?: boolean;
}

const DeviceBlock = ({
  dev,
  layout,
  channels,
  ctx,
  hueOf,
  onAdd,
  onPick,
  hideHeader = false,
}: DeviceBlockProps): ReactElement => {
  const claims = claimsFor(layout, channels, dev.key);
  const used = [...claims.keys()].length;
  const groups: ReactNode[] = [];
  const cellProps = {
    deviceKey: dev.key,
    layout,
    claims,
    channels,
    ctx,
    hueOf,
    onPick,
  };
  const handleAdd = (p: number) => onAdd(dev.key, p);
  if (layout.half != null) {
    const size = 2 * layout.half;
    for (let start = 0; start < layout.count; start += size) {
      const cells: ReactNode[] = [];
      for (let i = 0; i < layout.half; i++) {
        const pos = start + i;
        const neg = pos + layout.half;
        const pill = claims
          .get(neg)
          ?.some(
            (c) =>
              c.role === "negative" && claims.get(pos)?.some((o) => o.key === c.key),
          );
        cells.push(
          <Cell
            key={pos}
            port={pos}
            {...cellProps}
            onAdd={handleAdd}
            pill={pill === true}
            row={1}
            column={i + 1}
          />,
        );
        if (pill !== true)
          cells.push(
            <Cell
              key={neg}
              port={neg}
              {...cellProps}
              onAdd={handleAdd}
              pill={false}
              row={2}
              column={i + 1}
            />,
          );
      }
      groups.push(
        <div
          key={start}
          className="ni-port-map__group ni-port-map__group--paired"
          style={{ gridTemplateColumns: `repeat(${layout.half}, var(--ni-cell))` }}
        >
          {cells}
        </div>,
      );
    }
  } else {
    const width = Math.min(8, layout.count);
    for (let start = 0; start < layout.count; start += 16) {
      const cells: ReactNode[] = [];
      for (let p = start; p < Math.min(start + 16, layout.count); p++)
        cells.push(
          <Cell
            key={p}
            port={p}
            {...cellProps}
            onAdd={handleAdd}
            pill={false}
            row={Math.floor((p - start) / width) + 1}
            column={((p - start) % width) + 1}
          />,
        );
      groups.push(
        <div
          key={start}
          className="ni-port-map__group"
          style={{ gridTemplateColumns: `repeat(${width}, var(--ni-cell))` }}
        >
          {cells}
        </div>,
      );
    }
  }
  return (
    <div className="ni-port-map__device">
      {!hideHeader && (
        <div
          className={`ni-port-map__device-header ${Menu.CONTEXT_TARGET}`}
          data-menu-key={`${dev.key}|device`}
        >
          <span className="ni-port-map__model">{dev.model}</span>
          <span className="ni-port-map__meta">
            {dev.location} · {used} of {layout.count} ports used
          </span>
        </div>
      )}
      <div className="ni-port-map__groups">{groups}</div>
    </div>
  );
};

interface AddDeviceProps {
  rack?: number;
  exclude: string[];
  onAdd: (dev: device.Device) => void;
  pinned?: boolean;
}

const AddDevice = ({
  rack,
  exclude,
  onAdd,
  pinned = false,
}: AddDeviceProps): ReactElement => {
  const filter = (d: device.Device) =>
    d.make === "NI" &&
    (rack == null || d.rack === rack) &&
    (layoutOf(d.model)?.count ?? 0) > 0 &&
    !exclude.includes(d.key);
  const client = Synnax.use();
  const handleChange = (key: string) => {
    if (client == null || !key) return;
    void client.devices.retrieve({ key }).then(onAdd);
  };
  return (
    <div className={pinned ? "ni-port-map__add--pinned" : "ni-port-map__add"}>
      <Device.SelectSingle
        value={undefined}
        onChange={handleChange}
        initialQuery={{
          makes: ["NI"],
          models: Object.keys(LAYOUTS).filter((m) => LAYOUTS[m].count > 0),
          ...(rack != null ? { racks: [rack] } : {}),
        }}
        filter={filter}
        emptyContent="No other NI devices with analog inputs on this rack"
        triggerProps={{
          placeholder: "Add device",
          variant: "text",
          icon: <Icon.Add />,
          hideCaret: true,
          // Reads as the list's "Add channel" item, in the same place.
          className: pinned ? "console-create-list-item console--small" : undefined,
        }}
      />
    </div>
  );
};

export interface PortMapProps {
  wide?: boolean;
  column?: boolean;
}

export const PortMap = ({
  wide = false,
  column = false,
}: PortMapProps): ReactElement | null => {
  const ctx = use(Context);
  const channels = Form.useFieldValue<AIChannel[]>("config.channels");
  const { set } = Form.useContext();
  const sortedKeys = Task.useChannelDeviceKeys();
  const devices = useByKeys(sortedKeys) as device.Device[] | undefined;
  const deviceKeys = [
    ...new Set(channels.map((c) => c.device).filter((k): k is string => !!k)),
  ];
  const hueOf = useHueOf();
  const menuProps = Menu.useContextMenu();
  if (ctx == null) return null;
  const add = (deviceKey: string, port: number) => {
    const ch = createNextAIChannel(channels, ctx.selected[0]);
    (ch as { device: string }).device = deviceKey;
    (ch as { port: number }).port = port;
    set("config.channels", [...channels, ch]);
    ctx.onSelect([ch.key]);
    requestAnimationFrame(() => scrollToChannel(ch.key));
  };
  const addDevice = (dev: device.Device) => {
    const layout = layoutOf(dev.model);
    if (layout == null) return;
    const port = firstFree(layout, claimsFor(layout, channels, dev.key));
    if (port == null) return;
    // A new task starts with one channel and no device: give it the device.
    const empty = channels.findIndex((c) => !c.device);
    if (empty < 0) return add(dev.key, port);
    const key = channels[empty].key;
    set(`config.channels.${key}.device`, dev.key);
    set(`config.channels.${key}.port`, port);
    ctx.onSelect([channels[empty].key]);
  };
  const rack = devices?.[0]?.rack;
  const parseKey = (key: string) => {
    const [deviceKey, what] = key.split("|");
    const dev = devices?.find((d) => d.key === deviceKey);
    const layout = layoutOf(dev?.model);
    if (dev == null || layout == null) return null;
    const claims = claimsFor(layout, channels, deviceKey);
    if (what === "device") return { dev, layout, claims };
    const port = Number(what);
    const list = claims.get(port) ?? [];
    const owner = list.find((c) => c.role === "own") ?? list[0];
    const ch = owner != null ? channels.find((c) => c.key === owner.key) : undefined;
    return { dev, layout, claims, port, ch };
  };
  const remove = (keys: string[]) => {
    const next = channels.filter((c) => !keys.includes(c.key));
    set("config.channels", next);
    ctx.onSelect(next.length > 0 ? [next[0].key] : []);
  };
  const handleContextMenu = (e: MouseEvent) => {
    const el = (e.target as HTMLElement).closest<HTMLElement>("[data-menu-key]");
    const ch = el != null ? parseKey(el.dataset.menuKey ?? "")?.ch : undefined;
    // The menu acts on the channel the details pane shows.
    if (ch != null) ctx.onSelect([ch.key]);
    menuProps.open(e);
  };
  const renderMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    const parsed = keys.length > 0 ? parseKey(keys[0]) : null;
    let items: ReactNode = null;
    if (parsed != null && parsed.port == null) {
      const free = firstFree(parsed.layout, parsed.claims);
      const onDevice = channels.filter((c) => c.device === parsed.dev.key);
      items = (
        <>
          {free != null && (
            <Menu.Item itemKey="add" onClick={() => add(parsed.dev.key, free)}>
              <Icon.Add />
              Add channel
            </Menu.Item>
          )}
          <Menu.Divider />
          <Menu.Item
            itemKey="removeDevice"
            onClick={() => remove(onDevice.map((c) => c.key))}
          >
            <Icon.Close />
            Remove device
          </Menu.Item>
          <Menu.Divider />
        </>
      );
    } else if (parsed?.port != null && parsed.ch == null) {
      const { port } = parsed;
      items = (
        <>
          <Menu.Item itemKey="add" onClick={() => add(parsed.dev.key, port)}>
            <Icon.Add />
            Add channel
          </Menu.Item>
          <Menu.Divider />
        </>
      );
    } else if (parsed?.ch != null) {
      const { ch, layout } = parsed;
      const path = `config.channels.${ch.key}`;
      const port = channelPort(ch);
      const wiring =
        "terminalConfig" in ch && port != null && isPositive(layout, port)
          ? usesNegative(layout, ch)
            ? { label: "Use RSE", value: "RSE", icon: <Icon.Disconnect /> }
            : { label: "Use differential", value: "Diff", icon: <Icon.Connect /> }
          : null;
      items = (
        <>
          <PlatformContextMenu.RenameItem
            onClick={() => Text.edit(`ni-details-name-${ch.key}`)}
          />
          {wiring != null && (
            <Menu.Item
              itemKey="wiring"
              onClick={() => set(`${path}.terminalConfig`, wiring.value)}
            >
              {wiring.icon}
              {wiring.label}
            </Menu.Item>
          )}
          {ch.disabled ? (
            <Menu.Item itemKey="enable" onClick={() => set(`${path}.disabled`, false)}>
              <Icon.Enable />
              Enable
            </Menu.Item>
          ) : (
            <Menu.Item itemKey="disable" onClick={() => set(`${path}.disabled`, true)}>
              <Icon.Disable />
              Disable
            </Menu.Item>
          )}
          <Menu.Divider />
          <PlatformContextMenu.RemoveItem onClick={() => remove([ch.key])} />
          <Menu.Divider />
        </>
      );
    }
    return (
      <PlatformContextMenu.Menu>
        {items}
        <PlatformContextMenu.ReloadConsoleItem />
      </PlatformContextMenu.Menu>
    );
  };
  const body = (
    <div
      className={[
        "ni-port-map",
        wide && "ni-port-map--wide",
        column && "ni-port-map--column",
      ]
        .filter(Boolean)
        .join(" ")}
      onContextMenu={handleContextMenu}
    >
      <div className="ni-port-map__devices">
        {deviceKeys.map((k) => {
          const dev = devices?.find((d) => d.key === k);
          const layout = layoutOf(dev?.model);
          if (dev == null || layout == null) return null;
          return (
            <DeviceBlock
              key={k}
              dev={dev}
              layout={layout}
              channels={channels}
              ctx={ctx}
              hueOf={hueOf}
              onAdd={add}
            />
          );
        })}
      </div>
      <AddDevice rack={rack} exclude={deviceKeys} onAdd={addDevice} pinned={column} />
    </div>
  );
  return (
    <Menu.ContextMenu {...menuProps} menu={renderMenu}>
      {body}
    </Menu.ContextMenu>
  );
};

/** The port mark in a list item: the same cell as the map. */
export const PortMark = ({ itemKey }: { itemKey: string }): ReactElement | null => {
  const ctx = use(Context);
  const ch = Form.useFieldValue<AIChannel>(`config.channels.${itemKey}`);
  const hueOf = useHueOf();
  const port = channelPort(ch);
  if (ctx == null) return null;
  return (
    <span
      id={`ni-port-item-${itemKey}`}
      className="ni-port-mark"
      style={hueStyle(hueOf(itemKey))}
    >
      {port ?? "NA"}
    </span>
  );
};

/** The selected channel's port in the details header, in the state colors. */
export const PortBadge = ({ path }: { path: string }): ReactElement => {
  const ch = Form.useFieldValue<AIChannel>(path);
  const channels = Form.useFieldValue<AIChannel[]>("config.channels");
  const devices = useByKeys(ch.device ? [ch.device] : []) as
    device.Device[] | undefined;
  const layout = layoutOf(devices?.find((d) => d.key === ch.device)?.model);
  const port = channelPort(ch);
  let conflict = false;
  let title: string | undefined;
  if (layout != null && port != null) {
    const claims = claimsFor(layout, channels, ch.device);
    const shared = (p: number) => (claims.get(p)?.length ?? 0) > 1;
    conflict =
      shared(port) || (usesNegative(layout, ch) && shared(port + layout.half!));
    if (conflict) {
      const p = shared(port) ? port : port + layout.half!;
      title = `Another channel also reads port ${p}. Pick a free port.`;
    }
  }
  return (
    <span
      title={title}
      className={[
        "ni-port-badge",
        conflict && "ni-port-badge--conflict",
        port == null && "ni-port-badge--empty",
        ch.disabled === true && "ni-port-badge--disabled",
      ]
        .filter(Boolean)
        .join(" ")}
    >
      {port ?? "–"}
    </span>
  );
};

/** A dropdown of the device's real ports, with what uses each one. */
export const PortSelectField = ({ prefix }: { prefix: string }): ReactElement => {
  const deviceKey = Form.useFieldValue<string>(`${prefix}.device`);
  const selfKey = Form.useFieldValue<string>(`${prefix}.key`);
  const channels = Form.useFieldValue<AIChannel[]>("config.channels");
  const devices = useByKeys(deviceKey ? [deviceKey] : []) as
    device.Device[] | undefined;
  const dev = devices?.find((d) => d.key === deviceKey);
  const layout = layoutOf(dev?.model);
  if (layout == null || layout.count === 0) return <PortField path={prefix} />;
  const others = channels.filter((c) => c.key !== selfKey);
  const claims = claimsFor(layout, others, deviceKey);
  const items: ReactNode[] = [];
  for (let p = 0; p < layout.count; p++) {
    const list = claims.get(p) ?? [];
    let note = "";
    if (list.some((c) => c.role === "negative"))
      note = `Negative input of port ${p - layout.half!}`;
    else if (list.length > 0) note = "Used";
    items.push(
      <Select.Item<number> key={p} itemKey={p} justify="between">
        <span>Port {p}</span>
        <span className="ni-port-select__note">{note}</span>
      </Select.Item>,
    );
  }
  return (
    <Form.Field<number> path={`${prefix}.port`} label="Port" grow>
      {({ value, onChange }) => (
        <Select.Simple<number>
          value={value}
          onChange={onChange}
          resourceName="port"
          allowNone={false}
        >
          {items}
        </Select.Simple>
      )}
    </Form.Field>
  );
};

/** The device's ports as the port field: a click sets this channel's port. */
export const PortPickerField = ({ prefix }: { prefix: string }): ReactElement => {
  const ctx = use(Context);
  const deviceKey = Form.useFieldValue<string>(`${prefix}.device`);
  const channels = Form.useFieldValue<AIChannel[]>("config.channels");
  const devices = useByKeys(deviceKey ? [deviceKey] : []) as
    device.Device[] | undefined;
  const hueOf = useHueOf();
  const dev = devices?.find((d) => d.key === deviceKey);
  const layout = layoutOf(dev?.model);
  if (ctx == null || dev == null || layout == null || layout.count === 0)
    return <PortField path={prefix} />;
  return (
    <Form.Field<number> path={`${prefix}.port`} label="Port" grow>
      {({ onChange }) => (
        <div className="ni-port-map ni-port-map--picker">
          <DeviceBlock
            dev={dev}
            layout={layout}
            channels={channels}
            ctx={ctx}
            hueOf={hueOf}
            onAdd={() => {}}
            onPick={onChange}
            hideHeader
          />
        </div>
      )}
    </Form.Field>
  );
};
