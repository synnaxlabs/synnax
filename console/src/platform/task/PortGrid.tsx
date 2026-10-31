// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/PortGrid.css";

import { Menu } from "@synnaxlabs/lyra/menu";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { type CSSProperties, type MouseEvent, type ReactElement } from "react";

import { CSS } from "@/platform/css";

/** Whether a port is free, holds one channel, or holds more than one. */
export type PortState = "free" | "used" | "conflict";

export interface PortCell {
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
  state: PortState;
  /** Only disabled channels hold the port: it shows as used, but faded. */
  disabled?: boolean;
  selected?: boolean;
  /**
   * The channel on the port. A used cell carries it in `data-menu-key`, so a right
   * click opens the channel's menu in the enclosing channel list.
   */
  channel?: string;
  tooltip: string;
}

/** Cells laid out together, such as the positive and negative rows of eight pairs. */
export interface PortGroup {
  key: string;
  columns: number;
  cells: PortCell[];
}

export interface PortGridProps {
  /** Names the grid for assistive technology. */
  label: string;
  groups: PortGroup[];
  onClick: (cell: PortCell) => void;
  /**
   * Opens a menu over a free port. A used port leaves the event to the enclosing
   * channel list, which opens its channel's menu. Without it, a free port is not a
   * menu target.
   */
  onContextMenu?: (e: MouseEvent) => void;
}

const cellClass = (
  { state, selected = false, paired = false, disabled = false }: PortCell,
  target: boolean,
): string =>
  CSS.cls(
    target && Menu.CONTEXT_TARGET,
    CSS.BE("port-grid", "cell"),
    CSS.BEM("port-grid", "cell", state),
    paired && CSS.BEM("port-grid", "cell", "paired"),
    disabled && CSS.BEM("port-grid", "cell", "disabled"),
    selected && CSS.BEM("port-grid", "cell", "selected"),
  );

const cellStyle = ({ row, column, paired = false }: PortCell): CSSProperties => ({
  gridColumn: column,
  gridRow: paired ? `${row} / span 2` : row,
});

/**
 * The ports of a device as a grid of cells colored by state: free, used, or in
 * conflict. A channel's own color never shows, so the channel on a port shows in its
 * tooltip.
 */
export const PortGrid = ({
  label,
  groups,
  onClick,
  onContextMenu,
}: PortGridProps): ReactElement => (
  <div role="group" aria-label={label} className={CSS.B("port-grid")}>
    {groups.map(({ key, columns, cells }) => (
      <div
        key={key}
        className={CSS.BE("port-grid", "group")}
        style={{ gridTemplateColumns: `repeat(${columns}, var(--console-port-cell))` }}
      >
        {cells.map((cell) => {
          const menuKey =
            cell.channel ?? (onContextMenu == null ? undefined : cell.key);
          return (
            <Tooltip.Dialog key={cell.key} location="top">
              {cell.tooltip}
              <button
                type="button"
                aria-label={cell.name}
                aria-pressed={cell.selected ?? false}
                data-menu-key={menuKey}
                className={cellClass(cell, menuKey != null)}
                style={cellStyle(cell)}
                onClick={() => onClick(cell)}
                onContextMenu={cell.state === "free" ? onContextMenu : undefined}
              >
                {cell.labels.map((l) => (
                  <span key={l}>{l}</span>
                ))}
              </button>
            </Tooltip.Dialog>
          );
        })}
      </div>
    ))}
  </div>
);

export interface PortBadgeProps {
  state: PortState;
  /** The channel is disabled: the badge shows faded, as the grid does. */
  disabled?: boolean;
  children: string;
}

/** A port's number in its state color, as the grid shows it. */
export const PortBadge = ({
  state,
  disabled = false,
  children,
}: PortBadgeProps): ReactElement => (
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
