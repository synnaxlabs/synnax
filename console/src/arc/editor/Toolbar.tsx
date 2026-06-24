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

import { Graph } from "@/arc/editor/graph";
import { Text } from "@/arc/editor/text";
import { useSelectExists } from "@/arc/selectors";

export interface ToolbarProps {
  layoutKey: string;
}

const Internal = ({ layoutKey }: ToolbarProps): ReactElement | null => {
  Arc.useEnsureRetrieved({ key: layoutKey });
  const mode = Arc.useSelectMode({ key: layoutKey });
  if (mode === "text") return <Text.Toolbar layoutKey={layoutKey} />;
  return <Graph.Toolbar layoutKey={layoutKey} />;
};

export const Toolbar = (props: ToolbarProps): ReactElement | null => {
  const exists = useSelectExists(props.layoutKey);
  if (!exists) return null;
  return <Internal {...props} />;
};
