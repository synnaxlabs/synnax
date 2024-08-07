import "@/hardware/task/common/common.css";

import { task, UnexpectedError } from "@synnaxlabs/client";
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
} from "@synnaxlabs/pluto";
import { caseconv, deep, Key, Keyed, Optional, UnknownRecord } from "@synnaxlabs/x";
import { useQuery } from "@tanstack/react-query";
import { FC, ReactElement, useCallback, useState } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { Menu as CMenu } from "@/components/menu";
import { CSS } from "@/css";
import { Layout } from "@/layout";

export interface ControlsProps {
  onStartStop: () => void;
  startingOrStopping: boolean;
  onConfigure: () => void;
  configuring: boolean;
  state?: task.State<{ running?: boolean; message?: string }>;
}

const CONFIGURE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export interface ChannelListEmptyContentProps {
  onAdd: () => void;
}

export const ChannelListEmptyContent = ({ onAdd }: ChannelListEmptyContentProps) => (
  <Align.Space direction="y" style={{ height: "100%" }}>
    <Align.Center direction="y">
      <Text.Text level="p">No channels in task.</Text.Text>
      <Text.Link level="p" onClick={onAdd}>
        Add a channel
      </Text.Link>
    </Align.Center>
  </Align.Space>
);

interface ChannelListContextMenuProps<T> {
  keys: string[];
  value: T[];
  onSelect: (keys: string[], index: number) => void;
  remove: (indices: number[]) => void;
  onDuplicate?: (indices: number[]) => void;
  path: string;
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
  const allowDisable = indices.some((i) => value[i].enabled);
  const allowEnable = indices.some((i) => !value[i].enabled);
  return (
    <Menu.Menu
      onChange={{
        remove: handleRemove,
        duplicate: handleDuplicate,
        disable: handleDisable,
        enable: handleEnable,
      }}
      level="small"
    >
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
      else if (state.details != null && state.details.errors != null) {
        state.details.errors.forEach((e) => {
          const path = `config.${caseconv.snakeToCamel(e.path)}`;
          setStatus(path, { variant: "error", message: "" });
        });
      }
    },
  });
  return taskState;
};

export const Controls = ({
  state,
  onStartStop,
  startingOrStopping,
  onConfigure,
  configuring,
}: ControlsProps) => (
  <Align.Space direction="x" className={CSS.B("task-controls")} justify="spaceBetween">
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
      {state?.details?.message != null && (
        <Status.Text variant={state?.variant as Status.Variant}>
          {state?.details?.message}
        </Status.Text>
      )}
    </Align.Space>
    <Align.Space
      direction="x"
      style={{
        borderRadius: "1rem",
        border: "var(--pluto-border)",
        padding: "2rem",
      }}
      justify="end"
    >
      <Align.Space direction="y">
        <Button.Icon
          loading={startingOrStopping}
          disabled={startingOrStopping || state == null}
          onClick={onStartStop}
          variant="outlined"
        >
          {state?.details?.running === true ? <Icon.Pause /> : <Icon.Play />}
        </Button.Icon>
      </Align.Space>
      <Align.Space direction="y">
        <Button.Button
          loading={configuring}
          disabled={configuring}
          onClick={onConfigure}
          triggers={[CONFIGURE_TRIGGER]}
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
  </Align.Space>
);

export const ChannelListHeader = ({ onAdd }: ChannelListEmptyContentProps) => (
  <Header.Header level="h4">
    <Header.Title weight={500}>Channels</Header.Title>
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
  </Header.Header>
);

export interface EnableDisableButtonProps {
  value: boolean;
  onChange: (v: boolean) => void;
}

export const EnableDisableButton = ({
  value,
  onChange: onClick,
}: EnableDisableButtonProps) => (
  <Button.Toggle
    checkedVariant="outlined"
    uncheckedVariant="outlined"
    className={CSS.B("enable-disable-button")}
    value={value}
    size="small"
    onClick={(e) => e.stopPropagation()}
    onChange={onClick}
    tooltip={
      <Text.Text level="small" style={{ maxWidth: 300 }}>
        Data acquisition for this channel is {value ? "enabled" : "disabled"}. Click to
        {value ? " disable" : " enable"} it.
      </Text.Text>
    }
  >
    <Status.Text variant={value ? "success" : "disabled"} level="small" align="center">
      {value ? "Enabled" : "Disabled"}
    </Status.Text>
  </Button.Toggle>
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

export const wrapTaskLayout = <T extends task.Task, P extends task.Payload>(
  Wrapped: FC<WrappedTaskLayoutProps<T, P>>,
  zeroPayload: P,
): Layout.Renderer => {
  const Wrapper: Layout.Renderer = ({ layoutKey }) => {
    const client = Synnax.use();
    const args = Layout.useSelectArgs<{ create: boolean }>(layoutKey);
    const altKey = Layout.useSelectAltKey(layoutKey);
    const fetchTask = useQuery<WrappedTaskLayoutProps<T, P>>({
      queryKey: [layoutKey, client?.key, altKey],
      queryFn: async () => {
        if (client == null || args.create)
          return { initialValues: deep.copy(zeroPayload), layoutKey };
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
    else if (!fetchTask.isPending)
      content = <Wrapped {...fetchTask.data} layoutKey={layoutKey} />;
    return <Eraser.Eraser>{content}</Eraser.Eraser>;
  };
  Wrapper.displayName = `TaskWrapper(${Wrapped.displayName ?? Wrapped.name})`;
  return Wrapper;
};
