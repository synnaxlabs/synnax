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
import { type color, type notation } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { ColorField } from "@/platform/multiedit/Colors";
import { type ColorValue } from "@/platform/multiedit/selection";

/**
 * Holds the color controls of several elements. The label column keeps its width as
 * the Selection control appears and leaves.
 */
export const ColorsSection = ({ children }: PropsWithChildren): ReactElement => (
  <Form.Section title="Colors" className={CSS.B("multiedit-colors")}>
    {children}
  </Form.Section>
);

const STALENESS_TIMEOUT_BOUNDS = { lower: 1, upper: Infinity };
const PRECISION_BOUNDS = { lower: 0, upper: 10 };

export interface StalenessSectionProps {
  /** The staleness color of each element that has the field. */
  colors: ColorValue[];
  timeout?: number;
  onColorChange: (value?: color.Color) => void;
  onTimeoutChange: (value: number) => void;
}

/** Edits the staleness color and timeout of several elements. */
export const StalenessSection = ({
  colors,
  timeout = Staleness.DEFAULT_TIMEOUT,
  onColorChange,
  onTimeoutChange,
}: StalenessSectionProps): ReactElement => (
  <Form.Section title="Staleness">
    <ColorField label="Color" values={colors} onChange={onColorChange} />
    <Input.Item label="Timeout" align="start" padHelpText={false}>
      <Input.Numeric
        bounds={STALENESS_TIMEOUT_BOUNDS}
        endContent="s"
        value={timeout}
        onChange={onTimeoutChange}
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
  notation = "standard",
  precision = 2,
  onNotationChange,
  onPrecisionChange,
}: NumberFormatSectionProps): ReactElement => (
  <Form.Section title="Number format">
    <Input.Item label="Notation" align="start" padHelpText={false}>
      <Notation.Select value={notation} onChange={onNotationChange} />
    </Input.Item>
    <Input.Item label="Precision" align="start" padHelpText={false}>
      <Input.Numeric
        bounds={PRECISION_BOUNDS}
        value={precision}
        onChange={onPrecisionChange}
      />
    </Input.Item>
  </Form.Section>
);
