// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import type { deep } from "@synnaxlabs/x";
import { useEffect, useState } from "react";

import { Task } from "@/platform/task";

interface Range {
  minVal: number;
  maxVal: number;
}

interface Held extends Range {
  /** The path of the minimum value the range was typed for. */
  at: string;
  /** The stored range when the hold started. */
  stored: Range;
}

const MESSAGE = "Minimum value must be less than maximum value";

export interface MinMaxValueFieldsProps {
  path: deep.Path<Range>;
  units?: string;
}

/**
 * Edits a channel's range. The Core refuses a range whose minimum is not below its
 * maximum, so the fields hold such a range with an error and save nothing until the
 * range is valid again. A held range stops a deploy. Leaving the channel, or a change
 * to the stored range, drops the held range.
 */
export const MinMaxValueFields = ({ path, units }: MinMaxValueFieldsProps) => {
  const ctx = Form.useContext();
  const minVal = Form.useFieldValue(path.minVal);
  const maxVal = Form.useFieldValue(path.maxVal);
  const at = String(path.minVal);
  const [held, setHeld] = useState<Held>();
  const stale =
    held != null &&
    (held.at !== at || held.stored.minVal !== minVal || held.stored.maxVal !== maxVal);
  if (stale) setHeld(undefined);
  const holding = held != null && !stale;
  const shown: Range = holding ? held : { minVal, maxVal };

  Task.useFlushFirst(async () => !holding);

  useEffect(() => {
    if (!holding) return;
    ctx.setStatus(path.minVal, { variant: "error", message: MESSAGE });
    // A check can replace the hold's error once the range saves, so only the hold's
    // own error is cleared.
    return () => {
      if (ctx.get(path.minVal, { optional: true })?.status.message === MESSAGE)
        ctx.clearStatus(path.minVal);
    };
  }, [ctx, holding, path.minVal]);

  const handleChange = (key: keyof Range) => (value: number) => {
    const next = { ...shown, [key]: value };
    if (next.minVal >= next.maxVal) {
      setHeld({ ...next, at, stored: { minVal, maxVal } });
      return;
    }
    setHeld(undefined);
    ctx.batch(() => {
      ctx.set(path.minVal, next.minVal);
      ctx.set(path.maxVal, next.maxVal);
    });
  };

  return (
    <Input.Item label="Range">
      <Flex.Box x gap="small">
        <Form.Field<number>
          path={path.minVal}
          label="Minimum value"
          showLabel={false}
          showHelpText={false}
        >
          {(p) => (
            <Input.Numeric
              {...p}
              value={shown.minVal}
              onChange={handleChange("minVal")}
              endContent={units}
              grow
              aria-label="Minimum value"
            />
          )}
        </Form.Field>
        <Form.Field<number>
          path={path.maxVal}
          label="Maximum value"
          showLabel={false}
          showHelpText={false}
        >
          {(p) => (
            <Input.Numeric
              {...p}
              value={shown.maxVal}
              onChange={handleChange("maxVal")}
              endContent={units}
              grow
              aria-label="Maximum value"
            />
          )}
        </Form.Field>
      </Flex.Box>
    </Input.Item>
  );
};
