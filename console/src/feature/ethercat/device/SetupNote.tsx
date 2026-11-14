// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, type task } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Note } from "@synnaxlabs/lyra/note";
import { Text } from "@synnaxlabs/lyra/text";
import { deep } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useState } from "react";

import { useResultSlave, useSetStandalone } from "@/feature/ethercat/device/queries";
import { canSwitch, of } from "@/feature/ethercat/device/setup";
import {
  type SlaveDevice,
  type SlaveProperties,
} from "@/feature/ethercat/device/types";
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

interface ReaderNoteProps {
  slave: SlaveDevice;
  reader: string;
  shared: boolean;
  onSwitched: (answer: task.Status) => void;
}

/**
 * Says which program sets up the slave, offers the switch to Synnax only when the slave
 * has one, and shows why a switch failed.
 */
const ReaderNote = ({
  slave,
  reader,
  shared,
  onSwitched,
}: ReaderNoteProps): ReactElement => {
  const {
    update,
    variant,
    status: result,
  } = useSetStandalone({ afterSuccess: ({ data }) => onSwitched(data.answer) });
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

/** @returns the note for the slave, or null when Synnax alone sets it up. */
const noteOf = (
  slave: SlaveDevice,
  onSwitched: (answer: task.Status) => void,
): ReactElement | null => {
  const { properties } = slave;
  const setup = of(properties);
  if (setup.kind === "switched")
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
  return (
    <ReaderNote
      slave={slave}
      reader={setup.reader}
      shared={setup.kind === "shared"}
      onSwitched={onSwitched}
    />
  );
};

interface Answer {
  status: task.Status;
  /** The properties of the slave when the Driver answered. */
  properties: SlaveProperties;
}

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
  const [answer, setAnswer] = useState<Answer | null>(null);
  if (slave == null) return null;
  // The answer of the Driver holds until a scan changes the slave.
  const note =
    answer != null && deep.equal(answer.properties, slave.properties) ? (
      <NoteBody
        variant={answer.status.variant === "success" ? "info" : "warning"}
        title={answer.status.message}
      >
        {answer.status.description != null && (
          <Detail>{answer.status.description}</Detail>
        )}
      </NoteBody>
    ) : (
      noteOf(slave, (status) => setAnswer({ status, properties: slave.properties }))
    );
  if (note == null || !aligned) return note;
  return <Form.Aligned>{note}</Form.Aligned>;
};
