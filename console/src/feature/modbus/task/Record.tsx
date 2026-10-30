// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Scratch only: the locked record row. Never commit.

import { channel as clientChannel } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { context } from "@synnaxlabs/lyra/context";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Store } from "@synnaxlabs/lyra/store";
import { Telem } from "@synnaxlabs/pluto";
import { DataType, deep, Density } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type MouseEvent,
  type PropsWithChildren,
  type ReactElement,
  useState,
} from "react";

import { useFromConfig } from "@/feature/modbus/device/queries";
import {
  isVariableDensityReadChannel,
  READ_CHANNEL_SCHEMAS,
  type ReadChannel,
  type ReadChannelType,
} from "@/feature/modbus/task/types";
import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

// The four Modbus tables are a width (one bit or a 16-bit word) and an access (read
// only or read and write). A bit is one cell and a word two. Solid is read and write,
// hollow is read only.
const SHAPES: Record<"bit" | "word", [string, string]> = {
  bit: ["M7.5 6h9v12h-9z", "M9.5 8v8h5V8z"],
  word: ["M2 6h9v12H2zM13 6h9v12h-9z", "M4 8v8h5V8zM15 8v8h5V8z"],
};

const RegisterGlyph = ({ type }: { type: ReadChannelType }): ReactElement => {
  const [outer, inner] =
    SHAPES[type === "coil" || type === "discrete_input" ? "bit" : "word"];
  const hollow = type === "discrete_input" || type === "input_register";
  return (
    <svg
      viewBox="0 0 24 24"
      className={CSS.BE("modbus", "glyph")}
      fill="currentColor"
      aria-hidden
    >
      <path fillRule="evenodd" d={hollow ? `${outer}${inner}` : outer} />
    </svg>
  );
};

const MENU_PROPS = { className: CSS.BE("modbus", "menu") };

const REGISTERS: Record<ReadChannelType, { name: string; description: string }> = {
  coil: { name: "Coil", description: "Bit, read and write" },
  discrete_input: { name: "Discrete input", description: "Bit, read only" },
  input_register: { name: "Input register", description: "Word, read only" },
  holding_register: { name: "Holding register", description: "Word, read and write" },
};

const RegisterField = ({ path }: { path: string }): ReactElement | null => {
  const field = PForm.useField<ReadChannelType>(`${path}.type`, {
    onChange: (value, { get, set }) => {
      const prev = get<ReadChannel>(path).value;
      if (prev.type === value) return;
      const schema = READ_CHANNEL_SCHEMAS[value];
      const next = schema.parse({ type: value });
      set(path, { ...deep.overrideValidItems(next, prev, schema), type: value });
    },
  });
  if (field == null) return null;
  return (
    <Select.Simple<ReadChannelType>
      value={field.value}
      onChange={field.onChange}
      preview={field.preview}
      allowNone={false}
      resourceName="register type"
      dialogProps={MENU_PROPS}
    >
      {(Object.keys(REGISTERS) as ReadChannelType[]).map((t) => (
        <Select.Item<ReadChannelType> key={t} itemKey={t}>
          <RegisterGlyph type={t} />
          <span className={CSS.BE("modbus", "option-name")}>{REGISTERS[t].name}</span>
          <span className={CSS.BE("modbus", "option-description")}>
            {REGISTERS[t].description}
          </span>
        </Select.Item>
      ))}
    </Select.Simple>
  );
};

const registerCount = (dataType: string): number =>
  Math.max(1, new DataType(dataType).density.valueOf() / 2);

const span = (c: ReadChannel): [number, number] => [
  c.address,
  c.address + (isVariableDensityReadChannel(c) ? registerCount(c.dataType) : 1) - 1,
];

const overlaps = (channels: ReadChannel[], c: ReadChannel): boolean => {
  const [lo, hi] = span(c);
  return channels.some((o) => {
    if (o.key === c.key || o.type !== c.type) return false;
    const [oLo, oHi] = span(o);
    return oLo <= hi && lo <= oHi;
  });
};

const AddressField = ({ path }: { path: string }) => {
  const field = PForm.useField<number>(`${path}.address`);
  const ch = PForm.useFieldValue<ReadChannel>(path);
  const channels = PForm.useFieldValue<ReadChannel[]>("config.channels");
  if (field == null) return null;
  const [lo, hi] = span(ch);
  const overlapping = overlaps(channels, ch);
  return (
    <div
      className={CSS.cls(
        CSS.BE("modbus", "address"),
        overlapping && CSS.BEM("modbus", "address", "overlap"),
      )}
      style={{ "--digits": String(lo).length } as CSSProperties}
    >
      <Input.Numeric
        value={field.value}
        onChange={field.onChange}
        preview={field.preview}
        variant="shadow"
        showDragHandle={false}
        aria-label="Address"
        bounds={{ lower: 0, upper: 65536 }}
      />
      {hi > lo && <span className={CSS.BE("modbus", "address-end")}>{`–${hi}`}</span>}
    </div>
  );
};

const DataTypeField = ({ path }: { path: string }) => {
  const field = PForm.useField<string>(`${path}.dataType`);
  if (field == null) return null;
  return (
    <Telem.SelectDataType
      value={field.value}
      onChange={field.onChange}
      preview={field.preview}
      hideVariableDensity
      location="bottom"
    />
  );
};

type Swaps = "device" | "none" | "bytes" | "words" | "both";

const SWAPS: Record<Exclude<Swaps, "device">, [boolean, boolean]> = {
  none: [false, false],
  bytes: [true, false],
  words: [false, true],
  both: [true, true],
};

const SWAP_NAMES: Record<Exclude<Swaps, "device">, string> = {
  none: "Big-endian",
  bytes: "Byte swap",
  words: "Word swap",
  both: "Little-endian",
};

const order = (bytes: number, [bytesSwapped, wordsSwapped]: [boolean, boolean]) => {
  const words: string[][] = [];
  for (let i = 0; i < Math.max(bytes, 2); i += 2)
    words.push([String.fromCharCode(65 + i), String.fromCharCode(66 + i)]);
  if (wordsSwapped) words.reverse();
  return words.flatMap(([a, b]) => (bytesSwapped ? [b, a] : [a, b]));
};

const swapsOf = (bytes: number, [b, w]: [boolean, boolean]): Exclude<Swaps, "device"> => {
  if (bytes <= 2) return b ? "both" : "none";
  return (Object.entries(SWAPS).find(([, [sb, sw]]) => sb === b && sw === w)?.[0] ??
    "none") as Exclude<Swaps, "device">;
};

const GLYPH_WIDTH = 30;

// The bytes in wire order as slices of one fixed width, toned by significance with the
// most significant darkest. Each order keeps its shape at any data type size.
const OrderGlyph = ({ bytes, inherited }: { bytes: string[]; inherited?: boolean }) => {
  const n = bytes.length;
  const w = (GLYPH_WIDTH - (n - 1)) / n;
  return (
    <svg
      viewBox={`0 0 ${GLYPH_WIDTH} 12`}
      className={CSS.cls(
        CSS.BE("modbus", "order-glyph"),
        inherited === true && CSS.BEM("modbus", "order-glyph", "inherited"),
      )}
      aria-hidden
    >
      {bytes.map((b, i) => (
        <rect
          key={b}
          x={i * (w + 1)}
          width={w}
          height="12"
          style={{ "--rank": (b.charCodeAt(0) - 65) / Math.max(n - 1, 1) } as CSSProperties}
        />
      ))}
    </svg>
  );
};

const OrderField = ({ path, dataType }: { path: string; dataType: string }) => {
  const connection = useFromConfig()?.properties.connection;
  const ctx = PForm.useContext();
  const bytesPath = `${path}.bytesSwapped`;
  const wordsPath = `${path}.wordsSwapped`;
  const bytesSwapped = PForm.useFieldValue<boolean | undefined>(bytesPath, {
    optional: true,
  });
  const wordsSwapped = PForm.useFieldValue<boolean | undefined>(wordsPath, {
    optional: true,
  });
  const size = new DataType(dataType).density.valueOf();
  const multiWord = size > Density.BIT16.valueOf();
  const device: [boolean, boolean] = [
    connection?.swapBytes ?? false,
    multiWord && (connection?.swapWords ?? false),
  ];
  const resolved: [boolean, boolean] = [
    bytesSwapped ?? device[0],
    multiWord && (wordsSwapped ?? device[1]),
  ];
  const value: Swaps =
    bytesSwapped == null && wordsSwapped == null ? "device" : swapsOf(size, resolved);
  const options: Exclude<Swaps, "device">[] = multiWord
    ? ["none", "bytes", "words", "both"]
    : ["none", "both"];
  const handleChange = (next: Swaps) => {
    if (next === "device") {
      ctx.set(bytesPath, undefined);
      ctx.set(wordsPath, undefined);
      return;
    }
    const [b, w] = SWAPS[next];
    ctx.set(bytesPath, b);
    ctx.set(wordsPath, multiWord ? w : undefined);
  };
  const deviceBytes = order(size, device);
  return (
    <Select.Simple<Swaps>
      value={value}
      preview={ctx.mode === "preview"}
      onChange={handleChange}
      allowNone={false}
      resourceName="byte order"
      dialogProps={MENU_PROPS}
    >
      <Select.Item<Swaps> itemKey="device">
        <OrderGlyph bytes={deviceBytes} inherited />
        <span className={CSS.BE("modbus", "option-name")}>From device</span>
        <span className={CSS.BE("modbus", "option-description")}>
          {SWAP_NAMES[swapsOf(size, device)]}
        </span>
      </Select.Item>
      {options.map((o) => {
        const bytes = order(size, SWAPS[o]);
        return (
          <Select.Item<Swaps> key={o} itemKey={o}>
            <OrderGlyph bytes={bytes} />
            <span className={CSS.BE("modbus", "option-name")}>{SWAP_NAMES[o]}</span>
            <span
              className={CSS.cls(
                CSS.BE("modbus", "option-description"),
                CSS.BE("modbus", "letters"),
              )}
            >
              {bytes.join("")}
            </span>
          </Select.Item>
        );
      })}
    </Select.Simple>
  );
};

const NAME_TYPES: Record<ReadChannelType, string> = {
  coil: "coil_input",
  discrete_input: "discrete_input",
  holding_register: "holding_register_input",
  input_register: "register_input",
};

const defaultName = (deviceName: string | undefined, c: ReadChannel): string => {
  if (deviceName == null) return "New channel";
  let s = `${clientChannel.escapeInvalidName(deviceName)}_${NAME_TYPES[c.type]}_${c.address}`;
  if (isVariableDensityReadChannel(c)) s += `_${new DataType(c.dataType).toString(true)}`;
  return s;
};

export const PALETTE = [
  "#20A4F3",
  "#DC136C",
  "#7AC74F",
  "#8075FF",
  "#FFC43D",
  "#FE5F55",
  "#f18f01",
  "#52ffb8",
];

const useColor = (key: string): string => {
  const channels = PForm.useFieldValue<ReadChannel[]>("config.channels");
  const i = channels.findIndex((c) => c.key === key);
  return PALETTE[Math.max(i, 0) % PALETTE.length];
};

// The channel under the pointer, in the list or in the map. Both views mark it, so
// each points at the other.
const Hot = Store.createPresence("Modbus.Hot");
const [SetHot, useSetHot] = context.create<(key?: string) => void>({
  defaultValue: () => {},
  displayName: "Modbus.SetHot",
});

export const HotProvider = ({ children }: PropsWithChildren): ReactElement => {
  const [hot, setHot] = useState<string>();
  return (
    <SetHot value={setHot}>
      <Hot.Context value={hot}>{children}</Hot.Context>
    </SetHot>
  );
};

const Row = ({ itemKey, ...rest }: Task.ChannelListItemProps) => {
  const path = `config.channels.${itemKey}`;
  const ch = PForm.useFieldValue<ReadChannel>(path);
  const color = useColor(ch.key);
  const dev = useFromConfig();
  const { onSelect, selected } = Select.useItemState(itemKey);
  const hot = Hot.useIsPresent(itemKey);
  const setHot = useSetHot();
  const typed = isVariableDensityReadChannel(ch);
  // Only the checkbox and modifier clicks select, so editing a field never selects.
  const handleSelect = (_: string, e: MouseEvent<HTMLElement>) => {
    if (Select.hasModifier(e)) onSelect();
  };
  return (
    <List.Item
      {...rest}
      itemKey={itemKey}
      selected={selected}
      hovered={hot}
      onHover={() => setHot(itemKey)}
      onPointerLeave={() => setHot()}
      onSelect={handleSelect}
      className={CSS.BE("modbus", "record")}
      style={{ "--console-modbus-color": color } as CSSProperties}
      align="center"
      x
    >
      <Input.Checkbox
        value={selected}
        onChange={onSelect}
        size="medium"
        variant="text"
        reveal={!selected}
        aria-label="Select"
      />
      <div className={CSS.BE("modbus", "fields")}>
        <div className={CSS.BE("modbus", "cell")}>
          <RegisterField path={path} />
        </div>
        <div className={CSS.BE("modbus", "cell")}>
          <AddressField path={path} />
        </div>
        <div className={CSS.BE("modbus", "cell")}>
          {typed && <DataTypeField path={path} />}
        </div>
        <div className={CSS.BE("modbus", "cell")}>
          {typed && <OrderField path={path} dataType={ch.dataType} />}
        </div>
      </div>
      <Icon.Arrow.Right className={CSS.BE("modbus", "flow")} />
      <Flex.Box x align="center" justify="between" className={CSS.BE("modbus", "name")}>
        <Task.ChannelName
          channel={ch.channel}
          namePath={`${path}.name`}
          defaultName={defaultName(dev?.name, ch)}
          id={Task.getChannelNameID(itemKey)}
        />
        <Task.EnabledCheckbox path={`${path}.disabled`} />
      </Flex.Box>
    </List.Item>
  );
};

export const ROW = Component.renderProp(Row);

const TYPES: ReadChannelType[] = [
  "coil",
  "discrete_input",
  "input_register",
  "holding_register",
];

const rangeText = ([a, b]: [number, number]): string => (a === b ? `${a}` : `${a}–${b}`);

// A lane per register table on one shared address scale. Each channel's registers fill
// its color, and two channels that read the same register mark the overlap. Pointing
// at a segment marks its row, and pointing at a row marks its segment.
export const RegisterMap = (): ReactElement | null => {
  const channels = PForm.useFieldValue<ReadChannel[]>("config.channels");
  const hot = Hot.usePresent();
  const setHot = useSetHot();
  if (channels.length === 0) return null;
  const spans = channels.map((c, i) => ({
    c,
    span: span(c),
    color: PALETTE[i % PALETTE.length],
  }));
  const lo = Math.min(...spans.map((s) => s.span[0]));
  const hi = Math.max(...spans.map((s) => s.span[1]));
  const n = hi - lo + 1;
  const x = (a: number) => `${((a - lo) / n) * 100}%`;
  const w = (a: number, b: number) => `${((b - a + 1) / n) * 100}%`;
  const hotSpan = spans.find((s) => s.c.key === hot);
  return (
    <div
      className={CSS.cls(
        CSS.BE("modbus", "map"),
        hotSpan != null && CSS.BEM("modbus", "map", "hot"),
      )}
    >
      {TYPES.filter((t) => channels.some((c) => c.type === t)).map((t) => {
        const lane = spans.filter((s) => s.c.type === t);
        const hits: [number, number][] = [];
        lane.forEach((a, i) =>
          lane.slice(i + 1).forEach((b) => {
            const l = Math.max(a.span[0], b.span[0]);
            const h = Math.min(a.span[1], b.span[1]);
            if (l <= h) hits.push([l, h]);
          }),
        );
        return (
          <div
            key={t}
            className={CSS.cls(
              CSS.BE("modbus", "lane"),
              hotSpan?.c.type === t && CSS.BEM("modbus", "lane", "hot"),
            )}
          >
            <span className={CSS.BE("modbus", "lane-name")}>
              <RegisterGlyph type={t} />
              {REGISTERS[t].name}
            </span>
            <span className={CSS.BE("modbus", "track")}>
              {Array.from({ length: n }, (_, i) => (
                <span
                  key={i}
                  className={CSS.BE("modbus", "tick")}
                  style={{ left: x(lo + i) }}
                />
              ))}
              {lane.map(({ c, span: [a, b], color }) => (
                <span
                  key={c.key}
                  className={CSS.cls(
                    CSS.BE("modbus", "segment"),
                    c.key === hot && CSS.BEM("modbus", "segment", "hot"),
                  )}
                  style={{ left: x(a), width: w(a, b), background: color }}
                  onPointerEnter={() => setHot(c.key)}
                  onPointerLeave={() => setHot()}
                  onClick={() =>
                    document
                      .getElementById(c.key)
                      ?.scrollIntoView({ block: "nearest", behavior: "smooth" })
                  }
                />
              ))}
              {hits.map(([a, b]) => (
                <span
                  key={`${a}-${b}`}
                  className={CSS.BE("modbus", "overlap")}
                  style={{ left: x(a), width: w(a, b) }}
                />
              ))}
            </span>
          </div>
        );
      })}
      <div className={CSS.BE("modbus", "axis")}>
        <span className={CSS.BE("modbus", "scale")}>
          <span className={CSS.BE("modbus", "axis-end")}>{lo}</span>
          {hotSpan != null && (
            <span
              className={CSS.BE("modbus", "axis-hot")}
              style={{ left: x(hotSpan.span[0]), width: w(...hotSpan.span) }}
            >
              {rangeText(hotSpan.span)}
            </span>
          )}
          <span className={CSS.BE("modbus", "axis-end")}>{hi}</span>
        </span>
      </div>
    </div>
  );
};

export const HEADER = (
  <>
    <Header.Header>
      <Header.Title weight={500} color={10}>
        Channels
      </Header.Title>
    </Header.Header>
    <RegisterMap />
    <div className={CSS.BE("modbus", "headings")}>
      <span className={CSS.BE("modbus", "headings-check")} />
      <div className={CSS.BE("modbus", "fields")}>
        <span>Type</span>
        <span>Address</span>
        <span>Data type</span>
        <span>Byte order</span>
      </div>
      <span className={CSS.BE("modbus", "headings-flow")} />
      <span>Channel</span>
    </div>
  </>
);
