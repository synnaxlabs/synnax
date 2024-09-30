// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/status/Status.css";

import { Align } from "@synnaxlabs/pluto";
import { Text } from "@synnaxlabs/pluto/text";
import { type ReactElement } from "react";
import { v4 as uuidv4 } from "uuid";

import { CSS } from "@/css";
import { RackList } from "@/hardware/RackList";
import { type Layout } from "@/layout";

export const Status: Layout.Renderer = (): ReactElement => {
  return (
    <Align.Space className={CSS.B("hardware-status")}>
      <Text.Text level="h3" weight={400}>
        Hardware Status
      </Text.Text>
      <RackList />;
    </Align.Space>
  );
};

export type LayoutType = "hardwareStatus";
export const LAYOUT_TYPE: LayoutType = "hardwareStatus";

export const create =
  (initial: Omit<Partial<Layout.State>, "type">): Layout.Creator =>
  () => {
    const { name = "Hardware Status", location = "mosaic", ...rest } = initial;
    return {
      key: initial.key ?? uuidv4(),
      type: LAYOUT_TYPE,
      name,
      location,
      ...rest,
    };
  };
