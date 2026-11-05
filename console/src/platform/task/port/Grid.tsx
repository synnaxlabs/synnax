// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/port/Grid.css";

import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { type CSSProperties, type MouseEvent, type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { useHovered, useSetHovered } from "@/platform/task/port/Hovered";

/** Whether a port is free, holds one channel, or holds more than one. */
export type State = "free" | "used" | "conflict";

export interface Cell {
  /**
   * Unique in the grid. A free cell carries it in `data-menu-key` when the grid has a
   * menu of its own.
   */
  key: string;
  /** The port's name for assistive technology, such as "ai3". */
  name: string;
  /** The port number, then the negative input's number for a differential pair. */
  labels: string[];
  /** The cell's place in its group, from 1. */
  row: number;
  column: number;
  /** Spans two rows: a differential channel uses both ports of the pair. */
  paired?: boolean;
  state: State;
  /** Only disabled channels hold the port: it shows as used, but faded. */
  disabled?: boolean;
  selected?: boolean;
  /**
   * The channel on the port. A used cell carries it in `data-menu-key`, so a right
   * click opens the channel's menu in the enclosing channel list.
   */
  channel?: string;
  /** Every channel on the port. */
  channels: string[];
  tooltip: string;
}

/** Cells laid out together, such as the positive and negative rows of eight pairs. */
export interface Group {
  key: string;
  columns: number;
  cells: Cell[];
}

export interface GridProps {
  /** Names the grid for assistive technology. */
  label: string;
  groups: Group[];
  /** Handles a click that `onAdd` and `onMove` do not. */
  onClick?: (cell: Cell) => void;
  /**
   * Moves a channel to the port, on a click on any port: the grid is a picker. A free
   * port's tooltip then names the move.
   */
  onMove?: (cell: Cell) => void;
  /**
   * Adds a channel on a free port, in place of `onClick`. A free port then shows a
   * plus under the pointer, and its tooltip names the port the click adds on.
   */
  onAdd?: (cell: Cell) => void;
  /**
   * Opens a menu over a free port. A used port leaves the event to the enclosing
   * channel list, which opens its channel's menu. Without it, a free port is not a
   * menu target.
   */
  onContextMenu?: (e: MouseEvent) => void;
}

const cellClass = (
  { state, selected = false, paired = false, disabled = false }: Cell,
  target: boolean,
  addable: boolean,
  hovered: boolean,
): string =>
  CSS.cls(
    target && Menu.CONTEXT_TARGET,
    CSS.BE("port-grid", "cell"),
    CSS.BEM("port-grid", "cell", state),
    addable && CSS.BEM("port-grid", "cell", "addable"),
    hovered && CSS.BEM("port-grid", "cell", "hovered"),
    paired && CSS.BEM("port-grid", "cell", "paired"),
    disabled && CSS.BEM("port-grid", "cell", "disabled"),
    selected && CSS.BEM("port-grid", "cell", "selected"),
  );

const cellStyle = ({ row, column, paired = false }: Cell): CSSProperties => ({
  gridColumn: column,
  gridRow: paired ? `${row} / span 2` : row,
});

/**
 * The ports of a device as a grid of cells colored by state: free, used, or in
 * conflict. A channel's own color never shows, so the channel on a port shows in its
 * tooltip. Under a {@link HoverProvider}, the pointer on a port marks its channels,
 * and a marked channel marks its port.
 */
export const Grid = ({
  label,
  groups,
  onClick,
  onMove,
  onAdd,
  onContextMenu,
}: GridProps): ReactElement => {
  const hovered = useHovered();
  const setHovered = useSetHovered();
  return (
    <div role="group" aria-label={label} className={CSS.B("port-grid")}>
      {groups.map(({ key, columns, cells }) => (
        <div
          key={key}
          className={CSS.BE("port-grid", "group")}
          style={{ gridTemplateColumns: `repeat(${columns}, var(--pluto-port-width))` }}
        >
          {cells.map((cell) => {
            const menuKey =
              cell.channel ?? (onContextMenu == null ? undefined : cell.key);
            const add = cell.state === "free" ? onAdd : undefined;
            const addable = add != null;
            let tooltip = cell.tooltip;
            if (addable) tooltip = `Add a channel on ${cell.name}`;
            else if (onMove != null && cell.state === "free")
              tooltip = `Move to ${cell.name}`;
            const linked = hovered != null && cell.channels.includes(String(hovered));
            const handleClick = add ?? onMove ?? onClick;
            return (
              <Tooltip.Dialog key={cell.key} location="top">
                {tooltip}
                <button
                  type="button"
                  aria-label={cell.name}
                  aria-pressed={cell.selected ?? false}
                  data-menu-key={menuKey}
                  className={cellClass(cell, menuKey != null, addable, linked)}
                  style={cellStyle(cell)}
                  onClick={() => handleClick?.(cell)}
                  onPointerEnter={() => {
                    if (cell.channel != null) setHovered(cell.channel);
                  }}
                  onPointerLeave={() => {
                    if (cell.channel != null) setHovered();
                  }}
                  onContextMenu={cell.state === "free" ? onContextMenu : undefined}
                >
                  {cell.labels.map((l) => (
                    <span key={l}>{l}</span>
                  ))}
                  {addable && <Icon.Add className={CSS.BE("port-grid", "add")} />}
                </button>
              </Tooltip.Dialog>
            );
          })}
        </div>
      ))}
    </div>
  );
};

export interface BadgeProps {
  state: State;
  /** The channel is disabled: the badge shows faded, as the grid does. */
  disabled?: boolean;
  children: string;
}

/** A port's number in its state color, as the grid shows it. */
export const Badge = ({
  state,
  disabled = false,
  children,
}: BadgeProps): ReactElement => (
  <span
    className={CSS.cls(
      CSS.B("port-badge"),
      CSS.BM("port-badge", state),
      disabled && CSS.BM("port-badge", "disabled"),
    )}
  >
    {children}
  </span>
);
