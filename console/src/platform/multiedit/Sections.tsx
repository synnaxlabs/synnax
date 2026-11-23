// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/multiedit/Sections.css";

import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Notation, Staleness } from "@synnaxlabs/pluto";
import { notation } from "@synnaxlabs/x";
import type { PropsWithChildren, ReactElement } from "react";

import { CSS } from "@/platform/css";

/**
 * Holds the color controls of several elements. The label column keeps its width as
 * the Selection control appears and leaves.
 */
export const ColorsSection = ({ children }: PropsWithChildren): ReactElement => (
  <Form.Section title="Colors" className={CSS.B("multiedit-colors")}>
    {children}
  </Form.Section>
);

export interface StalenessSectionProps extends PropsWithChildren {
  /** The timeout in nanoseconds. */
  timeout?: number;
  onTimeoutChange: (value: number) => void;
}

/** Edits the staleness timeout of several elements, below the fields in children. */
export const StalenessSection = ({
  children,
  timeout,
  onTimeoutChange,
}: StalenessSectionProps): ReactElement => (
  <Form.Section title="Staleness">
    {children}
    <Input.Item label="Timeout" align="start" padHelpText={false}>
      <Input.TimeSpan
        inline
        value={timeout ?? Staleness.DEFAULT_TIMEOUT}
        onChange={onTimeoutChange}
        check={Staleness.checkTimeout}
      />
    </Input.Item>
  </Form.Section>
);

export interface NumberFormatSectionProps {
  notation?: notation.Notation;
  precision?: number;
  onNotationChange: (value: notation.Notation) => void;
  onPrecisionChange: (value: number) => void;
}

/** Edits the notation and precision of several elements. */
export const NumberFormatSection = ({
  notation: selected = "standard",
  precision = 2,
  onNotationChange,
  onPrecisionChange,
}: NumberFormatSectionProps): ReactElement => (
  <Form.Section title="Number format">
    <Input.Item label="Notation" align="start" padHelpText={false}>
      <Notation.Select value={selected} onChange={onNotationChange} />
    </Input.Item>
    <Input.Item label="Precision" align="start" padHelpText={false}>
      <Input.Numeric
        value={precision}
        onChange={onPrecisionChange}
        integer
        bounds={notation.PRECISION_BOUNDS}
      />
    </Input.Item>
  </Form.Section>
);
