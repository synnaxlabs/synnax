// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Theming } from "@synnaxlabs/lyra/theming";
import { type color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Color } from "@/color";

export const ColorField: Form.FieldT<color.Crude> = (props): ReactElement => {
  const theme = Theming.use();
  return (
    <Form.Field
      label="Color"
      align="start"
      padHelpText={false}
      // An unchosen color is absent, so the swatch shows the theme color it resolves
      // to until a pick writes one.
      defaultValue={theme.colors.gray.l11}
      {...props}
    >
      {({ value, ...rest }) => <Color.Swatch value={value} {...rest} bordered />}
    </Form.Field>
  );
};
