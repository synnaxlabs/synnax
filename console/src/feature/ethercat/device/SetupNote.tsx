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
import { setupOf } from "@/feature/ethercat/device/setup";

interface NoteBodyProps {
  variant: "warning" | "info";
  title: string;
  children: ReactNode;
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

export interface SetupNoteProps {
  deviceKey: string;
}

/**
 * Says how other software sets up the slave and what the user can do about it. Renders
 * nothing when Synnax alone sets up the slave.
 */
export const SetupNote = ({ deviceKey }: SetupNoteProps): ReactElement | null => {
  const { data: slave } = useResultSlave({ key: deviceKey });
  const [switched, setSwitched] = useState(false);
  const {
    update,
    variant,
    status: result,
  } = useSetStandalone({ afterSuccess: () => setSwitched(true) });
  if (switched)
    return (
      <NoteBody variant="warning" title="Restart the power of the device">
        <Detail>
          The device runs Synnax only after a power restart. Until then, it refuses to
          start.
        </Detail>
      </NoteBody>
    );
  if (slave == null) return null;
  const setup = setupOf(slave.properties);
  if (setup.kind === "synnax") return null;
  const { reader } = setup;
  if (setup.kind === "shared")
    return (
      <NoteBody variant="info" title={`Set up by ${reader}`}>
        <Detail>
          {reader} sets up this device, so Synnax does not change its settings.
        </Detail>
      </NoteBody>
    );
  return (
    <NoteBody variant="warning" title={`Set up for ${reader}`}>
      <Detail>
        On this port, the device sends data that only {reader} reads. Choose how to use
        the device.
      </Detail>
      <Flex.Box y gap="tiny">
        <Text.Text weight={500}>Synnax only</Text.Text>
        <Detail>
          Synnax stops the tasks on the network of the device, then switches it. Restart
          the power of the device to finish.
        </Detail>
        <Flex.Box x>
          <Button.Button
            size="small"
            status={status.keepVariants(variant, "loading")}
            onClick={() => update({ slave })}
          >
            Switch to Synnax only
          </Button.Button>
        </Flex.Box>
        {variant === "error" && (
          <Text.Text level="small" status="error">
            {result.description ?? result.message}
          </Text.Text>
        )}
      </Flex.Box>
      <Flex.Box y gap="tiny">
        <Text.Text weight={500}>Synnax and {reader}</Text.Text>
        <Detail>
          Connect Synnax to the other EtherCAT port of the device. {reader} sets up the
          device, and Synnax reads its data.
        </Detail>
      </Flex.Box>
    </NoteBody>
  );
};
