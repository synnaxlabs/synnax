// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/lyra/context";
import { Form } from "@synnaxlabs/lyra/form";
import { Store } from "@synnaxlabs/lyra/store";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  type RefCallback,
  useState,
  useSyncExternalStore,
} from "react";

import {
  REGISTER_TYPES,
  RegisterGlyph,
  REGISTERS,
} from "@/feature/modbus/task/Register";
import { addressSpan, overlaps, type ReadChannel } from "@/feature/modbus/task/types";
import { CSS } from "@/platform/css";

const CHANNELS_PATH = "config.channels";

// The channel under the pointer, in the list or in the map. Both mark it, so each
// points at the other.
const Hovered = Store.createPresence("Modbus.Hovered");
const [SetHovered, useSetHoveredContext] = context.create<(key?: string) => void>({
  displayName: "Modbus.SetHovered",
  providerName: "Modbus.HoveredProvider",
});

/** Shares the channel under the pointer between the channel list and the map. */
export const HoveredProvider = ({ children }: PropsWithChildren): ReactElement => {
  const [hovered, setHovered] = useState<string>();
  return (
    <SetHovered value={setHovered}>
      <Hovered.Context value={hovered}>{children}</Hovered.Context>
    </SetHovered>
  );
};

/** @returns whether the channel is under the pointer, in the list or in the map. */
export const useIsHovered = Hovered.useIsPresent;

const useSetHovered = (): ((key?: string) => void) =>
  useSetHoveredContext("Modbus.useSetHovered");

/**
 * @returns a ref that marks the channel as hovered while the pointer is over the
 * element. It listens to native events, since React also sends the element the pointer
 * events of its portals, such as an open menu.
 */
export const useHoveredRef = (key: string): RefCallback<HTMLElement> => {
  const setHovered = useSetHovered();
  return (el) => {
    if (el == null) return;
    const handleEnter = () => setHovered(key);
    const handleLeave = () => setHovered();
    el.addEventListener("pointerenter", handleEnter);
    el.addEventListener("pointerleave", handleLeave);
    return () => {
      el.removeEventListener("pointerenter", handleEnter);
      el.removeEventListener("pointerleave", handleLeave);
    };
  };
};

const usePalette = (): (string | undefined)[] =>
  (Theming.use().colors.visualization.palettes.default ?? []).map((c) =>
    color.cssString(c),
  );

// Subscribes to a value derived from the channels, so a component re-renders only
// when that value changes, not on every edit to a channel.
const useDerived = <T,>(derive: (channels: ReadChannel[]) => T): T => {
  const { get, bind } = Form.useContext();
  const getSnapshot = () => derive(get<ReadChannel[]>(CHANNELS_PATH).value);
  return useSyncExternalStore(bind, getSnapshot, getSnapshot);
};

/** @returns the plot palette's color at the channel's position in the list. */
export const useColor = (key: string): string | undefined => {
  const palette = usePalette();
  const index = useDerived((channels) => channels.findIndex((c) => c.key === key));
  return palette.at(index % palette.length);
};

/**
 * @returns whether another channel of the same type reads one of the channel's
 * addresses.
 */
export const useOverlaps = (key: string): boolean =>
  useDerived((channels) => {
    // A channel that is being removed is gone before its item unmounts.
    const channel = channels.find((c) => c.key === key);
    return channel != null && overlaps(channels, channel);
  });

// Past this many addresses, the ticks would merge into one fill.
const MAX_TICKS = 128;

const rangeText = ([lo, hi]: [number, number]): string =>
  lo === hi ? `${lo}` : `${lo}–${hi}`;

/**
 * The addresses that the channels read: a lane for each register type, on one address
 * scale. Each channel fills its addresses in its color, and a hatched band marks an
 * address that two channels read. Pointing at a segment marks its channel in the list,
 * and the other way around.
 */
export const RegisterMap = (): ReactElement | null => {
  const channels = Form.useFieldValue<ReadChannel[]>(CHANNELS_PATH);
  const hovered = Hovered.usePresent();
  const setHovered = useSetHovered();
  const palette = usePalette();
  if (channels.length === 0) return null;
  const spans = channels.map((c, i) => ({
    channel: c,
    span: addressSpan(c),
    color: palette.at(i % palette.length),
  }));
  const lo = Math.min(...spans.map(({ span }) => span[0]));
  const hi = Math.max(...spans.map(({ span }) => span[1]));
  const count = hi - lo + 1;
  const left = (address: number) => `${((address - lo) / count) * 100}%`;
  const width = ([a, b]: [number, number]) => `${((b - a + 1) / count) * 100}%`;
  const hoveredSpan = spans.find(({ channel }) => channel.key === hovered);
  return (
    <div
      className={CSS.cls(
        CSS.BE("modbus", "map"),
        hoveredSpan != null && CSS.BEM("modbus", "map", "hovered"),
      )}
    >
      {REGISTER_TYPES.filter((t) => channels.some((c) => c.type === t)).map((t) => {
        const lane = spans.filter(({ channel }) => channel.type === t);
        const shared: [number, number][] = [];
        lane.forEach((a, i) =>
          lane.slice(i + 1).forEach((b) => {
            const sharedLo = Math.max(a.span[0], b.span[0]);
            const sharedHi = Math.min(a.span[1], b.span[1]);
            if (sharedLo <= sharedHi) shared.push([sharedLo, sharedHi]);
          }),
        );
        return (
          <div
            key={t}
            className={CSS.cls(
              CSS.BE("modbus", "lane"),
              hoveredSpan?.channel.type === t && CSS.BEM("modbus", "lane", "hovered"),
            )}
          >
            <span className={CSS.BE("modbus", "lane-name")}>
              <RegisterGlyph type={t} />
              {REGISTERS[t].name}
            </span>
            <span className={CSS.BE("modbus", "track")}>
              {count <= MAX_TICKS &&
                Array.from({ length: count }, (_, i) => (
                  <span
                    key={i}
                    className={CSS.BE("modbus", "tick")}
                    style={{ left: left(lo + i) }}
                  />
                ))}
              {lane.map(({ channel, span, color }) => (
                <span
                  key={channel.key}
                  className={CSS.cls(
                    CSS.BE("modbus", "segment"),
                    channel.key === hovered && CSS.BEM("modbus", "segment", "hovered"),
                  )}
                  style={{ left: left(span[0]), width: width(span), background: color }}
                  onPointerEnter={() => setHovered(channel.key)}
                  onPointerLeave={() => setHovered()}
                  onClick={() =>
                    document
                      .getElementById(channel.key)
                      ?.scrollIntoView({ block: "nearest", behavior: "smooth" })
                  }
                />
              ))}
              {shared.map((span) => (
                <span
                  key={span.join("-")}
                  className={CSS.BE("modbus", "overlap")}
                  style={{ left: left(span[0]), width: width(span) }}
                />
              ))}
            </span>
          </div>
        );
      })}
      <div className={CSS.BE("modbus", "axis")}>
        <span className={CSS.BE("modbus", "scale")}>
          <span className={CSS.BE("modbus", "axis-end")}>{lo}</span>
          {hoveredSpan != null && (
            <span
              className={CSS.BE("modbus", "axis-hovered")}
              style={{
                left: left(hoveredSpan.span[0]),
                width: width(hoveredSpan.span),
              }}
            >
              {rangeText(hoveredSpan.span)}
            </span>
          )}
          <span className={CSS.BE("modbus", "axis-end")}>{hi}</span>
        </span>
      </div>
    </div>
  );
};
