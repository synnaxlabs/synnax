// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/label/filter/Chips.css";

import { type label } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Label } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Errors } from "@/platform/errors";
import { View } from "@/platform/view";

const path = deep.path<{ where?: { labels?: label.Key[] } }>();

/** @returns the names, or a count when there are more than two. */
const summarize = (names: string[]): string =>
  names.length > 2 ? `${names.length} labels` : names.join(", ");

interface ValueProps {
  keys: label.Key[];
}

const Value = ({ keys }: ValueProps): ReactElement => {
  const labels = Label.useMultiple({ keys });
  return (
    <>
      <Flex.Box x empty className={CSS.BE("label-filter", "dots")}>
        {labels.map(({ key, color }) => (
          <Icon.Circle key={key} color={color} />
        ))}
      </Flex.Box>
      {summarize(labels.map(({ name }) => name))}
    </>
  );
};

/** A chip for the labels in the filter of the view. Without labels, it is absent. */
export const Chips = (): ReactElement | null => {
  const field = Form.useField(deep.present(path.where).labels, {
    optional: true,
  });
  const labels = field?.value;
  if (field == null || labels == null || labels.length === 0) return null;
  return (
    <View.FilterChip>
      <View.FilterChipName icon={Icon.Label}>Labels</View.FilterChipName>
      <View.FilterChipOperator>
        {labels.length === 1 ? "has" : "has any of"}
      </View.FilterChipOperator>
      <View.FilterChipValue>
        <Errors.SuspenseBoundary loading={null}>
          <Value keys={labels} />
        </Errors.SuspenseBoundary>
      </View.FilterChipValue>
      <View.FilterChipRemove name="Labels" onClick={() => field.onChange(undefined)} />
    </View.FilterChip>
  );
};
