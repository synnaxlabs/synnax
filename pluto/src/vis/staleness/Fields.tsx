// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { deep, TimeSpan } from "@synnaxlabs/x";
import type { ReactElement } from "react";

const path = deep.path<{ stalenessTimeout?: number }>();

export interface TimeoutFieldProps {
  /** The unit of the stored timeout. Defaults to nanoseconds. */
  unit?: TimeSpan;
}

// The Core stores the timeout as int64 nanoseconds, and a client holds it as a float64,
// so the Core and the client bound it at 2^53 - 1 ns.
const MAX_TIMEOUT = new TimeSpan(Number.MAX_SAFE_INTEGER);

/** @returns why a staleness timeout is out of bounds, or null when it is valid. */
export const checkTimeout = (span: TimeSpan): string | null => {
  if (span.lessThan(TimeSpan.SECOND)) return "Timeout must be 1s or more";
  if (span.greaterThan(MAX_TIMEOUT))
    return `Timeout must be ${MAX_TIMEOUT.toString()} or less`;
  return null;
};

/**
 * Edits the delay after which a component reads stale once its source stops sending. A
 * number typed alone is seconds.
 */
export const TimeoutField = ({
  unit = TimeSpan.NANOSECOND,
}: TimeoutFieldProps): ReactElement => (
  <Form.Field<number | undefined>
    path={path.stalenessTimeout}
    label="Timeout"
    padHelpText={false}
  >
    {({ value, onChange, ...p }) => (
      <Input.TimeSpan
        {...p}
        {...INPUT_PROPS}
        inline
        value={value == null ? undefined : Number(unit.mult(value).valueOf())}
        onChange={(ns) => onChange(ns / Number(unit.valueOf()))}
        check={checkTimeout}
      />
    )}
  </Form.Field>
);

const INPUT_PROPS = { style: { maxWidth: "18rem" } } as const;
