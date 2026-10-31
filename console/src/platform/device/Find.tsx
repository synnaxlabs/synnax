// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/device/Find.css";

import {
  type device,
  DisconnectedError,
  type rack,
  task,
  TimeSpan,
} from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Pane } from "@synnaxlabs/lyra/pane";
import { Status } from "@synnaxlabs/lyra/status";
import { Steps } from "@synnaxlabs/lyra/steps";
import { Text } from "@synnaxlabs/lyra/text";
import { Device as PDevice, Rack, Synnax, Task } from "@synnaxlabs/pluto";
import { type ReactElement, type ReactNode, useState } from "react";

import { CSS } from "@/platform/css";
import { Configure } from "@/platform/device/Configure";
import { type Hardware } from "@/platform/device/integration";
import { isRunning } from "@/platform/device/racks";
import { Modals } from "@/platform/modals";
import { type Session } from "@/session";

const SCAN_COMMAND = "scan";
const SCAN_TIMEOUT = TimeSpan.seconds(15);

const scanDisabledZ = task.scanConfigZ.pick({ disabled: true });

const isScanDisabled = (scanner: task.Task): boolean =>
  scanDisabledZ.safeParse(scanner.config).data?.disabled === true;

const DRIVER_CHECK = "The Synnax Driver runs on the computer the hardware connects to.";

const QUIET = new Set<Status.Variant>(["loading", "disabled"]);

interface StateProps {
  variant: Status.Variant;
  message: ReactNode;
  description?: string;
  action?: ReactNode;
}

const State = ({ variant, message, description, action }: StateProps): ReactElement => (
  <Pane.Item>
    <Flex.Box y gap="tiny" grow>
      <Text.Text level="small" color={QUIET.has(variant) ? 9 : 10}>
        <Status.Indicator variant={variant} />
        {message}
      </Text.Text>
      {description != null && description.length > 0 && (
        <Text.Text level="small" color={9}>
          {description}
        </Text.Text>
      )}
    </Flex.Box>
    {action}
  </Pane.Item>
);

interface ScannerStateProps {
  rack: rack.Rack;
  scanner: task.Task | undefined;
  name: string;
}

const ScannerState = ({
  rack: r,
  scanner,
  name,
}: ScannerStateProps): ReactElement | null => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  if (!isRunning(r))
    return (
      <State
        variant="warning"
        message={r.status?.message ?? "Driver not running"}
        description={r.status?.description}
      />
    );
  if (scanner == null)
    return (
      <State
        variant="warning"
        message={`This Driver does not scan for ${name} hardware`}
      />
    );
  if (isScanDisabled(scanner))
    return (
      <State
        variant="disabled"
        message="Scanning is off on this Driver"
        action={
          <Button.Button
            size="small"
            variant="outlined"
            onClick={() =>
              handleError(async () => {
                if (client == null) throw new DisconnectedError();
                // Never write the status back: the Driver owns it.
                await client.tasks.create({
                  ...scanner.payload,
                  config: { ...scanner.config, disabled: false },
                  status: undefined,
                });
              }, "Failed to turn on scanning")
            }
          >
            Turn on
          </Button.Button>
        }
      />
    );
  const scanStatus = scanner.status;
  if (scanStatus?.variant === "error" || scanStatus?.variant === "warning")
    return (
      <State
        variant={scanStatus.variant}
        message={scanStatus.message}
        description={scanStatus.description}
      />
    );
  return null;
};

const isScanning = (r: rack.Rack, scanner: task.Task | undefined): boolean =>
  isRunning(r) &&
  scanner != null &&
  !isScanDisabled(scanner) &&
  scanner.status?.variant !== "error";

interface DeviceItemProps {
  device: device.Device;
  onConfigure: (key: device.Key) => void;
}

const DeviceItem = ({ device: d, onConfigure }: DeviceItemProps): ReactElement => (
  <Pane.Item>
    <PDevice.StatusIndicator status={d.status} />
    <Flex.Box y gap="tiny" grow>
      <Text.Text level="p" weight={500} color={11}>
        {d.name}
      </Text.Text>
      <Text.Text level="small" color={9}>
        {d.model}
        {d.location.length > 0 && (
          <span className={CSS.BE("device-find", "location")}>{d.location}</span>
        )}
      </Text.Text>
    </Flex.Box>
    {d.configured ? (
      <Text.Text
        level="small"
        color={9}
        className={CSS.BE("device-find", "configured")}
      >
        <Icon.Check />
        Configured
      </Text.Text>
    ) : (
      <Button.Button size="small" variant="filled" onClick={() => onConfigure(d.key)}>
        Configure
      </Button.Button>
    )}
  </Pane.Item>
);

interface RackCardProps {
  rack: rack.Rack;
  scanner: task.Task | undefined;
  devices: device.Device[];
  name: string;
  onConfigure: (key: device.Key) => void;
}

const RackCard = ({
  rack: r,
  scanner,
  devices,
  name,
  onConfigure,
}: RackCardProps): ReactElement => (
  <Pane.Pane>
    <Pane.Header level="p">
      <Header.Title weight={500} color={11}>
        <Icon.Rack />
        {r.name}
      </Header.Title>
      <Header.Actions>
        {devices.length > 0 && (
          <Text.Text level="small" color={9}>
            {devices.length === 1 ? "1 device" : `${devices.length} devices`}
          </Text.Text>
        )}
        <Rack.StatusIndicator status={r.status} />
      </Header.Actions>
    </Pane.Header>
    <ScannerState rack={r} scanner={scanner} name={name} />
    {devices.map((d) => (
      <DeviceItem key={d.key} device={d} onConfigure={onConfigure} />
    ))}
    {devices.length === 0 && isScanning(r, scanner) && (
      <State variant="loading" message={`Looking for ${name} hardware`} />
    )}
  </Pane.Pane>
);

interface ChecksProps {
  checks: string[];
  open: boolean;
}

const Checks = ({ checks, open }: ChecksProps): ReactElement => (
  <Pane.Disclosure open={open}>
    <Pane.Summary>
      <Icon.QuestionMark />
      Not seeing your hardware?
    </Pane.Summary>
    <Pane.Item>
      <Steps.List level="small" className={CSS.BE("device-find", "steps")}>
        {[DRIVER_CHECK, ...checks].map((check) => (
          <Steps.Item key={check}>{check}</Steps.Item>
        ))}
      </Steps.List>
    </Pane.Item>
  </Pane.Disclosure>
);

export interface FindProps {
  hardware: Hardware;
  /** Lists only this rack. */
  rack?: rack.Key;
  /** Lists only the devices it accepts. */
  filter?: (device: device.Device) => boolean;
  /** Returns to the view that opened this one. Adds a "Back" action. */
  onBack?: () => void;
  /** Runs when the user picks a found device to configure. */
  onConfigure: (deviceKey: device.Key) => void;
}

/**
 * Lists what each Driver with the integration found, with the state of its scanner, an
 * action to scan now, and what to check when the hardware is missing.
 */
export const Find = ({
  hardware: { key, name, title, icon, make, scanType, checks },
  rack: rackKey,
  filter,
  onBack,
  onConfigure,
}: FindProps): ReactElement => {
  const allRacks = Rack.useResultMultiple({}).data;
  const scope = allRacks?.find((r) => r.key === rackKey);
  const running = allRacks?.some(isRunning) === true;
  const racks = allRacks?.filter(
    (r) => r.integrations.includes(key) && (rackKey == null || r.key === rackKey),
  );
  const scanners = Task.useResultMultiple({ types: [scanType] }).data;
  const found = (PDevice.useResultMultiple({ makes: [make] }).data ?? [])
    .filter(
      (d) => racks?.some((r) => r.key === d.rack) === true && (filter?.(d) ?? true),
    )
    .toSorted((a, b) => Number(a.configured) - Number(b.configured));
  const handleError = Status.useErrorHandler();
  const [scanning, setScanning] = useState(false);
  const targets = (scanners ?? []).filter(
    (s) =>
      !isScanDisabled(s) &&
      racks?.some((r) => r.key === s.rack && isRunning(r)) === true,
  );
  const scan = async () => {
    setScanning(true);
    try {
      await Promise.all(
        targets.map((s) =>
          s.executeCommandSync({ type: SCAN_COMMAND, timeout: SCAN_TIMEOUT }),
        ),
      );
    } catch (e) {
      handleError(e, `Failed to scan for ${name} hardware`);
    }
    setScanning(false);
  };
  return (
    <Modal.Frame className={CSS.B("device-find")}>
      <Modal.Header icon={icon}>{["Device", "Connect", title]}</Modal.Header>
      <Modal.Body gap="large" justify="start">
        {racks?.length === 0 && (
          <Pane.Pane>
            <Flex.Box
              y
              align="center"
              gap="small"
              className={CSS.BE("device-find", "empty")}
            >
              <Icon.Tile>{icon}</Icon.Tile>
              <Text.Text level="p" weight={500} color={11}>
                {scope != null
                  ? `${scope.name} does not have ${name} enabled`
                  : running
                    ? `Your Driver is running, but ${name} is off`
                    : "No Driver is running"}
              </Text.Text>
              <Text.Text level="small" color={9}>
                {scope != null || running
                  ? `${name} is off when the Driver configuration disables it, or when the Driver's computer cannot run it. See the checks below.`
                  : "Start the Synnax Driver on the computer the hardware connects to."}
              </Text.Text>
            </Flex.Box>
          </Pane.Pane>
        )}
        {racks?.map((r) => (
          <RackCard
            key={r.key}
            rack={r}
            scanner={scanners?.find((s) => s.rack === r.key)}
            devices={found.filter((d) => d.rack === r.key)}
            name={name}
            onConfigure={onConfigure}
          />
        ))}
        <Checks checks={checks} open={found.length === 0} />
      </Modal.Body>
      <Modal.Footer>
        {onBack != null && (
          <Nav.Bar.Start>
            <Button.Button variant="text" onClick={onBack}>
              <Icon.Arrow.Left />
              Back
            </Button.Button>
          </Nav.Bar.Start>
        )}
        <Nav.Bar.End>
          <Button.Button
            variant="outlined"
            onClick={() => void scan()}
            status={scanning ? "loading" : undefined}
            disabled={targets.length === 0}
          >
            <Icon.Refresh />
            Scan now
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

export interface FindModalParams extends Pick<FindProps, "hardware" | "filter"> {}

const FindModal = ({
  hardware,
  filter,
  close,
}: Session.Modals.ContentProps<FindModalParams, device.Key>): ReactElement => {
  const [configuring, setConfiguring] = useState<device.Key | null>(null);
  if (configuring == null)
    return <Find hardware={hardware} filter={filter} onConfigure={setConfiguring} />;
  return (
    <Configure
      deviceKey={configuring}
      icon={hardware.icon}
      initialProperties={hardware.zeroProperties}
      onBack={() => setConfiguring(null)}
      close={() => close(configuring)}
    />
  );
};

/**
 * Opens {@link Find} in a modal. Resolves with the device the user configures, or null
 * when the user closes the modal first.
 */
export const useFindModal = Modals.createPrompt<device.Key, FindModalParams>(FindModal);
