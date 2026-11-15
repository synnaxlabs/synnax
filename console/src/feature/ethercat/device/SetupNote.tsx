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
  variant: "warning" | "info";
  title: string;
  /** Sits at the right of the text when it fits, else below it. */
  action?: ReactNode;
  children?: ReactNode;
}

const NoteBody = ({
  variant,
  title,
  action,
  children,
}: NoteBodyProps): ReactElement => (
  <Note.Note variant={variant} x align="center" justify="between" gap="medium" wrap>
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
    status: result,
  } = useSetStandalone({
    afterSuccess: ({ data: { answer } }) =>
      addStatus({
        variant: answer.variant,
        message: answer.message,
        description: answer.description,
      }),
  });
  const action = switchable ? (
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

/** Says which program sets up the slave, and offers the switch when it has one. */
const ReaderNote = ({ slave, reader, shared }: ReaderNoteProps): ReactElement => {
  const openDocs = Docs.useOpen(SETUP_DOCS);
  return (
    <SwitchNote
      slave={slave}
      switchable={canSwitch(slave.properties)}
      variant={shared ? "info" : "warning"}
      title={
        shared ? `${reader} sets up this device` : `Only ${reader} can use this device`
      }
    >
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
  <SwitchNote
    slave={slave}
    switchable={switchable}
    variant="warning"
    title={deviceStatus.message}
  >
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
      <NoteBody variant="warning" title="Connected to EtherCAT bus 2 IN">
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
