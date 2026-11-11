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
import { caseconv } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useState } from "react";

import { useResultSlave, useSetStandalone } from "@/feature/ethercat/device/queries";
import { canSwitch, of, showsSwitch } from "@/feature/ethercat/device/setup";
import { type SlaveDevice } from "@/feature/ethercat/device/types";

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

interface ChoiceProps {
  title: string;
  children: ReactNode;
}

const Choice = ({ title, children }: ChoiceProps): ReactElement => (
  <Flex.Box y gap="tiny">
    <Text.Text weight={500}>{title}</Text.Text>
    {children}
  </Flex.Box>
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
          Switch to Synnax only
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

const SWITCH_STEPS =
  "Synnax stops the tasks on the network of the device, then switches it. Restart " +
  "the power of the device to finish.";

const BUS_2_STEPS =
  "Synnax stops the tasks on the network of the device, then switches it. Connect " +
  "Synnax to EtherCAT bus 1 IN, then restart the power of the device to finish.";

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
  if (slave == null) return null;
  const { properties } = slave;
  const setup = of(properties);
  const bus = properties.settings?.mode?.bus;
  const restarting = switched && (setup.kind !== "synnax" || !showsSwitch(properties));
  if (setup.kind === "switched" || restarting)
    return (
      <NoteBody variant="warning" title="Restart the power of the device">
        <Detail>
          {bus === 2 && "Connect Synnax to EtherCAT bus 1 IN. "}
          The device runs Synnax only after a power restart.
        </Detail>
      </NoteBody>
    );
  if (setup.kind === "synnax") return null;
  if (setup.kind === "bus")
    return (
      <NoteBody variant="warning" title="Connect Synnax to EtherCAT bus 1 IN">
        <Detail>
          The device runs Synnax only, which uses EtherCAT bus 1 IN. Synnax is on bus 2
          IN.
        </Detail>
      </NoteBody>
    );
  const { reader } = setup;
  const switchChoice = canSwitch(properties) && (
    <Choice title="Synnax only">
      <Detail>{bus === 2 ? BUS_2_STEPS : SWITCH_STEPS}</Detail>
      <Switch slave={slave} onSwitched={() => setSwitched(true)} />
    </Choice>
  );
  if (setup.kind === "shared")
    return (
      <NoteBody variant="info" title={`Set up by ${reader}`}>
        <Detail>
          {caseconv.capitalize(reader)} sets up the device. Synnax uses its data, but
          does not change its settings. To set up the device in Synnax, switch it to
          Synnax only.
        </Detail>
        {bus === 1 && (
          <Detail>
            In dual mode, EtherCAT bus 1 IN is for {reader}. To run {reader} at the same
            time, connect {reader} to bus 1 IN and Synnax to bus 2 IN.
          </Detail>
        )}
        {switchChoice}
      </NoteBody>
    );
  const { secondBus } = setup;
  return (
    <NoteBody variant="warning" title={`Set up for ${reader}`}>
      <Detail>
        {secondBus ? "On EtherCAT bus 1 IN, the" : "The"} device sends data that only{" "}
        {reader} reads. Choose how to use the device.
      </Detail>
      {switchChoice}
      {secondBus && (
        <Choice title={`Synnax and ${reader}`}>
          <Detail>
            Connect Synnax to EtherCAT bus 2 IN, and {reader} to bus 1 IN. {reader} sets
            up the device, and Synnax reads its data.
          </Detail>
        </Choice>
      )}
    </NoteBody>
  );
};
