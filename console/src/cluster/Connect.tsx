// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/cluster/Connect.css";

import { type connection, type SynnaxProps, synnaxPropsZ } from "@synnaxlabs/client";
import {
  Align,
  Button,
  componentRenderProp,
  Form,
  Input,
  Nav,
  Status,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { statusVariants } from "@/cluster/Badges";
import { useSelectMany } from "@/cluster/selectors";
import { set, setActive } from "@/cluster/slice";
import { testConnection } from "@/cluster/testConnection";
import { CSS } from "@/css";
import { Layout } from "@/layout";

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const connectWindowLayout: Layout.State = {
  key: "connectCluster",
  windowKey: "connectCluster",
  type: "connectCluster",
  name: "Cluster.Connect",
  icon: "Cluster",
  location: "modal",
  window: {
    resizable: false,
    size: { height: 430, width: 650 },
    navTop: true,
  },
};

/**
 * Connect implements the LayoutRenderer component type to provide a form for connecting
 * to a cluster.
 */
export const Connect = ({ onClose }: Layout.RendererProps): ReactElement => {
  const dispatch = useDispatch();
  const [connState, setConnState] = useState<connection.State | null>(null);
  const [loading, setLoading] = useState<"test" | "submit" | null>(null);
  const names = useSelectMany().map((c) => c.name);

  const formSchema = synnaxPropsZ.omit({ connectivityPollFrequency: true }).extend({
    name: z
      .string()
      .min(1, { message: "Name is required" })
      .refine((n) => !names.includes(n), {
        message: "A cluster with this name already exists",
      }),
  });

  const methods = Form.use<typeof formSchema>({
    schema: formSchema,
    values: {
      name: "",
      host: "",
      port: "",
      username: "",
      password: "",
      secure: false,
    },
  });

  const handleSubmit = (): void => {
    void (async () => {
      if (!methods.validate()) return;
      const data = methods.value();
      setConnState(null);
      setLoading("submit");
      const state = await testConnection(data);
      setLoading(null);
      if (state.status !== "connected") return setConnState(state);
      dispatch(
        set({
          key: state.clusterKey,
          name: data.name,
          props: data as SynnaxProps,
        }),
      );
      dispatch(setActive(state.clusterKey));
      onClose();
    })();
  };

  const handleTestConnection = (): void => {
    void (async (): Promise<void> => {
      if (!methods.validate()) {
        return;
      }
      setConnState(null);
      setLoading("test");
      const state = await testConnection(methods.value() as SynnaxProps);
      setConnState(state);
      setLoading(null);
    })();
  };

  return (
    <Align.Space grow className={CSS.B("connect-cluster")}>
      <Form.Form {...methods}>
        <Align.Space className="console-form" grow size={0.5} justify="center">
          <Form.TextField
            path="name"
            inputProps={{
              autoFocus: true,
              variant: "natural",
              level: "h2",
              placeholder: "My Synnax Cluster",
            }}
          />
          <Align.Space direction="x">
            <Form.Field<string> path="host" grow>
              {(p) => <Input.Text placeholder="localhost" {...p} />}
            </Form.Field>
            <Form.Field<string> path="port" className={CSS.BE("input", "port")}>
              {(p) => <Input.Text placeholder="9090" {...p} />}
            </Form.Field>
          </Align.Space>
          <Form.Field<string> path="username">
            {(p) => <Input.Text placeholder="synnax" {...p} />}
          </Form.Field>
          <Align.Space direction="x">
            <Form.Field<string> path="password" className={CSS.BE("input", "password")}>
              {(p) => <Input.Text {...p} placeholder="seldon" type="password" />}
            </Form.Field>
            <Form.Field<boolean> path="secure">
              {componentRenderProp(Input.Switch)}
            </Form.Field>
          </Align.Space>
        </Align.Space>
      </Form.Form>
      <Layout.BottomNavBar>
        <Nav.Bar.Start size="small">
          {connState != null ? (
            <Status.Text variant={statusVariants[connState.status]}>
              {connState.status === "connected"
                ? caseconv.capitalize(connState.status)
                : connState.message}
            </Status.Text>
          ) : (
            <>
              <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
              <Text.Text shade={7} level="small">
                To Test Connection
              </Text.Text>
            </>
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Button.Button
            loading={loading === "test"}
            disabled={loading !== null}
            variant="text"
            onClick={handleTestConnection}
            triggers={[SAVE_TRIGGER]}
          >
            Test Connection
          </Button.Button>
          <Button.Button
            onClick={handleSubmit}
            loading={loading === "submit"}
            disabled={loading !== null}
          >
            Done
          </Button.Button>
        </Nav.Bar.End>
      </Layout.BottomNavBar>
    </Align.Space>
  );
};
