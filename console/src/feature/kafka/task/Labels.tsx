// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import type { ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

const TOPIC_KIND_COLOR = "var(--pluto-primary-z)";

/** Names a topic in a task tree, or stands in for a topic with no name. */
export const TopicLabel = ({ name }: { name: string }): ReactElement => (
  <Task.Views.ItemLabel
    kind="Topic"
    kindColor={TOPIC_KIND_COLOR}
    color={name === "" ? 8 : 10}
    variant="code"
  >
    {name === "" ? "New topic" : name}
  </Task.Views.ItemLabel>
);

export interface RecordLabelProps {
  recordKey: string;
  /** Stands in, in gray, for an empty record key, such as "Any key". */
  emptyLabel: string;
}

/** Names a record of a topic in a task tree by its record key. */
export const RecordLabel = ({
  recordKey,
  emptyLabel,
}: RecordLabelProps): ReactElement => (
  <Flex.Box x align="center" className={CSS.B("kafka-record-label")}>
    <Task.Views.ItemLabel
      kind="JSON"
      kindColor="var(--pluto-secondary-z)"
      color={recordKey === "" ? 8 : 10}
      variant="code"
    >
      {recordKey === "" ? emptyLabel : recordKey}
    </Task.Views.ItemLabel>
  </Flex.Box>
);
