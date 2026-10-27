// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Status } from "@synnaxlabs/lyra/status";
import { Tag } from "@synnaxlabs/lyra/tag";
import { caseconv } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { getSelected } from "@/platform/status/filter/selected";
import { View } from "@/platform/view";

export const Chips = (): ReactElement | null => {
  const { editable } = View.useContext();
  const field = Form.useField<status.Where["variant"]>("where.variant", {
    optional: true,
  });
  const variants = getSelected(field?.value);
  if (variants.length === 0 || field == null) return null;
  const handleClose = (variant: status.Variant) => {
    const next = variants.filter((v) => v !== variant);
    field.onChange(next.length === 0 ? undefined : { in: next });
  };
  return (
    <Flex.Box x pack background={0}>
      <View.FilterChip>Variants</View.FilterChip>
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
