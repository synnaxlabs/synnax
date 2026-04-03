// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/status/filter/Chips.css";

import { Flex, Form, Status, Tag, Text } from "@synnaxlabs/pluto";
import { caseconv, type status } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { View } from "@/view";

export const Chips = (): ReactElement | null => {
  const { editable } = View.useContext();
  const field = Form.useField<status.Variant[]>("query.variants", {
    optional: true,
  });
  const variants = field?.value;
  if (variants == null || variants.length === 0 || field == null) return null;
  const handleClose = (variant: status.Variant) =>
    field.onChange(variants.filter((v) => v !== variant));
  return (
    <Flex.Box x pack background={0}>
      <Text.Text
        bordered
        className={CSS.BE("status-variant", "filter-chips")}
        size="small"
        borderColor={5}
        level="small"
      >
        Variants
      </Text.Text>
      {variants.map((variant) => (
        <Tag.Tag
          key={variant}
          icon={<Status.Indicator variant={variant} />}
          size="small"
          onClose={editable ? () => handleClose(variant) : undefined}
        >
          {caseconv.capitalize(variant)}
        </Tag.Tag>
      ))}
    </Flex.Box>
  );
};
