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
import type { ReactElement } from "react";

import { Color } from "@/color";
import { createFormPath } from "@/label/queries";

export interface FieldsProps extends Pick<
  Input.TextProps,
  "level" | "variant" | "autoFocus"
> {}

/**
 * The color and the name of a label on one line, for the enclosing label form. The
 * props style the name.
 */
export const Fields = (props: FieldsProps): ReactElement => (
  <Flex.Box x align="center" gap="medium">
    <Form.Field path={createFormPath.color} showLabel={false} padHelpText={false}>
      {(p) => <Color.Swatch aria-label="Label color" {...p} />}
    </Form.Field>
    <Form.Field path={createFormPath.name} showLabel={false} padHelpText={false} grow>
      {(p) => (
        <Input.Text
          placeholder="Label name"
          aria-label="Label name"
          full="x"
          {...props}
          {...p}
        />
      )}
    </Form.Field>
  </Flex.Box>
);
