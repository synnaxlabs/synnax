// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/task/common/common.css";

import { ontology, type task, UnexpectedError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Eraser,
  Form,
  Header,
  Menu,
  Observe,
  Status,
  Synnax,
  Text,
  Triggers,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import {
  caseconv,
  deep,
  type Key,
  type Keyed,
  type migrate,
  type Optional,
  type UnknownRecord,
} from "@synnaxlabs/x";
import { useQuery } from "@tanstack/react-query";
import { type Migratable } from "node_modules/@synnaxlabs/x/dist/src/migrate/migrate";
import { type FC, type ReactElement, useCallback, useId, useState } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { Menu as CMenu } from "@/components/menu";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { overviewLayout } from "@/range/external";

export interface ControlsProps {
  layoutKey: string;
  onStartStop: () => void;
  startingOrStopping: boolean;
  onConfigure: () => void;
  configuring: boolean;
  snapshot?: boolean;
  state?: task.State<{ running?: boolean; message?: string }>;
}

const CONFIGURE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export interface ChannelListEmptyContentProps {
  onAdd: () => void;
  snapshot?: boolean;
}

export const ChannelListEmptyContent = ({
  onAdd,
  snapshot = false,
}: ChannelListEmptyContentProps) => (
  <Align.Space direction="y" style={{ height: "100%" }}>
    <Align.Center direction="y">
      <Text.Text level="p">No channels in task.</Text.Text>
      {!snapshot && (
        <Text.Link level="p" onClick={onAdd}>
          Add a channel
        </Text.Link>
      )}
    </Align.Center>
  </Align.Space>
);

interface ChannelListContextMenuProps<T> {
  keys: string[];
  value: T[];
  onSelect: (keys: string[], index: number) => void;
  remove: (indices: number[]) => void;
  path: string;
  onDuplicate?: (indices: number[]) => void;
  snapshot?: boolean;
  allowTare?: boolean;
  onTare?: (indices: number[]) => void;
}

export const ChannelListContextMenu = <
  K extends Key,
  T extends Keyed<K> & { enabled: boolean },
>({
  keys,
  value,
  onSelect,
  remove,
  onDuplicate,
  path,
  snapshot,
  allowTare,
  onTare,
}: ChannelListContextMenuProps<T>) => {
  const methods = Form.useContext();
  const indices = keys.map((k) => value.findIndex((v) => v.key === k));
  const handleRemove = () => {
    remove(indices);
    onSelect([], -1);
  };
  const handleDuplicate = () => onDuplicate?.(indices);
  const handleDisable = () =>
    value.forEach((_, i) => {
      if (!indices.includes(i)) return;
      methods.set(`${path}.${i}.enabled`, false);
    });
  const handleEnable = () =>
    value.forEach((_, i) => {
      if (!indices.includes(i)) return;
      methods.set(`${path}.${i}.enabled`, true);
    });
  const handleTare = () => onTare?.(indices);
  const allowDisable = indices.some((i) => value[i].enabled);
  const allowEnable = indices.some((i) => !value[i].enabled);
  return (
    <Menu.Menu
      onChange={{
        remove: handleRemove,
        duplicate: handleDuplicate,
        disable: handleDisable,
        enable: handleEnable,
        tare: handleTare,
      }}
      level="small"
    >
      {!snapshot && (
        <>
          <Menu.Item itemKey="remove" startIcon={<Icon.Close />}>
            Remove
          </Menu.Item>
          {onDuplicate != null && (
            <Menu.Item itemKey="duplicate" startIcon={<Icon.Copy />}>
              Duplicate
            </Menu.Item>
          )}
          <Menu.Divider />
          {allowDisable && (
            <Menu.Item itemKey="disable" startIcon={<Icon.Disable />}>
              Disable
            </Menu.Item>
          )}
          {allowEnable && (
            <Menu.Item itemKey="enable" startIcon={<Icon.Enable />}>
              Enable
            </Menu.Item>
          )}
          {(allowEnable || allowDisable) && <Menu.Divider />}
          {allowTare && (
            <>
              <Menu.Item itemKey="tare" startIcon={<Icon.Tare />}>
                Tare
              </Menu.Item>
              <Menu.Divider />
            </>
          )}
        </>
      )}
      <CMenu.HardReloadItem />
    </Menu.Menu>
  );
};

export const parserErrorZ = z.object({
  message: z.string(),
  path: z.string(),
});

export type ParserError = z.infer<typeof parserErrorZ>;

export const parserErrorsZ = z.array(parserErrorZ);

export type ParserErrors = z.infer<typeof parserErrorsZ>;

interface ParserErrorsDetails extends UnknownRecord {
  errors?: ParserErrors;
}

export const useObserveState = <T extends ParserErrorsDetails>(
  setStatus: Form.UseReturn<any>["setStatus"],
  clearStatuses: Form.UseReturn<any>["clearStatuses"],
  taskKey?: string,
  initialState?: task.State<T>,
): task.State<T> | undefined => {
  const client = Synnax.use();
  const [taskState, setTaskState] = useState<task.State<T> | undefined>(initialState);
  Observe.useListener({
    key: [taskKey],
    open: async () => await client?.hardware.tasks.openStateObserver<T>(),
    onChange: (state) => {
      if (state.task !== taskKey) return;
      setTaskState(state);
      if (state.variant !== "error") clearStatuses();
      else if (state.details != null && state.details.errors != null) 
        state.details.errors.forEach((e) => {
          const path = `config.${caseconv.snakeToCamel(e.path)}`;
          setStatus(path, { variant: "error", message: "" });
        });
      
    },
  });
  return taskState;
};

export const Controls = ({
  state,
  onStartStop,
  layoutKey,
  startingOrStopping,
  onConfigure,
  configuring,
  snapshot = false,
}: ControlsProps) => {
  let content: ReactElement | null = null;
  if (state?.details?.message != null)
    content = (
      <Status.Text variant={state?.variant as Status.Variant}>
        {state?.details?.message}
      </Status.Text>
    );
  if (snapshot)
    content = (
      <Status.Text.Centered variant="disabled" hideIcon>
        This task is a snapshot and cannot be modified or started.
      </Status.Text.Centered>
    );
  const isActive = Layout.useSelectActiveMosaicTabKey() === layoutKey;
  return (
    <Align.Space
      direction="x"
      className={CSS.B("task-controls")}
      justify="spaceBetween"
    >
      <Align.Space
        className={CSS.B("task-state")}
        direction="x"
        style={{
          borderRadius: "1rem",
          border: "var(--pluto-border)",
          padding: "2rem",
          width: "100%",
        }}
      >
        {content}
      </Align.Space>
      <Align.Space
        direction="x"
        bordered
        rounded
        style={{
          padding: "2rem",
          borderRadius: "1rem",
        }}
        justify="end"
      >
        <Button.Icon
          loading={startingOrStopping}
          disabled={startingOrStopping || state == null || snapshot}
          onClick={onStartStop}
          variant="outlined"
        >
          {state?.details?.running === true ? <Icon.Pause /> : <Icon.Play />}
        </Button.Icon>
        <Button.Button
          loading={configuring}
          disabled={configuring || snapshot}
          onClick={onConfigure}
          triggers={isActive ? [CONFIGURE_TRIGGER] : undefined}
          tooltip={
            <Align.Space direction="x" align="center" size="small">
              <Triggers.Text shade={7} level="small" trigger={CONFIGURE_TRIGGER} />
              <Text.Text shade={7} level="small">
                To Configure
              </Text.Text>
            </Align.Space>
          }
        >
          Configure
        </Button.Button>
      </Align.Space>
    </Align.Space>
  );
};

export interface ChannelListHeaderProps extends ChannelListEmptyContentProps {}

export const ChannelListHeader = ({ onAdd, snapshot }: ChannelListHeaderProps) => (
  <Header.Header level="h4">
    <Header.Title weight={500}>Channels</Header.Title>
    {!snapshot && (
      <Header.Actions>
        {[
          {
            key: "add",
            onClick: onAdd,
            children: <Icon.Add />,
            size: "large",
          },
        ]}
      </Header.Actions>
    )}
  </Header.Header>
);

export interface EnableDisableButtonProps {
  value: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
  snapshot?: boolean;
}

export const EnableDisableButton = ({
  value,
  onChange,
  disabled,
  snapshot = false,
}: EnableDisableButtonProps) => (
  <Button.ToggleIcon
    checkedVariant={snapshot ? "preview" : "outlined"}
    uncheckedVariant={snapshot ? "preview" : "outlined"}
    className={CSS.B("enable-disable-button")}
    disabled={disabled}
    value={value}
    size="small"
    onClick={(e) => e.stopPropagation()}
    onChange={(v) => onChange(v)}
    tooltip={
      <Text.Text level="small" style={{ maxWidth: 300 }}>
        Data acquisition for this channel is {value ? "enabled" : "info"}. Click to
        {value ? " disable" : " enable"} it.
      </Text.Text>
    }
  >
    <Status.Circle variant={value ? "success" : "disabled"} />
  </Button.ToggleIcon>
);

export interface TareButtonProps {
  onClick: () => void;
  disabled?: boolean;
}

export const TareButton = ({ onClick, disabled }: TareButtonProps) => (
    <Button.Icon
      variant={"outlined"}
      disabled={disabled}
      onClick={onClick}
      tooltip="Click to tare"
    >
      <Icon.Tare />
    </Button.Icon>
  );

export const useCreate = <
  C extends UnknownRecord,
  D extends {} = UnknownRecord,
  T extends string = string,
>(
  layoutKey: string,
): ((
  t: Optional<task.NewTask<C, T>, "key">,
) => Promise<task.Task<C, D, T> | undefined>) => {
  const client = Synnax.use();
  const dispatch = useDispatch();
  return useCallback(
    async (pld: task.NewTask<C, T>) => {
      if (client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      const ot = await rack.createTask<C, D, T>(pld);
      dispatch(Layout.setAltKey({ key: layoutKey, altKey: ot.key }));
      dispatch(Layout.setArgs({ key: layoutKey, args: { create: false } }));
      return ot;
    },
    [client, layoutKey],
  );
};

export interface WrappedTaskLayoutProps<T extends task.Task, P extends task.Payload> {
  layoutKey: string;
  task?: T;
  initialValues: P;
}

export interface TaskLayoutArgs<P extends task.Payload> {
  create: boolean;
  initialValues?: deep.Partial<P>;
}

export const wrapTaskLayout = <T extends task.Task, P extends task.Payload>(
  Wrapped: FC<WrappedTaskLayoutProps<T, P>>,
  zeroPayload: P,
  migrator?: migrate.Migrator,
): Layout.Renderer => {
  const Wrapper: Layout.Renderer = ({ layoutKey }) => {
    const client = Synnax.use();
    const args = Layout.useSelectArgs<TaskLayoutArgs<P>>(layoutKey);
    const altKey = Layout.useSelectAltKey(layoutKey);
    const id = useId();
    // The query can't take into account state changes, so we need to use a unique
    // key for every query.
    const fetchTask = useQuery<WrappedTaskLayoutProps<T, P>>({
      queryKey: [layoutKey, client?.key, altKey, id],
      queryFn: async () => {
        if (client == null || args.create) {
          let initialValues = deep.copy(zeroPayload);
          if (args.initialValues != null)
            initialValues = deep.override(initialValues, args.initialValues);
          return { initialValues, layoutKey };
        }
        // try to parse the key as a big int. If the parse fails, set the lat key as a key
        let key: string = layoutKey;
        try {
          BigInt(layoutKey);
        } catch (e) {
          if (altKey == undefined)
            throw new UnexpectedError(
              `Task has non-bigint layout key ${layoutKey} with no alternate key`,
            );
          if (e instanceof SyntaxError) key = altKey;
        }
        const t = await client.hardware.tasks.retrieve(key, { includeState: true });
        if (migrator != null) t.config = migrator(t.config as Migratable);
        return { initialValues: t as unknown as P, task: t as T, layoutKey };
      },
    });
    let content: ReactElement | null = null;
    if (fetchTask.isError)
      content = (
        <Align.Space direction="y" grow style={{ height: "100%" }}>
          <Status.Text.Centered variant="error">
            {fetchTask.error.message}
          </Status.Text.Centered>
        </Align.Space>
      );
    else if (fetchTask.isSuccess)
      content = <Wrapped {...fetchTask.data} layoutKey={layoutKey} />;
    return <Eraser.Eraser>{content}</Eraser.Eraser>;
  };
  Wrapper.displayName = `TaskWrapper(${Wrapped.displayName ?? Wrapped.name})`;
  return Wrapper;
};

export interface ParentRangeButtonProps {
  taskKey?: string;
}

export const ParentRangeButton = ({
  taskKey,
}: ParentRangeButtonProps): ReactElement | null => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const [parent, setParent] = useState<ontology.Resource | null>();
  const placer = Layout.usePlacer();

  useAsyncEffect(async () => {
    try {
      if (client == null || taskKey == null) return;
      const tsk = await client.hardware.tasks.retrieve(taskKey);
      const parent = await tsk.snapshottedTo();
      if (parent != null) setParent(parent);
      const tracker = await client.ontology.openDependentTracker({
        target: new ontology.ID({ key: taskKey, type: "task" }),
        dependents: parent == null ? [] : [parent],
        relationshipDirection: "to",
      });
      tracker.onChange((parents) => {
        if (parents.length === 0) return setParent(null);
        setParent(parents[0]);
      });
      return async () => await tracker.close();
    } catch (e) {
      addStatus({
        variant: "error",
        message: `Failed to retrieve child ranges`,
        description: (e as Error).message,
      });
      return undefined;
    }
  }, [taskKey, client?.key]);
  if (parent == null) return null;
  return (
    <Align.Space direction="x" size="small" align="center">
      <Text.Text level="p">Snapshotted to</Text.Text>
      <Button.Button
        variant="text"
        shade={7}
        weight={400}
        startIcon={<Icon.Range />}
        iconSpacing="small"
        style={{ padding: "1rem" }}
        onClick={() =>
          placer({ ...overviewLayout, key: parent.id.key, name: parent.name })
        }
      >
        {parent.name}
      </Button.Button>
    </Align.Space>
  );
};
