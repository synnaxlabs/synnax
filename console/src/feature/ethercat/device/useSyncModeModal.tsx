// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/ethercat/device/SyncMode.css";

import { type device, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { useResultSlave, useUpdateOpMode } from "@/feature/ethercat/device/queries";
import {
  autoIndex,
  choose,
  describeMode,
  describeTicks,
  isSM,
  isStale,
  modeName,
  sync0Error,
  sync1Error,
  takesCycles,
} from "@/feature/ethercat/device/syncMode";
import {
  type OpMode,
  type OpModeChoice,
  type SlaveDevice,
} from "@/feature/ethercat/device/types";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const AUTO = -1;
const STALE = -2;

const BADGE_CLASS = CSS.BE("ethercat-sync", "badge");

const APPLY_NOTE =
  "A change applies the next time the network starts, after every task on it stops.";

const STALE_ERROR =
  "The device no longer lists this mode, so the network fails to start. Pick a mode.";

/** Says which mode auto runs on the device, and why. */
const autoDescription = (modes: OpMode[]): string => {
  const auto = autoIndex(modes);
  if (auto == null)
    return "The Driver uses SM sync, because the device lists no modes.";
  if (isSM(modes[auto]))
    return "The Driver uses SM sync, because the device lists it. Best for most needs.";
  return (
    `The Driver uses ${modeName(modes[auto], auto)}, the first mode, because the ` +
    "device lists no SM mode. Best for most needs."
  );
};

const modeIcon = (mode: OpMode): Icon.ReactElement =>
  isSM(mode) ? <Icon.Sync /> : <Icon.Time />;

const staleLabel = ({ index, assignActivate }: OpModeChoice): string =>
  `Mode ${index + 1} (0x${assignActivate.toString(16).padStart(4, "0").toUpperCase()})`;

interface ModeSelectProps {
  modes: OpMode[];
  choice: OpModeChoice | null;
  stale: boolean;
  onChange: (choice: OpModeChoice | null) => void;
}

const ModeSelect = ({
  modes,
  choice,
  stale,
  onChange,
}: ModeSelectProps): ReactElement => {
  const auto = autoIndex(modes);
  const autoName = auto == null ? "SM" : modeName(modes[auto], auto);
  let value = AUTO;
  if (stale) value = STALE;
  else if (choice != null) value = choice.index;
  let icon: Icon.ReactElement;
  if (value === AUTO) icon = <Icon.Auto />;
  else if (value === STALE) icon = <Icon.Warning />;
  else icon = modeIcon(modes[value]);
  return (
    <Select.Simple<number>
      value={value}
      onChange={(k: number | null) => {
        if (k == null || k === STALE) return;
        onChange(k === AUTO ? null : choose(modes, k));
      }}
      resourceName="mode"
      searchHidden
      allowNone={false}
      icon={icon}
      variant="floating"
      triggerProps={{ status: stale ? "error" : undefined }}
      footer={
        <Flex.Box className={CSS.BE("ethercat-sync", "auto")} full="x">
          <Select.Item<number>
            itemKey={AUTO}
            textValue={autoName}
            label={
              <>
                {autoName}
                <Input.AutoBadge className={BADGE_CLASS} />
              </>
            }
          >
            <Icon.Auto />
            <Text.Text>
              {autoName}
              <Input.AutoBadge className={BADGE_CLASS} />
            </Text.Text>
            <Select.ItemDescription>{autoDescription(modes)}</Select.ItemDescription>
          </Select.Item>
        </Flex.Box>
      }
    >
      {modes.map((mode, i) => (
        <Select.Item<number>
          key={i}
          itemKey={i}
          label={modeName(mode, i)}
          textValue={modeName(mode, i)}
        >
          {modeIcon(mode)}
          {modeName(mode, i)}
          <Select.ItemDescription>{describeMode(mode)}</Select.ItemDescription>
        </Select.Item>
      ))}
      {stale && choice != null && (
        <Select.Item<number>
          itemKey={STALE}
          label={staleLabel(choice)}
          textValue="stale"
        >
          <Icon.Warning />
          {staleLabel(choice)}
          <Select.ItemDescription>
            The device no longer lists this mode.
          </Select.ItemDescription>
        </Select.Item>
      )}
    </Select.Simple>
  );
};

interface CycleFieldProps {
  label: string;
  value: number;
  onChange: (ns: number) => void;
  error?: boolean;
}

/** A cycle in nanoseconds, where 0 is the mode's own cycle. */
const CycleField = ({
  label,
  value,
  onChange,
  error,
}: CycleFieldProps): ReactElement => {
  const auto = value === 0;
  return (
    <Input.Item label={label} padHelpText={false}>
      <Input.TimeSpan
        inline
        unit={TimeSpan.MICROSECOND}
        precision={TimeSpan.NANOSECOND}
        value={auto ? undefined : value}
        onChange={onChange}
        onClear={() => onChange(0)}
        placeholder=""
        status={error === true ? "error" : undefined}
        aria-label={label}
        className={CSS.BE("ethercat-sync", "cycle")}
        endContent={auto ? <Input.AutoBadge /> : undefined}
        after={auto ? undefined : <Input.AutoButton onClick={() => onChange(0)} />}
      />
    </Input.Item>
  );
};

/**
 * Sets a cycle on the choice. A cycle typed in auto chooses the mode that auto runs,
 * at the index.
 */
const withCycle = (
  modes: OpMode[],
  choice: OpModeChoice | null,
  index: number,
  key: "sync0Cycle" | "sync1Cycle",
  ns: number,
): OpModeChoice | null => {
  if (choice == null && ns === 0) return null;
  return { ...(choice ?? choose(modes, index)), [key]: ns };
};

interface HelpProps {
  text: string;
  error?: boolean;
}

const Help = ({ text, error }: HelpProps): ReactElement => (
  <Text.Text
    level="small"
    color={error === true ? undefined : 9}
    status={error === true ? "error" : undefined}
  >
    {text}
  </Text.Text>
);

interface FormProps {
  slave: SlaveDevice;
  close: () => void;
}

const Form = ({ slave, close }: FormProps): ReactElement => {
  const [choice, setChoice] = useState<OpModeChoice | null>(
    () => slave.properties.opMode ?? null,
  );
  const { update, variant } = useUpdateOpMode({ afterSuccess: () => close() });
  const modes = slave.properties.opModes;
  const stale = choice != null && isStale(modes, choice);
  const index = choice == null ? autoIndex(modes) : choice.index;
  let mode: OpMode | null = null;
  if (!stale && index != null && takesCycles(modes[index])) mode = modes[index];
  const sync0Problem = mode == null ? null : sync0Error(mode, choice);
  const sync1Problem = mode == null ? null : sync1Error(mode, choice);
  const cycleError = sync0Problem ?? sync1Problem;
  const handleSave = () => {
    if (stale || cycleError != null) return;
    update({ key: slave.key, opMode: choice ?? undefined });
  };
  let help: HelpProps;
  if (stale) help = { text: STALE_ERROR, error: true };
  else if (cycleError != null) help = { text: cycleError, error: true };
  else if (mode != null) help = { text: describeTicks(mode, choice) };
  else if (choice == null) help = { text: autoDescription(modes) };
  else help = { text: describeMode(modes[choice.index]) };
  return (
    <Modal.Frame className={CSS.BE("ethercat-sync", "frame")}>
      <Modal.Header icon={<Icon.Logo.EtherCAT />}>
        {[slave.name, "Sync mode"]}
      </Modal.Header>
      <Modal.Body align="stretch" gap="large">
        <Flex.Box y gap="small">
          <Input.Item label="Mode" padHelpText={false}>
            <ModeSelect
              modes={modes}
              choice={choice}
              stale={stale}
              onChange={setChoice}
            />
          </Input.Item>
          {mode == null && <Help {...help} />}
        </Flex.Box>
        {mode != null && index != null && (
          <Flex.Box y gap="small">
            <Flex.Box x gap="large">
              <CycleField
                label="SYNC0 cycle"
                value={choice == null ? 0 : choice.sync0Cycle}
                error={sync0Problem != null}
                onChange={(ns) =>
                  setChoice(withCycle(modes, choice, index, "sync0Cycle", ns))
                }
              />
              <CycleField
                label="SYNC1 cycle"
                value={choice == null ? 0 : choice.sync1Cycle}
                error={sync1Problem != null}
                onChange={(ns) =>
                  setChoice(withCycle(modes, choice, index, "sync1Cycle", ns))
                }
              />
            </Flex.Box>
            <Help {...help} />
          </Flex.Box>
        )}
        <Text.Text level="small" color={9}>
          {APPLY_NOTE}
        </Text.Text>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End>
          <Button.Button
            status={status.keepVariants(variant, "loading")}
            onClick={handleSave}
            variant="filled"
            trigger={Triggers.SAVE}
            triggerIndicator
            type="submit"
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

interface SyncModeParams {
  deviceKey: device.Key;
}

const SyncMode = ({
  deviceKey,
  close,
}: Modals.ContentProps<SyncModeParams, void>): ReactElement | null => {
  const { data: slave } = useResultSlave({ key: deviceKey });
  if (slave == null) return null;
  return <Form slave={slave} close={close} />;
};

/** Opens the form that sets the sync mode of one EtherCAT slave. */
export const useSyncModeModal = Modals.create(SyncMode);
