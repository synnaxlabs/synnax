// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/device/Location.css";

import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/css";

/**
 * How a device's location reads. A slot shows its last part, an address its host, a
 * list its first address and a count of the rest, and a plain location the whole value.
 */
export type LocationShape = "slot" | "address" | "list" | "plain";

export interface LocationFormat {
  shape: LocationShape;
  /** The schemes, such as "https://", that the location hides from an address. */
  hiddenSchemes?: string[];
}

const SEPARATORS = ".:/\\";

const lastPart = (value: string): string => {
  for (let i = value.length - 2; i > 0; i--)
    if (SEPARATORS.includes(value[i])) return value.slice(i + 1);
  return value;
};

const SCHEME = /^[a-z][a-z0-9+.-]*:\/\//i;

const splitAddress = (value: string): [string, string, string] => {
  const scheme = SCHEME.exec(value)?.[0] ?? "";
  const rest = value.slice(scheme.length);
  let end: number;
  if (rest.startsWith("[")) {
    const close = rest.indexOf("]");
    end = close === -1 ? rest.length : close + 1;
  } else {
    const slash = rest.indexOf("/");
    const authority = slash === -1 ? rest : rest.slice(0, slash);
    // More than one colon is an IPv6 host without brackets, so no port follows it.
    const at = authority.split(":").length > 2 ? -1 : authority.indexOf(":");
    end = at === -1 ? authority.length : at;
  }
  return [scheme, rest.slice(0, end), rest.slice(end)];
};

const DIM = CSS.BE("device-location", "dim");
const BRIGHT = CSS.BE("device-location", "bright");

interface Shown {
  /** The telling part, kept in view while the location has room. */
  lead: ReactNode;
  /** The rest, which gives up its width first. */
  tail?: ReactNode;
  keep: number;
  /** The text that shows, without the count. */
  text: string;
}

const showWhole = (value: string): Shown => ({
  lead: <span className={BRIGHT}>{value}</span>,
  keep: value.length,
  text: value,
});

const showAddress = (value: string, hiddenSchemes: string[] = []): Shown => {
  const [scheme, host, rest] = splitAddress(value);
  if (host === "") return showWhole(value);
  const shown = hiddenSchemes.includes(scheme.toLowerCase()) ? "" : scheme;
  return {
    lead: (
      <>
        {shown !== "" && <span className={DIM}>{shown}</span>}
        <span className={BRIGHT}>{host}</span>
      </>
    ),
    tail: rest === "" ? undefined : <span className={DIM}>{rest}</span>,
    keep: shown.length + host.length,
    text: shown + host + rest,
  };
};

const show = (value: string, { shape, hiddenSchemes }: LocationFormat): Shown => {
  switch (shape) {
    case "slot":
      return showWhole(lastPart(value));
    case "address":
      return showAddress(value, hiddenSchemes);
    case "list":
      return showAddress(value.split(",")[0].trim(), hiddenSchemes);
    case "plain":
      return showWhole(value);
  }
};

const countOthers = (value: string, { shape }: LocationFormat): number =>
  shape === "list" ? value.split(",").filter((v) => v.trim() !== "").length - 1 : 0;

interface PartProps {
  className: string;
  children: ReactNode;
}

const Part = ({ className, children }: PartProps): ReactElement => (
  <Text.Text
    el="span"
    level="small"
    variant="code"
    overflow="fade"
    className={className}
  >
    {children}
  </Text.Text>
);

export interface LocationProps {
  value: string;
  /** How the location reads. Plain when absent. */
  format?: LocationFormat;
  className?: string;
}

const PLAIN: LocationFormat = { shape: "plain" };

const showsAll = (location: HTMLElement): boolean =>
  Array.from(location.children).every((part) => part.scrollWidth <= part.clientWidth);

/**
 * Shows the telling part of a device's location on one line, brightest where it differs
 * most. A tooltip shows the whole value while any part is hidden. Set `min-width` to
 * `calc(var(--pluto-location-keep) * 1ch)` to keep that part in view while the rest
 * gives up its width.
 */
export const Location = ({
  value,
  format = PLAIN,
  className,
}: LocationProps): ReactElement => {
  const { lead, tail, keep, text } = show(value, format);
  const count = countOthers(value, format);
  const countText = count > 0 ? `+${count}` : "";
  const style: CSS.VarProperties = {
    "--pluto-location-keep": keep + (countText === "" ? 0 : countText.length + 1),
  };
  return (
    <Tooltip.Dialog hide={text === value ? showsAll : false}>
      {value}
      <Text.Text
        el="span"
        level="small"
        variant="code"
        className={CSS.cls(CSS.B("device-location"), className)}
        style={style}
      >
        <Part className={CSS.BE("device-location", "lead")}>{lead}</Part>
        {tail != null && (
          <Part className={CSS.BE("device-location", "tail")}>{tail}</Part>
        )}
        {countText !== "" && (
          <span className={CSS.cls(CSS.BE("device-location", "count"), DIM)}>
            {countText}
          </span>
        )}
      </Text.Text>
    </Tooltip.Dialog>
  );
};
