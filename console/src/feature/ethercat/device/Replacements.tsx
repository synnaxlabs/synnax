// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Status } from "@synnaxlabs/lyra/status";
import { useEffect, useRef } from "react";
import { z } from "zod";

import { useEverySlave, useReplace } from "@/feature/ethercat/device/queries";
import { canReplace, findReplacements } from "@/feature/ethercat/device/replace";
import { type Notifications } from "@/platform/notifications";

const detailsZ = z.object({
  replace: z.object({ slave: z.string(), old: z.array(z.string()) }),
});

/**
 * Asks once in a session, for each new slave that can take the place of old slaves,
 * whether it is one of them.
 */
export const useListenForReplacements = (): void => {
  const slaves = useEverySlave();
  const addStatus = Status.useAdder();
  const asked = useRef(new Set<device.Key>());
  useEffect(() => {
    if (slaves == null) return;
    for (const { slave, old } of findReplacements(slaves)) {
      if (asked.current.has(slave.key)) continue;
      asked.current.add(slave.key);
      addStatus<typeof detailsZ>({
        variant: "info",
        name: slave.name,
        message:
          old.length === 1
            ? `Is this your old ${old[0].name}?`
            : "Is this one of your old devices?",
        description:
          "Yes moves the module into the old device, so its tasks and settings keep working. Synnax first stops the EtherCAT tasks on the network.",
        details: { replace: { slave: slave.key, old: old.map((o) => o.key) } },
      });
    }
  }, [slaves, addStatus]);
};

const Notification: Notifications.Notification = ({ status: st, silence }) => {
  const { replace } = detailsZ.parse(st.details);
  const slaves = useEverySlave();
  const addStatus = Status.useAdder();
  const { update, variant } = useReplace({
    afterSuccess: ({ data: { old, stopped } }) => {
      silence(st.key);
      addStatus({
        variant: "success",
        name: old.name,
        message: "Moved the module into the device",
        description: stopped.length === 0 ? undefined : "Start your tasks again.",
      });
    },
  });
  const slave = slaves?.find((s) => s.key === replace.slave);
  // The user may configure the new slave, or plug the old module back in, first.
  const old =
    slave == null
      ? []
      : (slaves ?? []).filter(
          (s) => replace.old.includes(s.key) && canReplace(s, slave),
        );
  const actions =
    slave == null || old.length === 0 ? undefined : (
      <>
        {old
          .sort((a, b) => a.properties.position - b.properties.position)
          .map((o) => (
            <Button.Button
              key={o.key}
              variant="outlined"
              size="tiny"
              status={status.keepVariants(variant, "loading")}
              onClick={() => update({ old: o.key, slave: slave.key })}
            >
              {replace.old.length === 1
                ? "Yes"
                : `${o.name} at position ${o.properties.position}`}
            </Button.Button>
          ))}
        <Button.Button variant="text" size="tiny" onClick={() => silence(st.key)}>
          No
        </Button.Button>
      </>
    );
  return <Status.Notification status={st} silence={silence} actions={actions} />;
};
Notification.match = (st) => detailsZ.safeParse(st.details).success;

export const NOTIFICATIONS: Notifications.Notification[] = [Notification];
