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
import { Form } from "@/form";
import { resolveColor } from "@/schematic/node/common/color";
import { Theming } from "@/theming";

export const ColorField: Form.FieldT<color.Crude> = (props): ReactElement => {
  const theme = Theming.use();
  return (
    <Form.Field hideIfNull label="Color" align="start" padHelpText={false} {...props}>
      {({ value, onChange, variant: _, ...rest }) => (
        <Color.Swatch
          value={resolveColor(value, theme)}
          onChange={onChange}
          {...rest}
          bordered
        />
      )}
    </Form.Field>
  );
};
