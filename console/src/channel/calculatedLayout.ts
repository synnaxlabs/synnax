// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";

import { type Layout } from "@/layout";

export const CALCULATED_LAYOUT_TYPE = "createCalculatedChannel";

export interface CalculatedLayoutArgs {
  channelKey?: number;
}

export interface CalculatedLayout extends Layout.BaseState<CalculatedLayoutArgs> {}

export const CALCULATED_LAYOUT: CalculatedLayout = {
  name: "Channel.Create.Calculated",
  icon: "Channel",
  location: "modal",
  tab: { closable: true, editable: false },
  window: {
    resizable: false,
    size: { height: 600, width: 1000 },
    navTop: true,
    showTitle: true,
  },
  type: CALCULATED_LAYOUT_TYPE,
  key: CALCULATED_LAYOUT_TYPE,
};

export interface CreateCalculatedLayoutArgs {
  key: channel.Key;
  name: channel.Name;
}

export const createCalculatedLayout = ({
  key,
  name,
}: CreateCalculatedLayoutArgs): CalculatedLayout => ({
  ...CALCULATED_LAYOUT,
  args: { channelKey: key },
  name: `${name}.Edit`,
});
