// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Graph } from "@/component/arc/editor/toolbar/graph";
import { Text } from "@/component/arc/editor/toolbar/text";

interface ToolbarProps extends Graph.ToolbarProps {}

export const Toolbar = (props: ToolbarProps): ReactElement | null => {
  const mode = Arc.useSelectMode();
  const C = mode === "text" ? Text.Toolbar : Graph.Toolbar;
  return <C {...props} />;
};
