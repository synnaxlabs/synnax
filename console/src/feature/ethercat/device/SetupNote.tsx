// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Note } from "@synnaxlabs/lyra/note";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, type ReactNode, useState } from "react";

import { useResultSlave, useSetStandalone } from "@/feature/ethercat/device/queries";
import { canSwitch, of, showsSwitch } from "@/feature/ethercat/device/setup";
import { type SlaveDevice } from "@/feature/ethercat/device/types";

interface NoteBodyProps {
  variant: "warning" | "info";
  title: string;
  children?: ReactNode;
}

const NoteBody = ({ variant, title, children }: NoteBodyProps): ReactElement => (
  <Note.Note variant={variant}>
    <Flex.Box y gap="small">
      <Text.Text weight={500} status={variant === "warning" ? "warning" : undefined}>
        {variant === "warning" ? <Icon.Warning /> : <Icon.Info />}
        {title}
      </Text.Text>
      {children}
    </Flex.Box>
  </Note.Note>
);

const Detail = ({ children }: { children: ReactNode }): ReactElement => (
  <Text.Text level="small" color={9}>
    {children}
  </Text.Text>
);

interface SwitchProps {
  slave: SlaveDevice;
  onSwitched: () => void;
}

/** Switches the slave to Synnax only, and shows why a switch failed. */
const Switch = ({ slave, onSwitched }: SwitchProps): ReactElement => {
  const {
    update,
    variant,
    status: result,
  } = useSetStandalone({
    afterSuccess: onSwitched,
  });
  return (
    <>
      <Flex.Box x>
        <Button.Button
          size="small"
          status={status.keepVariants(variant, "loading")}
          onClick={() => update({ slave })}
        >
          Switch to Synnax
        </Button.Button>
      </Flex.Box>
      {variant === "error" && (
        <Text.Text level="small" status="error">
          {result.description ?? result.message}
        </Text.Text>
      )}
    </>
  );
};

export interface SetupNoteProps {
  deviceKey: string;
}

/**
 * Says what keeps Synnax from using the slave, and offers the switch to Synnax only.
 * Renders nothing when Synnax alone sets up the slave.
 */
export const SetupNote = ({ deviceKey }: SetupNoteProps): ReactElement | null => {
  const { data: slave } = useResultSlave({ key: deviceKey });
  const [switched, setSwitched] = useState(false);
  if (slave == null) return null;
  const { properties } = slave;
  const setup = of(properties);
  const restarting = switched && (setup.kind !== "synnax" || !showsSwitch(properties));
  if (setup.kind === "switched" || restarting)
    return (
      <NoteBody variant="warning" title="Restart the power of the device">
        {properties.settings?.mode?.bus === 2 && (
          <Detail>First, connect Synnax to EtherCAT bus 1 IN.</Detail>
        )}
      </NoteBody>
    );
  if (setup.kind === "synnax") return null;
  if (setup.kind === "bus")
    return <NoteBody variant="warning" title="Connect Synnax to EtherCAT bus 1 IN" />;
  const shared = setup.kind === "shared";
  return (
    <NoteBody
      variant={shared ? "info" : "warning"}
      title={`Set up ${shared ? "by" : "for"} ${setup.reader}`}
    >
      <Detail>
        {shared
          ? "Synnax reads its data, but cannot change its settings."
          : "Synnax cannot read its data."}
      </Detail>
      {canSwitch(properties) && (
        <Switch slave={slave} onSwitched={() => setSwitched(true)} />
      )}
    </NoteBody>
  );
};
