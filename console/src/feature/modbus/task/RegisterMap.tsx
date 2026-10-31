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
import { Menu } from "@synnaxlabs/lyra/menu";
import { Store } from "@synnaxlabs/lyra/store";
import {
  type PointerEvent,
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
import {
  addressSpan,
  type Channel,
  findOverlaps,
  type Overlaps,
} from "@/feature/modbus/task/types";
import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

const CHANNELS_PATH = "config.channels";

// The channel under the pointer, in the list or in the map. Both mark it, so each
// points at the other.
const Hovered = Store.createPresence("Modbus.Hovered");
const [SetHovered, useSetHoveredContext] = context.create<(key?: string) => void>({
  displayName: "Modbus.SetHovered",
  providerName: "Modbus.RegisterMapProvider",
});

const [FindOverlaps, useFindOverlaps] = context.create<
  (channels: Channel[]) => Overlaps
>({ displayName: "Modbus.FindOverlaps", providerName: "Modbus.RegisterMapProvider" });

// Finds the overlaps once for each version of the channels, however many items ask.
const createFindOverlaps = () => {
  let last: { channels: Channel[]; overlaps: Overlaps } | undefined;
  return (channels: Channel[]): Overlaps => {
    if (last?.channels !== channels)
      last = { channels, overlaps: findOverlaps(channels) };
    return last.overlaps;
  };
};

/**
 * Shares the channel under the pointer and the overlapping addresses between the
 * channel list and the map.
 */
export const RegisterMapProvider = ({ children }: PropsWithChildren): ReactElement => {
  const [hovered, setHovered] = useState<string>();
  const [find] = useState(createFindOverlaps);
  return (
    <SetHovered value={setHovered}>
      <FindOverlaps value={find}>
        <Hovered.Context value={hovered}>{children}</Hovered.Context>
      </FindOverlaps>
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

// Subscribes to a value derived from the channels, so a component re-renders only
// when that value changes, not on every edit to a channel.
const useDerived = <T,>(derive: (channels: Channel[]) => T): T => {
  const { get, bind } = Form.useContext();
  const getSnapshot = () => derive(get<Channel[]>(CHANNELS_PATH).value);
  return useSyncExternalStore(bind, getSnapshot, getSnapshot);
};

/**
 * @returns whether another channel of the same type uses one of the channel's
 * addresses.
 */
export const useOverlaps = (key: string): boolean => {
  const find = useFindOverlaps("Modbus.useOverlaps");
  return useDerived((channels) => find(channels).keys.has(key));
};

// Past this many addresses, the ticks would merge into one fill.
const MAX_TICKS = 128;

const rangeText = ([lo, hi]: [number, number]): string =>
  lo === hi ? `${lo}` : `${lo}–${hi}`;

interface Free<C extends Channel> {
  type: C["type"];
  address: number;
}

export interface RegisterMapProps<C extends Channel> {
  /** Creates the channel that a click on a free address adds. */
  createChannel: (type: C["type"], address: number) => C;
}

/**
 * The addresses that the channels use: a lane for each register type, on one address
 * scale. Used addresses fill, and a hatched band marks an address that two channels
 * use. Pointing at a segment marks its channel in the list, and the other way around.
 * A segment opens its channel's menu, and a click on a free address adds a channel
 * there.
 */
export const RegisterMap = <C extends Channel>({
  createChannel,
}: RegisterMapProps<C>): ReactElement | null => {
  const channels = Form.useFieldValue<C[]>(CHANNELS_PATH);
  const { set } = Form.useContext();
  const isPreview = Task.useIsPreview();
  const hovered = Hovered.usePresent();
  const setHovered = useSetHovered();
  const find = useFindOverlaps("Modbus.RegisterMap");
  const [free, setFree] = useState<Free<C>>();
  if (channels.length === 0) return null;
  const { ranges } = find(channels);
  const spans = channels.map((channel) => ({ channel, span: addressSpan(channel) }));
  const lo = Math.min(...spans.map(({ span }) => span[0]));
  const hi = Math.max(...spans.map(({ span }) => span[1]));
  const count = hi - lo + 1;
  const left = (address: number) => `${((address - lo) / count) * 100}%`;
  const width = ([a, b]: [number, number]) => `${((b - a + 1) / count) * 100}%`;
  const hoveredSpan = spans.find(({ channel }) => channel.key === hovered);
  const axisSpan: [number, number] | undefined =
    hoveredSpan?.span ?? (free == null ? undefined : [free.address, free.address]);
  const handleTrackMove = (type: C["type"], e: PointerEvent<HTMLElement>) => {
    if (isPreview) return;
    const box = e.currentTarget.getBoundingClientRect();
    const address = lo + Math.floor(((e.clientX - box.left) / box.width) * count);
    const used = spans.some(
      ({ channel, span }) =>
        channel.type === type && span[0] <= address && address <= span[1],
    );
    setFree(used ? undefined : { type, address });
  };
  const handleAdd = () => {
    if (free == null) return;
    set(CHANNELS_PATH, [...channels, createChannel(free.type, free.address)]);
    setFree(undefined);
  };
  return (
    <div
      className={CSS.cls(
        CSS.BE("modbus", "map"),
        hoveredSpan != null && CSS.BEM("modbus", "map", "hovered"),
      )}
    >
      {REGISTER_TYPES.filter((t): t is C["type"] =>
        channels.some((c) => c.type === t),
      ).map((t) => {
        const lane = spans.filter(({ channel }) => channel.type === t);
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
            <span
              className={CSS.BE("modbus", "track")}
              onPointerMove={(e) => handleTrackMove(t, e)}
              onPointerLeave={() => setFree(undefined)}
              onClick={handleAdd}
            >
              {count <= MAX_TICKS &&
                Array.from({ length: count }, (_, i) => (
                  <span
                    key={i}
                    className={CSS.BE("modbus", "tick")}
                    style={{ left: left(lo + i) }}
                  />
                ))}
              {free?.type === t && (
                <span
                  className={CSS.BE("modbus", "free")}
                  style={{
                    left: left(free.address),
                    width: width([free.address, free.address]),
                  }}
                />
              )}
              {lane.map(({ channel, span }) => (
                <span
                  key={channel.key}
                  className={CSS.cls(
                    Menu.CONTEXT_TARGET,
                    CSS.BE("modbus", "segment"),
                    channel.key === hovered && CSS.BEM("modbus", "segment", "hovered"),
                    channel.disabled === true &&
                      CSS.BEM("modbus", "segment", "disabled"),
                  )}
                  data-menu-key={channel.key}
                  style={{ left: left(span[0]), width: width(span) }}
                  onPointerEnter={() => setHovered(channel.key)}
                  onPointerLeave={() => setHovered()}
                  onPointerMove={(e) => e.stopPropagation()}
                  onClick={(e) => {
                    e.stopPropagation();
                    document
                      .getElementById(channel.key)
                      ?.scrollIntoView({ block: "nearest", behavior: "smooth" });
                  }}
                />
              ))}
              {ranges[t]?.map((span) => (
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
        <span
          className={CSS.cls(
            CSS.BE("modbus", "scale"),
            axisSpan != null && CSS.BEM("modbus", "scale", "pointed"),
          )}
        >
          <span className={CSS.BE("modbus", "axis-end")}>{lo}</span>
          {axisSpan != null && (
            <span
              className={CSS.BE("modbus", "axis-hovered")}
              style={{ left: left(axisSpan[0]), width: width(axisSpan) }}
            >
              {rangeText(axisSpan)}
            </span>
          )}
          <span className={CSS.BE("modbus", "axis-end")}>{hi}</span>
        </span>
      </div>
    </div>
  );
};
