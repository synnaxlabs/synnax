// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/perf/components/MetricRow.css";

import { Flex, Input, Text } from "@synnaxlabs/pluto";
import { type bounds } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { type MacroConfig } from "@/perf/macros/types";

const ITERATIONS_BOUNDS: bounds.Bounds = { lower: 1, upper: Infinity };
const DELAY_BOUNDS: bounds.Bounds = { lower: 0, upper: 10000 };

interface ConfigField {
  key: keyof MacroConfig;
  label: string;
  tooltip: string;
  bounds: bounds.Bounds;
}

const CONFIG_FIELDS: ConfigField[] = [
  {
    key: "iterations",
    label: "Iterations",
    tooltip: "Number of times to run selected macros",
    bounds: ITERATIONS_BOUNDS,
  },
  {
    key: "delayBetweenMacrosMs",
    label: "Macro Delay (ms)",
    tooltip: "Milliseconds to wait between macros",
    bounds: DELAY_BOUNDS,
  },
  {
    key: "delayBetweenStepsMs",
    label: "Step Delay (ms)",
    tooltip: "Milliseconds to wait between steps within a macro",
    bounds: DELAY_BOUNDS,
  },
];

export interface MacroConfigInputsProps {
  config: MacroConfig;
  onChange: (config: MacroConfig) => void;
  disabled?: boolean;
}

export const MacroConfigInputs = ({
  config,
  onChange,
  disabled,
}: MacroConfigInputsProps): ReactElement => {
  const handleChange = useCallback(
    (key: keyof MacroConfig, value: number) => {
      onChange({ ...config, [key]: value });
    },
    [config, onChange],
  );

  return (
    <>
      {CONFIG_FIELDS.map((field) => (
        <Flex.Box
          key={field.key}
          x
          justify="between"
          align="center"
          className="console-perf-row"
          title={field.tooltip}
        >
          <Text.Text level="small" className="console-perf-row-label">
            {field.label}
          </Text.Text>
          <Input.Numeric
            value={config[field.key] as number}
            onChange={(v) => handleChange(field.key, v)}
            bounds={field.bounds}
            disabled={disabled}
            showDragHandle={false}
            size="small"
            variant="shadow"
          />
        </Flex.Box>
      ))}
    </>
  );
};
