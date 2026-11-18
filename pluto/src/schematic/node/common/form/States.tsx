// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration, type schematic } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { type deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Enumeration } from "@/enumeration";
import { StateMappingForm } from "@/schematic/node/common/form/StateMapping";

export interface StatesProps {
  /** Path to the enumeration that the node names. */
  enumeration: deep.Path<enumeration.Key | undefined>;
  /** Path to the options that the node draws when it names no enumeration. */
  options: deep.Path<schematic.StateMapping[]>;
  showColor?: boolean;
}

/** Picks the enumeration that a node draws, or edits its options when it has none. */
export const States = ({
  enumeration: enumerationPath,
  options,
  showColor,
}: StatesProps): ReactElement => {
  const { set, remove } = Form.useContext();
  const key = Form.useFieldValue(enumerationPath, { optional: true }) ?? undefined;
  const handleChange = (next: enumeration.Key | undefined): void => {
    if (next == null) remove(enumerationPath);
    else set(enumerationPath, next);
  };
  return (
    <Flex.Box y gap="small" grow>
      <Input.Item label="Enumeration" padHelpText={false}>
        <Enumeration.SelectSingle value={key} onChange={handleChange} allowNone />
      </Input.Item>
      {key == null && <StateMappingForm path={options} showColor={showColor} />}
    </Flex.Box>
  );
};
