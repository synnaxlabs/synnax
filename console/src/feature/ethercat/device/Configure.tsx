// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/ethercat/device/Configure.css";

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Device as PDevice, type Flux } from "@synnaxlabs/pluto";
import { deep, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { useRunningOnNetwork } from "@/feature/ethercat/device/queries";
import { SetupNote } from "@/feature/ethercat/device/SetupNote";
import { SlaveSettings } from "@/feature/ethercat/device/SlaveSettings";
import { StartupParameters } from "@/feature/ethercat/device/StartupParameters";
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
  MAKE,
  type OpMode,
  type OpModeChoice,
  SLAVE_MODEL,
  SLAVE_SCHEMAS,
  type SlaveDevice,
  ZERO_SLAVE_PROPERTIES,
} from "@/feature/ethercat/device/types";
import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { Device } from "@/platform/device";
import { Task } from "@/platform/task";
import { Triggers } from "@/platform/triggers";

const AUTO = -1;
const STALE = -2;

const BADGE_CLASS = CSS.BE("ethercat-sync", "badge");

/** The start of the name of a network that the IgH master runs. */
const IGH_PREFIX = "igh:";

const STALE_ERROR =
  "The device no longer lists this mode, so the network fails to start. Pick a mode.";

/** Says which mode auto runs on the device, and why. */
const autoDescription = (modes: OpMode[], catalog: OpModeChoice | null): string => {
  const auto = autoIndex(modes, catalog);
  if (auto == null)
    return "The Driver uses SM sync, because the device lists no modes.";
  if (catalog != null)
    return (
      `The Driver uses ${modeName(modes[auto], auto)}, because the setup of the ` +
      "device needs it. Best for most needs."
    );
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
  /** The choice that the catalog of the device gives, or null for none. */
  catalog: OpModeChoice | null;
  value: OpModeChoice | null;
  stale: boolean;
  onChange: (choice: OpModeChoice | null) => void;
}

const ModeSelect = ({
  modes,
  catalog,
  value: choice,
  stale,
  onChange,
}: ModeSelectProps): ReactElement => {
  const auto = autoIndex(modes, catalog);
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
            <Select.ItemDescription>
              {autoDescription(modes, catalog)}
            </Select.ItemDescription>
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

type Problem = "error" | "warning";

interface CycleFieldProps {
  label: string;
  value: number;
  onChange: (ns: number) => void;
  problem?: Problem;
}

/** A cycle in nanoseconds, where 0 is the mode's own cycle. */
const CycleField = ({
  label,
  value,
  onChange,
  problem,
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
        status={problem}
        aria-label={label}
        className={CSS.BE("ethercat-sync", "cycle")}
        endContent={auto ? <Input.AutoBadge /> : undefined}
        after={auto ? undefined : <Input.AutoButton onClick={() => onChange(0)} />}
      />
    </Input.Item>
  );
};

/**
 * Sets a cycle on the choice. A cycle typed in auto chooses what auto runs: the choice
 * of the catalog, else the mode at the index.
 */
const withCycle = (
  modes: OpMode[],
  choice: OpModeChoice | null,
  auto: OpModeChoice | null,
  index: number,
  key: "sync0Cycle" | "sync1Cycle",
  ns: number,
): OpModeChoice | null => {
  if (choice == null && ns === 0) return null;
  return { ...(choice ?? auto ?? choose(modes, index)), [key]: ns };
};

interface HelpProps {
  text: string;
  problem?: Problem;
}

const Help = ({ text, problem }: HelpProps): ReactElement => (
  <Text.Text level="small" color={problem == null ? 9 : undefined} status={problem}>
    {text}
  </Text.Text>
);
const path = deep.path<SlaveDevice>();

const OP_MODE_PATH = path.properties.opMode;

const INITIAL_VALUES: SlaveDevice = {
  key: "",
  rack: 0,
  name: "",
  make: MAKE,
  model: SLAVE_MODEL,
  location: "",
  configured: false,
  properties: ZERO_SLAVE_PROPERTIES,
};

const useForm = PDevice.createForm(SLAVE_SCHEMAS, INITIAL_VALUES);

/** A check never makes the form invalid, so a field with an error blocks the save. */
const hasError = (statuses: Status.Crude[]): boolean =>
  statuses.some(({ variant }) => variant === "error");

const beforeSave = async ({
  getStatuses,
  set,
}: Flux.FormBeforeSaveParams<
  PDevice.RetrieveQuery,
  PDevice.FormSchema<typeof SLAVE_SCHEMAS>
>): Promise<boolean> => {
  if (hasError(getStatuses())) return false;
  set(path.configured, true);
  return true;
};

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "EtherCAT module",
  "aria-label": "Name",
} as const;

const General = (): ReactElement => {
  const name = Form.useFieldValue(path.name);
  const key = Form.useFieldValue(path.key);
  const suggestions = Device.useIdentifierSuggestions(name ?? "", key);
  const { set } = Form.useContext();
  return (
    <Form.Section title="General">
      <Form.TextField
        path={path.properties.identifier}
        label="Identifier"
        padHelpText={false}
        inputProps={{ placeholder: "dev1" }}
      />
      <Flex.Box y gap="small">
        <Text.Text level="small" color={9}>
          A short prefix for every channel on this device.
        </Text.Text>
        <Flex.Box x align="center" wrap>
          <Icon.Bolt />
          {suggestions.map((id) => (
            <Button.Button
              key={id}
              onClick={() => set(path.properties.identifier, id)}
              size="small"
              variant="text"
            >
              {id}
            </Button.Button>
          ))}
        </Flex.Box>
      </Flex.Box>
    </Form.Section>
  );
};

interface SaveOptions {
  createTask?: (deviceKey: string) => void;
  /** Stops the tasks that run on the network during the save, then starts them. */
  restart?: boolean;
}

/**
 * The form that sets the name, identifier, sync mode, settings, and startup parameters
 * of one EtherCAT slave.
 */
export const Configure = ({
  deviceKey,
  close,
  onBack,
  onCreateReadTask,
  onCreateWriteTask,
}: Device.ConfigureProps): ReactElement => {
  const { form, saveAsync, variant } = useForm({
    query: { key: deviceKey },
    beforeSave,
  });
  // Save sets configured, so the dialog keeps the slave as it was at open.
  const [opened] = useState(form.value);
  const checked = useRef(false);
  // The checks show the platform's messages in place of the stored schema's.
  useEffect(() => form.addCheck(path.name, Device.nameZ), [form]);
  const modes = Form.useFieldValue(path.properties.opModes, { ctx: form }) ?? [];
  const choice =
    Form.useFieldValue(OP_MODE_PATH, { ctx: form, optional: true }) ?? null;
  const setChoice = (next: OpModeChoice | null) =>
    form.set(OP_MODE_PATH, next ?? undefined);
  const settings = Form.useFieldValue(path.properties.settings, {
    ctx: form,
    optional: true,
  });
  const given = settings?.mode?.opMode ?? null;
  // The Driver writes no startup parameters to a slave that other software sets up.
  const shared = settings?.mode?.shared === true;
  const network = Form.useFieldValue(path.properties.network, { ctx: form });
  const pdos = Form.useFieldValue(path.properties.pdos, { ctx: form });
  // The bus applies the settings of each slave on the network, configured or not.
  const running = useRunningOnNetwork(opened.rack, network);
  const handleError = Status.useErrorHandler();
  // The IgH master runs no DC sync, so it does not run the choice of the catalog.
  const catalog =
    given != null && !isStale(modes, given) && !network.startsWith(IGH_PREFIX)
      ? given
      : null;
  const { capture } = Analytics.use();
  const stale = choice != null && isStale(modes, choice);
  const index = choice == null ? autoIndex(modes, catalog) : choice.index;
  // What the Driver runs: the choice, else the choice of the catalog.
  const effective = choice ?? catalog;
  let mode: OpMode | null = null;
  if (!stale && index != null && takesCycles(modes[index])) mode = modes[index];
  const sync0Problem = mode == null ? null : sync0Error(mode, effective);
  const sync1Problem =
    mode == null || sync0Problem != null ? null : sync1Error(mode, effective);
  const cycleProblem = sync0Problem ?? sync1Problem;
  // Auto fails the start too, but Save must still remove the choice.
  const severity: Problem = choice == null ? "warning" : "error";
  const cycleError = choice == null ? null : cycleProblem;
  const save = async ({ createTask, restart = false }: SaveOptions = {}) => {
    // A new slave has no identifier, so its check applies from the first Save on.
    if (!checked.current) {
      form.addCheck(path.properties.identifier, Device.identifierZ);
      checked.current = true;
    }
    if (stale || cycleError != null) return;
    // A mode that takes no cycles fails the start with any cycle.
    if (choice != null && mode == null) setChoice(choose(modes, choice.index));
    // A save that fails must not stop the tasks.
    if (restart && (!form.validate() || hasError(form.getStatuses()))) return;
    const saved = restart
      ? await Task.whileStopped({ tasks: running }, saveAsync)
      : await saveAsync();
    if (!saved) return;
    if (!opened.configured) capture("device_configured", { make: MAKE });
    createTask?.(deviceKey);
    close();
  };
  // In auto, the Mode select says what runs. Cycles show only to fix a failed start.
  let help: HelpProps | null = null;
  if (stale) help = { text: STALE_ERROR, problem: "error" };
  else if (cycleProblem != null) help = { text: cycleProblem, problem: severity };
  else if (choice != null)
    help = {
      text:
        mode == null ? describeMode(modes[choice.index]) : describeTicks(mode, choice),
    };
  const syncMode = (
    <Form.Section title="Sync mode">
      <Input.Item label="Mode" padHelpText={false}>
        <ModeSelect
          modes={modes}
          catalog={catalog}
          value={choice}
          stale={stale}
          onChange={setChoice}
        />
      </Input.Item>
      {mode != null && index != null && (choice != null || cycleProblem != null) && (
        <Flex.Box x>
          <CycleField
            label="SYNC0 cycle"
            value={choice == null ? 0 : choice.sync0Cycle}
            problem={sync0Problem == null ? undefined : severity}
            onChange={(ns) =>
              setChoice(withCycle(modes, choice, catalog, index, "sync0Cycle", ns))
            }
          />
          <CycleField
            label="SYNC1 cycle"
            value={choice == null ? 0 : choice.sync1Cycle}
            problem={sync1Problem == null ? undefined : severity}
            onChange={(ns) =>
              setChoice(withCycle(modes, choice, catalog, index, "sync1Cycle", ns))
            }
          />
        </Flex.Box>
      )}
      {help != null && <Help {...help} />}
    </Form.Section>
  );
  return (
    <Modal.Frame
      className={CSS.cls(
        CSS.BE("ethercat-configure", "frame"),
        !shared && CSS.M("wide"),
      )}
    >
      <Modal.Header icon={<Icon.Logo.EtherCAT />}>
        {[opened.name, "Configure"]}
      </Modal.Header>
      <Modal.Body align="stretch" gap="large">
        <Form.Form<PDevice.FormSchema<typeof SLAVE_SCHEMAS>> {...form}>
          <Form.TextField
            path={path.name}
            showLabel={false}
            padHelpText={false}
            inputProps={NAME_INPUT_PROPS}
          />
          <SetupNote deviceKey={deviceKey} />
          <div className={CSS.BE("ethercat-configure", "columns")}>
            {shared ? (
              <Form.Sections>
                <General />
                <SlaveSettings />
                {syncMode}
              </Form.Sections>
            ) : (
              <>
                <Form.Sections>
                  <General />
                  <SlaveSettings />
                </Form.Sections>
                <Form.Sections>
                  {syncMode}
                  <Form.Section title="Startup parameters">
                    <StartupParameters />
                  </Form.Section>
                </Form.Sections>
              </>
            )}
          </div>
        </Form.Form>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.Start gap="small">
          {onBack != null && (
            <Button.Button variant="text" onClick={onBack}>
              <Icon.Arrow.Left />
              Back
            </Button.Button>
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End gap="small">
          <Text.Text level="small" color={9}>
            Changes apply after every task on this network restarts.
          </Text.Text>
          {onCreateReadTask != null && !opened.configured && pdos.inputs.length > 0 && (
            <Button.Button
              status={status.keepVariants(variant, "loading")}
              onClick={() => void save({ createTask: onCreateReadTask })}
            >
              Save and create read task
            </Button.Button>
          )}
          {onCreateWriteTask != null &&
            !opened.configured &&
            pdos.outputs.length > 0 && (
              <Button.Button
                status={status.keepVariants(variant, "loading")}
                onClick={() => void save({ createTask: onCreateWriteTask })}
              >
                Save and create write task
              </Button.Button>
            )}
          {running.length > 0 && (
            <Button.Button
              status={status.keepVariants(variant, "loading")}
              onClick={() =>
                handleError(
                  async () => await save({ restart: true }),
                  "Failed to restart the tasks",
                )
              }
            >
              Save and restart tasks
            </Button.Button>
          )}
          <Button.Button
            status={status.keepVariants(variant, "loading")}
            onClick={() => void save()}
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
