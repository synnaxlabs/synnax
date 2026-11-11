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
import { canSwitch, confirmsSwitch, of } from "@/feature/ethercat/device/setup";
import { type SlaveDevice } from "@/feature/ethercat/device/types";
import { Docs } from "@/platform/docs";

const SETUP_DOCS = "hardware/ethercat/dewesoft#set-up-a-module-in-synnax";

interface NoteBodyProps {
  variant: "warning" | "info";
  className?: string;
  title: string;
  /** Sits at the right of the text when it fits, else below it. */
  action?: ReactNode;
  children?: ReactNode;
}

const NoteBody = ({
  variant,
  className,
  title,
  action,
  children,
}: NoteBodyProps): ReactElement => (
  <Note.Note
    variant={variant}
    className={className}
    x
    align="center"
    justify="between"
    gap="medium"
    wrap
  >
    <Flex.Box y gap="small">
      <Text.Text weight={500} status={variant === "warning" ? "warning" : undefined}>
        {variant === "warning" ? <Icon.Warning /> : <Icon.Info />}
        {title}
      </Text.Text>
      {children}
    </Flex.Box>
    {action}
  </Note.Note>
);

const Detail = ({ children }: { children: ReactNode }): ReactElement => (
  <Text.Text level="small" color={9}>
    {children}
  </Text.Text>
);

interface ReaderNoteProps {
  className?: string;
  slave: SlaveDevice;
  reader: string;
  shared: boolean;
  onSwitched: () => void;
}

/**
 * Says which program sets up the slave, offers the switch to Synnax only when the slave
 * has one, and shows why a switch failed.
 */
const ReaderNote = ({
  className,
  slave,
  reader,
  shared,
  onSwitched,
}: ReaderNoteProps): ReactElement => {
  const {
    update,
    variant,
    status: result,
  } = useSetStandalone({
    afterSuccess: onSwitched,
  });
  const openDocs = Docs.useOpen(SETUP_DOCS);
  const action = canSwitch(slave.properties) ? (
    <Button.Button
      size="small"
      status={status.keepVariants(variant, "loading")}
      onClick={() => update({ slave })}
    >
      Switch to Synnax
    </Button.Button>
  ) : undefined;
  return (
    <NoteBody
      variant={shared ? "info" : "warning"}
      className={className}
      title={
        shared ? `${reader} sets up this device` : `Only ${reader} can use this device`
      }
      action={action}
    >
      <Button.Button variant="text" size="small" onClick={openDocs}>
        What does this mean?
        <Icon.OpenExternal />
      </Button.Button>
      {variant === "error" && (
        <Text.Text level="small" status="error">
          {result.description ?? result.message}
        </Text.Text>
      )}
    </NoteBody>
  );
};

export interface SetupNoteProps {
  deviceKey: string;
  className?: string;
}

/**
 * Says which software sets up the slave and what the user must do to use it with
 * Synnax. Renders nothing when Synnax alone sets up the slave.
 */
export const SetupNote = ({
  deviceKey,
  className,
}: SetupNoteProps): ReactElement | null => {
  const { data: slave } = useResultSlave({ key: deviceKey });
  const [switched, setSwitched] = useState(false);
  if (slave == null) return null;
  const { properties } = slave;
  const setup = of(properties);
  const restarting =
    switched && (setup.kind !== "synnax" || !confirmsSwitch(properties));
  if (setup.kind === "switched" || restarting)
    return (
      <NoteBody
        variant="warning"
        className={className}
        title="Restart the power of the device"
      >
        {properties.settings?.mode?.bus === 2 && (
          <Detail>First, connect Synnax to EtherCAT bus 1 IN.</Detail>
        )}
      </NoteBody>
    );
  if (setup.kind === "synnax") return null;
  if (setup.kind === "bus")
    return (
      <NoteBody
        variant="warning"
        className={className}
        title="Connect Synnax to EtherCAT bus 1 IN"
      />
    );
  return (
    <ReaderNote
      className={className}
      slave={slave}
      reader={setup.reader}
      shared={setup.kind === "shared"}
      onSwitched={() => setSwitched(true)}
    />
  );
};
