// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, CSS as PCSS, Flex, Form, Icon, Input, Text } from "@synnaxlabs/pluto";

import { CSS } from "@/platform/css";

export interface CoefficientsFieldProps extends Omit<Flex.BoxProps, "children"> {
  path: string;
  label: string;
}

/**
 * Edits the coefficients of a polynomial, ordered from the constant term upwards, so
 * that index i holds the coefficient of x^i.
 * @param path - The form path of the coefficient array.
 * @param label - The heading shown above the rows.
 */
export const CoefficientsField = ({
  path,
  label,
  ...rest
}: CoefficientsFieldProps): React.ReactElement => {
  const { value, onChange, preview } = Form.useField<number[]>(path);
  const handleChange = (index: number, next: number) =>
    onChange(value.map((coeff, i) => (i === index ? next : coeff)));
  return (
    <Flex.Box y gap="small" className={CSS.B("coefficients")} {...rest}>
      <Text.Text level="small" justify="between" color={9}>
        {label}
        {!preview && (
          <Button.Button
            onClick={() => onChange([...value, 0])}
            variant="filled"
            tooltip={`Add ${label.toLowerCase()}`}
            size="small"
          >
            <Icon.Add />
          </Button.Button>
        )}
      </Text.Text>
      <Flex.Box y gap="small">
        {value.map((coeff, i) => (
          <Flex.Box
            x
            key={i}
            align="center"
            gap="small"
            className={CSS.cls(CSS.B("coefficient-row"), PCSS.M("reveals"))}
          >
            <Input.Numeric
              value={coeff}
              onChange={(next) => handleChange(i, next)}
              preview={preview}
              startContent={
                <Text.Text level="small" color={9}>
                  c{i}
                </Text.Text>
              }
              grow
            />
            {!preview && (
              <Button.Button
                variant="text"
                reveal
                size="small"
                tooltip={`Remove c${i}`}
                onClick={() => onChange(value.filter((_, j) => j !== i))}
              >
                <Icon.Close />
              </Button.Button>
            )}
          </Flex.Box>
        ))}
      </Flex.Box>
    </Flex.Box>
  );
};
