// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Graph } from "@/arc/editor/graph";
import { Text } from "@/arc/editor/text";
import { useSelectOptionalMode } from "@/arc/selectors";

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  const mode = useSelectOptionalMode(layoutKey);
  if (mode == null) return null;
  if (mode === "text") return <Text.Toolbar layoutKey={layoutKey} />;
  return <Graph.Toolbar layoutKey={layoutKey} />;
};
