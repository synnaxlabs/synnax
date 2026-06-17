// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Graph, type GraphProps } from "@/arc/Graph";
import { useSelectMode } from "@/arc/queries";
import { Flex } from "@/flex";
import { Text } from "@/text";

export interface EditorProps extends GraphProps {}

// TextEditor is a placeholder for text-mode editing, which is migrated to its
// own collaborative track in a later change.
const TextEditor = (): ReactElement => (
  <Flex.Box grow center>
    <Text.Text level="p" status="disabled">
      Text mode editing is not yet available here.
    </Text.Text>
  </Flex.Box>
);

// Editor is the root Arc editor. It switches on the Arc's representation mode,
// rendering the graph editor for graph mode and the text editor for text mode.
export const Editor = (props: EditorProps): ReactElement | null => {
  const mode = useSelectMode({ key: props.resourceKey });
  if (mode == null) return null;
  if (mode === "text") return <TextEditor />;
  return <Graph {...props} />;
};
