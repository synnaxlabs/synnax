// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, rack, task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Channel, Form, Rack, Status, Synnax } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";

import { Editor } from "@/code/Editor";
import { NULL_CLIENT_ERROR } from "@/errors";
import { Common } from "@/hardware/common";
import {
  type Config,
  configZ,
  type StateDetails,
  TYPE,
  type Type,
  ZERO_PAYLOAD,
} from "@/hardware/task/sequence/types";
import { type Layout } from "@/layout";
import { type Modals } from "@/modals";

export const LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  icon: "Control",
  name: ZERO_PAYLOAD.name,
  type: TYPE,
};

export interface CreateNewLayoutArgs {
  rackKey?: rack.Key;
  rename: Modals.PromptRename;
}

export const createNewLayout = async ({
  rackKey,
  rename,
}: CreateNewLayoutArgs): Promise<Common.Task.Layout | null> => {
  const name = await rename({}, { icon: "Control", name: "Control.Sequence.Create" });
  return name == null ? null : { ...LAYOUT, name, args: { rackKey } };
};

export const SELECTABLE: Layout.Selectable = {
  key: TYPE,
  title: "Control Sequence",
  icon: <Icon.Control />,
  create: async ({ layoutKey, rename }) => {
    const layout = await createNewLayout({ rename });
    return layout == null ? null : { ...layout, key: layoutKey };
  },
};

const schema = z.object({
  rack: rack.keyZ.min(1, "Rack is required"),
  config: configZ,
});

const Internal = ({
  task: base,
  layoutKey,
  configured,
  rackKey,
}: Common.Task.TaskProps<Config, StateDetails, Type>) => {
  const client = Synnax.use();
  const handleException = Status.useExceptionHandler();
  const methods = Form.use({
    values: { rack: rackKey ?? task.getRackKey(base.key ?? "0"), config: base.config },
    schema,
  });
  const create = Common.Task.useCreate(layoutKey);
  const [state, setState] = Common.Task.useState(base?.key, base?.state ?? undefined);

  const configureMutation = useMutation({
    mutationFn: async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      if (!(await methods.validateAsync())) return;
      const { config, rack } = methods.value();
      await create({ key: base.key, name: base.name, type: TYPE, config }, rack);
      setState("paused");
    },
    onError: (e) => handleException(e, `Failed to configure ${base.name}`),
  });
  const startOrStopMutation = useMutation({
    mutationFn: async () => {
      if (!configured) throw new Error("Sequence has not been configured");
      if (state.state === "loading")
        throw new Error(
          "State is loading, should not be able to start or stop sequence",
        );
      await base.executeCommand(state.state === "running" ? "stop" : "start");
    },
    onError: (e) =>
      handleException(
        e,
        `Failed to ${state.state === "running" ? "stop" : state.state === "paused" ? "start" : "start or stop"} task`,
      ),
  });
  const isSnapshot = base?.snapshot ?? false;

  const isLoading = state.state === "loading";
  const isConfiguring = configureMutation.isPending;
  const isDisabled = isLoading || isConfiguring || isSnapshot;

  return (
    <Align.Space
      style={{ padding: 0, height: "100%", minHeight: 0 }}
      direction="y"
      empty
    >
      <Form.Form {...methods} mode={base?.snapshot ? "preview" : "normal"}>
        <Form.Field<string>
          path="config.script"
          showLabel={false}
          showHelpText={false}
          padHelpText={false}
          style={{
            height: "100%",
            width: "100%",
            minHeight: 0,
            display: "flex",
            flex: 1,
            flexShrink: 1,
            overflow: "hidden",
          }}
        >
          {(p) => <Editor style={{ height: "100%", width: "100%", flex: 1 }} {...p} />}
        </Form.Field>
        <Align.Pack
          direction="y"
          bordered={false}
          style={{
            width: "100%",
            background: "var(--pluto-gray-l0)",
            boxShadow: "var(--pluto-shadow-menu)",
            borderTop: "var(--pluto-border)",
            flexShrink: 0, // Prevent the bottom section from shrinking
          }}
        >
          <Align.Space direction="y" style={{ padding: "2rem" }}>
            <Align.Space direction="x">
              <Form.Field<rack.Key>
                path="rack"
                label="Location"
                padHelpText={false}
                grow
              >
                {(p) => <Rack.SelectSingle allowNone={false} {...p} />}
              </Form.Field>
              <Form.NumericField
                label="Loop Rate"
                path="config.rate"
                padHelpText={false}
                style={{ width: 120 }}
                inputProps={{
                  endContent: "Hz",
                  bounds: { lower: 1, upper: 1001 },
                  dragScale: { x: 1, y: 1 },
                }}
              />
            </Align.Space>
            <Form.Field<channel.Key[]>
              path="config.read"
              label="Read From"
              padHelpText={false}
            >
              {({ value, onChange }) => (
                <Channel.SelectMultiple value={value} onChange={onChange} />
              )}
            </Form.Field>
            <Form.Field<channel.Key[]>
              path="config.write"
              label="Write To"
              padHelpText={false}
            >
              {({ value, onChange }) => (
                <Channel.SelectMultiple value={value} onChange={onChange} />
              )}
            </Form.Field>
          </Align.Space>
          <Align.Space
            direction="x"
            rounded
            style={{ padding: "2rem", borderTop: "var(--pluto-border)" }}
            justify="spaceBetween"
          >
            <Align.Space direction="x" style={{ borderRadius: "1rem", width: "100%" }}>
              {state.message != null && (
                <Status.Text variant={state.variant ?? "info"}>
                  {state.message}
                </Status.Text>
              )}
            </Align.Space>
            <Button.Button
              loading={isConfiguring}
              disabled={isDisabled}
              onClick={() => configureMutation.mutate()}
            >
              Configure
            </Button.Button>
            <Button.Icon
              loading={isLoading}
              disabled={isDisabled}
              onClick={() => startOrStopMutation.mutate()}
              variant="outlined"
            >
              {state.state === "running" ? <Icon.Pause /> : <Icon.Play />}
            </Button.Icon>
          </Align.Space>
        </Align.Pack>
      </Form.Form>
    </Align.Space>
  );
};

export const Sequence = Common.Task.wrap(Internal, {
  getInitialPayload: () => ZERO_PAYLOAD,
  configSchema: configZ,
});
