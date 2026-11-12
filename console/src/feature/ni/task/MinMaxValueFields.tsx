// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Bounds } from "@synnaxlabs/lyra/bounds";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { type bounds, type deep } from "@synnaxlabs/x";

import { type Device } from "@/feature/ni/device/types";

export interface MinMaxValueFieldsProps {
  path: deep.Path<{ minVal: number; maxVal: number }>;
  units?: string;
  /** The ranges that the device takes. When not empty, a select picks one. */
  ranges?: bounds.Bounds[];
}

export const MinMaxValueFields = ({
  path,
  units,
  ranges = [],
}: MinMaxValueFieldsProps) => {
  const ctx = Form.useContext();
  const min = Form.useFieldValue(path.minVal);
  const max = Form.useFieldValue(path.maxVal);
  const handleRangeChange = ({ lower, upper }: bounds.Bounds) =>
    ctx.batch(() => {
      ctx.set(path.minVal, lower);
      ctx.set(path.maxVal, upper);
    });
  return (
    <Input.Item label="Range">
      <Flex.Box x gap="small">
        {ranges.length > 0 && (
          <Bounds.Select
            data={ranges}
            value={{ lower: min, upper: max }}
            onChange={handleRangeChange}
            units={units}
            resourceName="range"
          />
        )}
        <Form.NumericField
          path={path.minVal}
          label="Minimum value"
          showLabel={false}
          showHelpText={false}
          inputProps={{ endContent: units, grow: true, "aria-label": "Minimum value" }}
        />
        <Form.NumericField
          path={path.maxVal}
          label="Maximum value"
          showLabel={false}
          showHelpText={false}
          inputProps={{ endContent: units, grow: true, "aria-label": "Maximum value" }}
        />
      </Flex.Box>
    </Input.Item>
  );
};

type RangeList =
  "aiVoltageRanges" | "aiCurrentRanges" | "aoVoltageRanges" | "aoCurrentRanges";

export interface DeviceRangeFieldsProps {
  path: deep.Path<{ minVal: number; maxVal: number; customScale: { type: string } }>;
  device: Device | undefined;
  /** The device ranges that the channel type uses. */
  list: RangeList;
  /** The units of the ranges. */
  units: string;
}

/**
 * The minimum and maximum of a voltage or current channel. With no custom scale, a
 * select also picks one of the device ranges. A custom scale puts the values in scaled
 * units, which the ranges do not use.
 */
export const DeviceRangeFields = ({
  path,
  device,
  list,
  units,
}: DeviceRangeFieldsProps) => {
  const scale = Form.useFieldValue(path.customScale.type);
  if (scale !== "none") return <MinMaxValueFields path={path} />;
  return (
    <MinMaxValueFields
      path={path}
      units={units}
      ranges={device?.properties.hardware?.[list]}
    />
  );
};
