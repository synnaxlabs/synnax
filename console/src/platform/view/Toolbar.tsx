// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Divider } from "@synnaxlabs/lyra/divider";
import { type ReactElement } from "react";

import { Search } from "@/platform/search";
import { useContext } from "@/platform/view/context";

export interface ToolbarProps extends Search.BarProps {}

export const Toolbar = (props: ToolbarProps): ReactElement => {
  const { editable } = useContext("View.Toolbar");
  if (!editable) return <Divider.Divider x />;
  return <Search.Bar {...props} />;
};
