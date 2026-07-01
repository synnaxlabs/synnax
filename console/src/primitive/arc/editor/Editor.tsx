// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@synnaxlabs/pluto";

import { Graph, type GraphProps } from "@/primitive/arc/editor/Graph";
import { Text } from "@/primitive/arc/editor/Text";

export interface EditorProps extends GraphProps {}

export const Editor = (props: EditorProps) => {
  const mode = Arc.useSelectMode();
  const C = mode === "graph" ? Graph : Text;
  return <C {...props} />;
};
