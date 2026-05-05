// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Color } from "@/color";
import { Form as Base } from "@/form";

export const ColorField: Base.FieldT<color.Crude> = (props): ReactElement => (
  <Base.Field hideIfNull label="Color" align="start" padHelpText={false} {...props}>
    {({ value, onChange, variant: _, ...rest }) => (
      <Color.Swatch value={value} onChange={onChange} {...rest} bordered />
    )}
  </Base.Field>
);
