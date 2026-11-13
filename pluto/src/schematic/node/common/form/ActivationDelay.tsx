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
import { type ReactElement } from "react";

const path = deep.path<{ onClickDelay?: number }>();

// A browser timer fires at once past 2^31 - 1 ms, which would skip the hold. The Core
// stores the same bound.
const MAX_DELAY = TimeSpan.milliseconds(2 ** 31 - 1);

const checkDelay = (span: TimeSpan): string | null =>
  span.greaterThan(MAX_DELAY) ? `Delay must be ${MAX_DELAY.toString()} or less` : null;

/** Edits the click delay in nanoseconds. A number typed alone is milliseconds. */
export const ActivationDelayField = (): ReactElement => (
  <Form.Field<number | undefined>
    label="Delay"
    path={path.onClickDelay}
    padHelpText={false}
  >
    {({ value, onChange, ...p }) => (
      <Input.TimeSpan
        {...p}
        inline
        unit={TimeSpan.MILLISECOND}
        value={value}
        onChange={onChange}
        check={checkDelay}
      />
    )}
  </Form.Field>
);
