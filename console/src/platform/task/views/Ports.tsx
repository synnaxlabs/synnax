// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

export interface PortsProps {
  /** The keys of the selected channels. */
  selected: string[];
  onSelect: (keys: string[]) => void;
}

export type View = "list" | "ports";

export interface ViewSwitchProps {
  value: View;
  onChange: (view: View) => void;
}

/** Switches the channel column between the list and the channels by port. */
export const ViewSwitch = ({ value, onChange }: ViewSwitchProps): ReactElement => (
  <Select.Buttons<View> size="small" value={value} onChange={onChange}>
    <Select.Item<View> itemKey="list" size="small">
      List
    </Select.Item>
    <Select.Item<View> itemKey="ports" size="small">
      Ports
    </Select.Item>
  </Select.Buttons>
);
