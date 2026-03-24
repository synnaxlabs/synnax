// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/http/task/EndpointListItem.css";

import { Component, Form as PForm, type List, Select, Text } from "@synnaxlabs/pluto";
import { type ReactNode } from "react";

import { CSS } from "@/css";

export interface EndpointListItemProps extends List.ItemProps<string> {
  extra?: ReactNode;
  textProps?: Text.TextProps;
}

export const EndpointListItem = ({
  extra,
  textProps,
  ...props
}: EndpointListItemProps) => {
  const { itemKey } = props;
  const method = PForm.useFieldValue<string>(`config.endpoints.${itemKey}.method`);
  const epPath = PForm.useFieldValue<string>(`config.endpoints.${itemKey}.path`);
  const shownText = method + (epPath !== "" ? ` ${epPath}` : "");
  // Escaping to allow for proper ellipsis handling.
  return (
    <Select.ListItem justify="between" align="start" {...props}>
      <Text.Text
        level="small"
        weight={500}
        className={CSS.BE("http-endpoint-list-item", "text")}
        {...textProps}
      >
        {`\u2066${shownText}\u2069`}
      </Text.Text>
      {extra}
    </Select.ListItem>
  );
};

export const endpointListItem = Component.renderProp(EndpointListItem);
