// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/device/Connect.css";

import { type device, type rack } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Pane } from "@synnaxlabs/lyra/pane";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Rack } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { ConfigureHardware } from "@/feature/device/Configure";
import { HARDWARE, type Hardware, NETWORK } from "@/feature/device/integrations";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Modals } from "@/platform/modals";
import { type Session } from "@/session";

export interface ConnectModalParams {
  /** The integration to open at, instead of the picker. */
  integration?: string;
  /**
   * The rack to start from. A connect form selects it, and the find view lists only
   * it.
   */
  rack?: rack.Key;
}

interface EntryProps {
  integration: PlatformDevice.Integration;
  reason: string | null;
  onSelect: (key: string) => void;
}

const Entry = ({ integration, reason, onSelect }: EntryProps): ReactElement => (
  <Pane.Button onClick={() => onSelect(integration.key)}>
    <Icon.Tile>{integration.icon}</Icon.Tile>
    <Flex.Box y gap="tiny" grow align="start">
      <Text.Text level="p" weight={500} color={11}>
        {integration.title}
      </Text.Text>
      <Text.Text level="small" color={9}>
        {integration.description}
      </Text.Text>
      {reason != null && (
        <Text.Text
          level="small"
          color={9}
          className={CSS.BE("device-connect", "reason")}
        >
          <Status.Indicator variant="warning" />
          {reason}
        </Text.Text>
      )}
    </Flex.Box>
  </Pane.Button>
);

interface GroupProps {
  title: string;
  integrations: PlatformDevice.Integration[];
  getReason: (integration: PlatformDevice.Integration) => string | null;
  onSelect: (key: string) => void;
}

const Group = ({
  title,
  integrations,
  getReason,
  onSelect,
}: GroupProps): ReactElement => (
  <Flex.Box y gap="small">
    <Text.Text
      level="small"
      weight={500}
      color={9}
      className={CSS.BE("device-connect", "title")}
    >
      {title}
    </Text.Text>
    <Pane.Pane>
      {integrations.map((i) => (
        <Entry key={i.key} integration={i} reason={getReason(i)} onSelect={onSelect} />
      ))}
    </Pane.Pane>
  </Flex.Box>
);

interface PickerProps {
  racks: rack.Rack[] | undefined;
  rack: rack.Key | undefined;
  onSelect: (key: string) => void;
}

const Picker = ({ racks, rack: rackKey, onSelect }: PickerProps): ReactElement => {
  const getReason = ({ key, name }: PlatformDevice.Integration): string | null =>
    racks == null
      ? null
      : PlatformDevice.getBlockedReason(racks, {
          integration: key,
          name,
          rack: rackKey,
        });
  return (
    <Modal.Frame className={CSS.B("device-connect")}>
      <Modal.Header icon={<Icon.Device />}>Device.Connect</Modal.Header>
      <Modal.Body gap="huge" justify="start">
        <Group
          title="Connect by address"
          integrations={NETWORK}
          getReason={getReason}
          onSelect={onSelect}
        />
        <Group
          title="Find hardware the Driver detects"
          integrations={HARDWARE}
          getReason={getReason}
          onSelect={onSelect}
        />
      </Modal.Body>
    </Modal.Frame>
  );
};

interface FindStepProps {
  hardware: Hardware;
  rack: rack.Key | undefined;
  close: () => void;
  onBack: () => void;
}

const FindStep = ({ hardware, rack, close, onBack }: FindStepProps): ReactElement => {
  const [configuring, setConfiguring] = useState<device.Key | null>(null);
  if (configuring == null)
    return (
      <PlatformDevice.Find
        hardware={hardware}
        rack={rack}
        onBack={onBack}
        onConfigure={setConfiguring}
      />
    );
  return (
    <ConfigureHardware
      integration={hardware.key}
      deviceKey={configuring}
      close={() => setConfiguring(null)}
      onBack={() => setConfiguring(null)}
      onTaskCreated={close}
    />
  );
};

const Connect = ({
  integration,
  rack: rackKey,
  close,
}: Session.Modals.ContentProps<ConnectModalParams, void>): ReactElement => {
  const { data: racks, variant } = Rack.useResultMultiple({});
  const [selected, setSelected] = useState<string | null>(integration ?? null);
  const network = NETWORK.find(({ key }) => key === selected);
  // A network form reads its default rack once, so it waits while the rack list loads.
  if (selected == null || (network != null && variant === "loading"))
    return <Picker racks={racks} rack={rackKey} onSelect={setSelected} />;
  const back = () => setSelected(null);
  if (network != null)
    return (
      <network.Connect
        rack={rackKey ?? PlatformDevice.getDefaultRack(racks ?? [], network.key)}
        close={() => close()}
        onBack={back}
      />
    );
  const hardware = HARDWARE.find(({ key }) => key === selected);
  if (hardware == null) throw new Error(`No integration named ${selected}`);
  return <FindStep hardware={hardware} rack={rackKey} close={close} onBack={back} />;
};

/** Opens a picker of every integration, each leading to its connect or find view. */
export const useConnectModal = Modals.create<ConnectModalParams>(Connect);
