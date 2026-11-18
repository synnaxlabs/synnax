// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, lineplot, panel, type ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { LinePlot, Panel as PPanel, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Panel } from "@/platform/panel";

interface PlotItemProps {
  plot: lineplot.Key;
  ranges: ranger.Range[];
  onAdd: (plot: lineplot.Key) => void;
}

/** A plot that already shows every one of the ranges stays in the list, checked. */
const PlotItem = ({ plot, ranges, onAdd }: PlotItemProps): ReactElement => {
  const name = LinePlot.useName({ key: plot });
  const shown = LinePlot.useRanges({ key: plot }) ?? [];
  const added = ranges.every((r) =>
    shown.some((ref) => ref.variant === "key" && ref.key === r.key),
  );
  return (
    <Menu.Item
      itemKey={plot}
      textValue={name}
      disabled={added}
      onClick={() => onAdd(plot)}
    >
      <Icon.LinePlot />
      <Text.Text overflow="fade" className={CSS.BE("range-menu", "plot")}>
        {name ?? "Line plot"}
      </Text.Text>
      {added && <Icon.Check className={CSS.BE("range-menu", "added")} />}
    </Menu.Item>
  );
};

export interface AddToPlotProps {
  ranges: ranger.Range[];
}

/** Adds the ranges to a line plot open in the panel. Absent with none open. */
export const AddToPlot = (props: AddToPlotProps): ReactElement | null => {
  const panelKey = Panel.useActiveKey();
  return panelKey == null ? null : <Plots {...props} panelKey={panelKey} />;
};

interface PlotsProps extends AddToPlotProps {
  panelKey: panel.Key;
}

const Plots = ({ ranges, panelKey }: PlotsProps): ReactElement | null => {
  const client = Synnax.use();
  const root = PPanel.useRoot({ key: panelKey });
  const openTab = Panel.useOpenTab();
  const handleError = Status.useErrorHandler();
  const plots = panel
    .tabs(root)
    .flatMap((t) =>
      t.variant === "resource" && t.resource.type === lineplot.TYPE_RESOURCE_ID.type
        ? [t.resource.key]
        : [],
    );
  if (plots.length === 0) return null;
  const handleAdd = (plot: lineplot.Key) =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      await client.lineplots.dispatch(
        plot,
        ranges.map((r) => lineplot.addRange({ range: { variant: "key", key: r.key } })),
      );
      openTab({ variant: "resource", resource: lineplot.resourceID(plot) });
    }, "Failed to add ranges to plot");
  return (
    <Menu.Sub itemKey="addToPlot" textValue="Add to plot">
      <Menu.SubTrigger>
        <Icon.LinePlot />
        Add to plot
      </Menu.SubTrigger>
      <Menu.Dialog aria-label="Add to plot">
        <Menu.Menu gap="small">
          {plots.map((plot) => (
            <PlotItem key={plot} plot={plot} ranges={ranges} onAdd={handleAdd} />
          ))}
        </Menu.Menu>
      </Menu.Dialog>
    </Menu.Sub>
  );
};
