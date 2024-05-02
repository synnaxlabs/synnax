// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import { synnaxPropsZ } from "@synnaxlabs/client";
import type { connection, SynnaxProps } from "@synnaxlabs/client";
import { Nav, componentRenderProp, Status, Form } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Button } from "@synnaxlabs/pluto/button";
import { Input } from "@synnaxlabs/pluto/input";
import { caseconv } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { statusVariants } from "@/cluster/Badges";
import { useSelectMany } from "@/cluster/selectors";
import { setActive, set } from "@/cluster/slice";
import { testConnection } from "@/cluster/testConnection";
import { CSS } from "@/css";
import { type Layout } from "@/layout";

import "@/cluster/Connect.css";

export const connectWindowLayout: Layout.LayoutState = {
  key: "connectCluster",
  windowKey: "connectCluster",
  type: "connectCluster",
  name: "Connect Cluster",
  location: "window",
  window: {
    resizable: false,
    size: { height: 430, width: 650 },
    navTop: true,
  },
};

/**
 * ConnectCluster implements the LayoutRenderer component type to provide a form for
 * connecting to a cluster.
 *
 * @param props - The standard LayoutRendererProps.
 */
export const Connect = ({ onClose }: Layout.RendererProps): ReactElement => {
  const dispatch = useDispatch();
  const [connState, setConnState] = useState<connection.State | null>(null);
  const [loading, setLoading] = useState<"test" | "submit" | null>(null);

  const names = useSelectMany().map((c) => c.name);

  const formSchema = synnaxPropsZ.omit({ connectivityPollFrequency: true }).extend({
    name: z.string().refine((n) => !names.includes(n), {
      message: "A cluster with this name already exists",
    }),
  });

  const handleSubmit = (): void => {
    void (async () => {
      if (!methods.validate()) {
        console.log("Invalid form");
        return;
      }
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

  const methods = Form.use<typeof formSchema>({
    schema: formSchema,
    values: {
      name: "",
      host: "",
      port: 9090,
      username: "",
      password: "",
      secure: false,
    },
  });

  const handleTestConnection = (): void => {
    void (async (): Promise<void> => {
      if (!methods.validate()) {
        console.log("Invalid form");
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
        <Align.Space className="console-form" grow empty justify="center">
          <Form.Field<string> path="name">
            {(p) => <Input.Text placeholder="My Synnax Cluster" autoFocus {...p} />}
          </Form.Field>
          <Align.Space direction="x">
            <Form.Field<string> path="host" grow>
              {(p) => <Input.Text placeholder="localhost" {...p} />}
            </Form.Field>
            <Form.Field<string> path="port" className={CSS.BE("input", "port")}>
              {(p) => <Input.Text placeholder="9090" {...p} />}
            </Form.Field>
          </Align.Space>
          <Form.Field<string> path="username">
            {(p) => <Input.Text placeholder="Harry" {...p} />}
          </Form.Field>
          <Align.Space direction="x">
            <Form.Field<string> path="password" className={CSS.BE("input", "password")}>
              {(p) => <Input.Text {...p} placeholder="Seldon" type="password" />}
            </Form.Field>
            <Form.Field<boolean> path="secure">
              {componentRenderProp(Input.Switch)}
            </Form.Field>
          </Align.Space>
        </Align.Space>
      </Form.Form>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.Start className={CSS.BE("footer", "start")}>
          {connState != null && (
            <Status.Text variant={statusVariants[connState.status]}>
              {connState.status === "connected"
                ? caseconv.capitalize(connState.status)
                : connState.message}
            </Status.Text>
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End className={CSS.BE("footer", "end")}>
          <Button.Button
            loading={loading === "test"}
            disabled={loading !== null}
            variant="text"
            onClick={handleTestConnection}
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
      </Nav.Bar>
    </Align.Space>
  );
};
