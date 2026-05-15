// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useEffect, useState } from "react";

import { Text } from "@/text";
import { useContext } from "@/triggers/Provider";
import { type Trigger } from "@/triggers/triggers";

export interface StatusProps extends Text.TextProps {}

export const Status = (props: StatusProps): ReactElement => {
  const { listen } = useContext();
  const [trigger, setTrigger] = useState<Trigger>([]);

  useEffect(
    () => listen(({ next: [trigger] }) => setTrigger(trigger ?? [])),
    [listen, setTrigger],
  );

  return <Text.Text {...props}>{trigger.join(" ")}</Text.Text>;
};
