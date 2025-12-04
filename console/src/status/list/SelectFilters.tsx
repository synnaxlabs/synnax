// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Icon, Label as PLabel, state, Tag, Text } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Label } from "@/label";
import { type Request as ViewRequest, type UseRequestReturn } from "@/view/context";

export interface Request extends ViewRequest {
  hasLabels?: string[];
}

export interface FilterContextMenuProps<R extends Request>
  extends UseRequestReturn<R> {}

export const FilterContextMenu = <R extends Request>({
  request,
  onRequestChange,
}: FilterContextMenuProps<R>): ReactElement => {
  const handleRequestChange = (setter: state.SetArg<R>) => {
    onRequestChange((prev) => {
      const next = state.executeSetter(setter, prev);
      return { ...next, offset: 0, limit: 0 };
    });
  };

  return (
    <Label.SelectMultiple
      value={request.hasLabels ?? []}
      onChange={(labels) => handleRequestChange((r) => ({ ...r, hasLabels: labels }))}
      triggerProps={{ hideTags: true, variant: "text" }}
      location={{ targetCorner: location.TOP_RIGHT, dialogCorner: location.TOP_LEFT }}
    />
  );
};

export interface HasLabelsFilterProps<R extends Request>
  extends Pick<UseRequestReturn<R>, "request"> {}

const HasLabelsFilter = <R extends Request>({
  request,
}: HasLabelsFilterProps<R>): ReactElement | null => {
  const labels =
    PLabel.useRetrieveMultiple({ keys: request.hasLabels ?? [] }).data ?? [];
  if (request.hasLabels == null || request.hasLabels.length === 0) return null;
  return (
    <Flex.Box x pack background={0}>
      <Text.Text
        el="span"
        bordered
        size="small"
        style={{ padding: "0 1rem", boxShadow: "var(--pluto-shadow-v1)" }}
        background={0}
        borderColor={5}
        level="small"
        color={9}
      >
        <Icon.Label />
        Labels
      </Text.Text>
      {labels.map(({ color, key, name }) => (
        <Tag.Tag key={key} color={color} size="small" textColor={9}>
          {name}
        </Tag.Tag>
      ))}
    </Flex.Box>
  );
};

export interface FiltersProps<R extends Request>
  extends Pick<UseRequestReturn<R>, "request"> {}

export const Filters = <R extends Request>({
  request,
}: FiltersProps<R>): ReactElement | null => <HasLabelsFilter request={request} />;
