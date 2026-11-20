// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Theming } from "@synnaxlabs/lyra/theming";
import type { ReactElement } from "react";

import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { Options } from "@/schematic/node/common/options";
import { Primitive } from "@/schematic/node/common/primitive";
import { Telem } from "@/schematic/node/common/telem";
import { optionName, Select } from "@/schematic/node/general/select/Primitive";
import type { NodeProps } from "@/schematic/node/spec";
import { Setpoint as BaseSetpoint } from "@/vis/setpoint";
import { Staleness } from "@/vis/staleness";
import { StateIndicator } from "@/vis/stateIndicator";

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config: {
    label,
    orientation = "left",
    control,
    fillColor,
    commandChannel,
    stateChannel,
    stalenessColor,
    stalenessTimeout,
    options: ownOptions,
    enumeration,
    size,
    disabled,
    inlineSize,
    onClickDelay,
  },
}: NodeProps<schematic.SelectNodeConfig>): ReactElement => {
  const theme = Theming.use();
  const badged = Staleness.useBadged();
  const options = Options.use({ enumeration, options: ownOptions });
  const sink = Telem.numberSink(commandChannel);
  const source = Telem.numberSource(commandChannel);
  const { set, value } = BaseSetpoint.use({ aetherKey: nodeKey, sink, source });
  const { key: readbackKey, stale } = StateIndicator.use({
    aetherKey: `${nodeKey}-readback`,
    source: Telem.numberSource(stateChannel),
    options,
    stalenessTimeout,
  });
  const readback = options.find((o) => o.key === readbackKey);
  const resize = Grid.useInlineResize(inlineSize, (inlineSize) =>
    onConfigChange({ inlineSize }),
  );
  const target = options.find((o) => o.value === value)?.key ?? null;
  return (
    <Grid.Grid nodeKey={nodeKey} allowRotate={false} editable={selected} {...resize}>
      <Control.State
        config={control}
        channel={commandChannel}
        onChange={onConfigChange}
      >
        {stale && badged ? <Staleness.Badge /> : null}
      </Control.State>
      <Label.Label config={label} onChange={onConfigChange} />
      <Select
        value={target}
        onSend={set}
        fillColor={fillColor}
        orientation={orientation}
        disabled={disabled}
        options={options}
        size={size}
        inlineSize={inlineSize}
        onClickDelay={onClickDelay}
        readback={
          stateChannel !== 0 && (
            <Primitive.Readout
              size={size}
              value={readback == null ? "-" : optionName(readback)}
              color={
                stale && !badged
                  ? Staleness.resolveColor(stalenessColor, theme)
                  : undefined
              }
            />
          )
        }
      />
    </Grid.Grid>
  );
};
