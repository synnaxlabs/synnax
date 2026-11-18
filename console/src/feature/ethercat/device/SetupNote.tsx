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
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Note } from "@synnaxlabs/lyra/note";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, type ReactNode } from "react";

import { useResultSlave, useSetStandalone } from "@/feature/ethercat/device/queries";
import { canSwitch, of } from "@/feature/ethercat/device/setup";
import { type SlaveDevice } from "@/feature/ethercat/device/types";
import { Docs } from "@/platform/docs";

const SETUP_DOCS = "hardware/ethercat/dewesoft#set-up-a-module-in-synnax";

interface NoteBodyProps {
  title: string;
  /** Sits at the right of the text when it fits, else below it. */
  action?: ReactNode;
  children?: ReactNode;
}

const NoteBody = ({ title, action, children }: NoteBodyProps): ReactElement => (
  <Note.Note variant="warning" x align="center" justify="between" gap="medium" wrap>
    <Flex.Box y gap="small">
      <Text.Text weight={500} status="warning">
        <Icon.Warning />
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

/** The Driver writes a switch to the slave, so it must reach the slave. */
const canReach = (slave: SlaveDevice): boolean =>
  slave.status?.details.connected === true;

interface SwitchNoteProps extends Omit<NoteBodyProps, "action"> {
  slave: SlaveDevice;
  /** Whether the note offers the switch to Synnax only. */
  switchable: boolean;
}

/**
 * A note that offers the switch of the slave to Synnax only when switchable. Adds the
 * answer of the Driver to the statuses, and shows why a switch failed.
 */
const SwitchNote = ({
  slave,
  switchable,
  children,
  ...rest
}: SwitchNoteProps): ReactElement => {
  const addStatus = Status.useAdder();
  const {
    update,
    variant,
    data,
    status: result,
  } = useSetStandalone({
    afterSuccess: ({ data: { answer } }) =>
      addStatus({
        variant: answer.variant,
        message: answer.message,
        description: answer.description,
      }),
  });
  // The note shows the setup of the last scan until the next one.
  const switched = variant === "success" && data != null;
  const action =
    switchable && canReach(slave) && !switched ? (
      <Button.Button
        size="small"
        status={status.keepVariants(variant, "loading")}
        onClick={() => update({ slave })}
      >
        Switch to Synnax
      </Button.Button>
    ) : undefined;
  return (
    <NoteBody {...rest} action={action}>
      {children}
      {variant === "error" && (
        <Text.Text level="small" status="error">
          {result.description ?? result.message}
        </Text.Text>
      )}
    </NoteBody>
  );
};

interface ReaderNoteProps {
  slave: SlaveDevice;
  reader: string;
  shared: boolean;
}

/** @returns what the user must do before a task uses the slave that reader sets up. */
const readerStep = (slave: SlaveDevice, reader: string): string => {
  if (!canSwitch(slave.properties))
    return `Set it up for Synnax in ${reader} before you use it in a task.`;
  if (!canReach(slave))
    return "Connect the device, then switch it to Synnax before you use it in a task.";
  return "Switch it to Synnax before you use it in a task.";
};

/** Says which program sets up the slave and what to do, with a switch if it has one. */
const ReaderNote = ({ slave, reader, shared }: ReaderNoteProps): ReactElement => {
  const openDocs = Docs.useOpen(SETUP_DOCS);
  return (
    <SwitchNote
      slave={slave}
      switchable={canSwitch(slave.properties)}
      title={
        shared ? `${reader} sets up this device` : `Only ${reader} can use this device`
      }
    >
      <Detail>{readerStep(slave, reader)}</Detail>
      <Button.Button variant="text" size="small" onClick={openDocs}>
        What does this mean?
        <Icon.OpenExternal />
      </Button.Button>
    </SwitchNote>
  );
};

interface DeviceStatusNoteProps {
  slave: SlaveDevice;
  deviceStatus: device.Status;
  switchable: boolean;
}

/** Shows the status of the slave from the Driver. */
const DeviceStatusNote = ({
  slave,
  deviceStatus,
  switchable,
}: DeviceStatusNoteProps): ReactElement => (
  <SwitchNote slave={slave} switchable={switchable} title={deviceStatus.message}>
    {deviceStatus.description !== "" && <Detail>{deviceStatus.description}</Detail>}
  </SwitchNote>
);

/** @returns the note for the slave, or null when Synnax alone sets it up. */
const noteOf = (slave: SlaveDevice): ReactElement | null => {
  const { properties, status: deviceStatus } = slave;
  const setup = of(properties);
  // The Driver tracks a switch until it takes effect, so its status of the device says
  // what to do next.
  if (
    (properties.switchedAt != null || setup.kind === "switched") &&
    deviceStatus?.variant === "warning"
  )
    return (
      <DeviceStatusNote
        slave={slave}
        deviceStatus={deviceStatus}
        switchable={canSwitch(properties) && properties.switchedAt != null}
      />
    );
  if (setup.kind === "synnax" || setup.kind === "switched") return null;
  if (setup.kind === "bus")
    return (
      <NoteBody title="Connected to EtherCAT bus 2 IN">
        <Detail>
          In Synnax only mode, the device uses EtherCAT bus 1 IN. Connect Synnax to bus
          1 IN.
        </Detail>
      </NoteBody>
    );
  return (
    <ReaderNote slave={slave} reader={setup.reader} shared={setup.kind === "shared"} />
  );
};

/**
 * @returns whether the user must act on the setup of the slave before a task uses it,
 * which is when {@link SetupNote} shows a note.
 */
export const needsSetup = (slave: SlaveDevice): boolean => noteOf(slave) != null;

export interface SetupNoteProps {
  deviceKey: string;
  /** Lines the note up with the controls of the form sections it sits in. */
  aligned?: boolean;
}

/**
 * Says which software sets up the slave and what the user must do to use it with
 * Synnax. Renders nothing when Synnax alone sets up the slave.
 */
export const SetupNote = ({
  deviceKey,
  aligned = false,
}: SetupNoteProps): ReactElement | null => {
  const { data: slave } = useResultSlave({ key: deviceKey });
  if (slave == null) return null;
  const note = noteOf(slave);
  if (note == null || !aligned) return note;
  return <Form.Aligned>{note}</Form.Aligned>;
};
