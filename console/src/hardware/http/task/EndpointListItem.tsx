// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Form as PForm, type List, Select, Text } from "@synnaxlabs/pluto";
import { type ReactNode } from "react";

export interface EndpointListItemProps extends List.ItemProps<string> {
  extra?: ReactNode;
}

export const EndpointListItem = ({ extra, ...props }: EndpointListItemProps) => {
  const { itemKey } = props;
  const method = PForm.useFieldValue<string>(`config.endpoints.${itemKey}.method`);
  const epPath = PForm.useFieldValue<string>(`config.endpoints.${itemKey}.path`);
  const shownText = method + (epPath !== "" ? ` ${epPath}` : "");
  return (
    <Select.ListItem justify="between" align="start" {...props}>
      <Text.Text level="small" weight={500}>
        {shownText}
      </Text.Text>
      {extra}
    </Select.ListItem>
  );
};

export const endpointListItem = Component.renderProp(EndpointListItem);
