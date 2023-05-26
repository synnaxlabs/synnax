// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";

import { Status, StatusTextProps } from "@/core/std/Status";
import { Trigger } from "@/core/triggers/triggers";
import { useTriggerContext } from "@/core/triggers/TriggersContext";

export interface TriggerStatusProps extends StatusTextProps {}

export const TriggerStatus = (props: TriggerStatusProps): ReactElement => {
  const { listen } = useTriggerContext();
  const [trigger, setTrigger] = useState<Trigger>([]);

  useEffect(
    () =>
      listen(({ next: [trigger] }) => {
        setTrigger(trigger ?? []);
      }),
    [listen, setTrigger]
  );

  return (
    <Status.Text hideIcon {...props}>
      {trigger.join(" ")}
    </Status.Text>
  );
};
