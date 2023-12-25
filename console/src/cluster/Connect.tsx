// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { synnaxPropsZ } from "@synnaxlabs/client";
import type { connection, SynnaxProps } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Button,
  Header,
  Input,
  Nav,
  Align,
  componentRenderProp,
  Status,
} from "@synnaxlabs/pluto";
import { Case } from "@synnaxlabs/x";
import { type FieldValues, useForm } from "react-hook-form";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { statusVariants } from "@/cluster/Badges";
import { setActive, set } from "@/cluster/slice";
import { testConnection } from "@/cluster/testConnection";
import { CSS } from "@/css";
import { type Layout } from "@/layout";

import "@/cluster/Connect.css";

const formSchema = synnaxPropsZ.extend({ name: z.string() });

export const connectWindowLayout: Layout.LayoutState = {
  key: "connectCluster",
  windowKey: "connectCluster",
  type: "connectCluster",
  name: "Connect a Cluster",
  location: "window",
  window: {
    resizable: false,
    size: { height: 430, width: 650 },
    navTop: true,
    transparent: true,
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

  const {
    getValues,
    trigger,
    control: c,
    handleSubmit: _handleSubmit,
  } = useForm({ resolver: zodResolver(formSchema) });

  const handleSubmit = _handleSubmit(async (_data: FieldValues): Promise<void> => {
    const { name, ...data } = _data;
    setConnState(null);
    setLoading("submit");
    const state = await testConnection(data as SynnaxProps);
    setLoading(null);
    if (state.status !== "connected") return setConnState(state);
    dispatch(
      set({
        key: state.clusterKey,
        name,
        props: data as SynnaxProps,
      }),
    );
    dispatch(setActive(state.clusterKey));
    onClose();
  });

  const handleTestConnection = (): void => {
    void (async (): Promise<void> => {
      const valid = await trigger();
      if (!valid) return;
      setConnState(null);
      setLoading("test");
      const state = await testConnection(getValues() as SynnaxProps);
      setConnState(state);
      setLoading(null);
    })();
  };

  return (
    <Align.Space grow className={CSS.B("connect-cluster")}>
      <Header.Header level="h4">
        <Header.Title startIcon={<Icon.Cluster />}>Connect a Cluster</Header.Title>
      </Header.Header>
      <Align.Space className="console-form" grow>
        {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
        <form onSubmit={handleSubmit} id="connect-cluster">
          <Align.Space>
            <Input.ItemControlled
              name="name"
              placeholder="My Synnax Cluster"
              control={c}
              autoFocus
            >
              {(p) => <Input.Text {...p} />}
            </Input.ItemControlled>
            <Align.Space direction="x" grow>
              <Input.ItemControlled
                name="host"
                placeholder="localhost"
                control={c}
                grow
              >
                {(p) => <Input.Text {...p} />}
              </Input.ItemControlled>
              <Input.ItemControlled
                name="port"
                type="number"
                placeholder="9090"
                control={c}
                className={CSS.BE("input", "port")}
                grow
              >
                {(p) => <Input.Text {...p} />}
              </Input.ItemControlled>
            </Align.Space>
            <Input.ItemControlled name="username" placeholder="Harry" control={c}>
              {(p) => <Input.Text {...p} />}
            </Input.ItemControlled>
            <Align.Space direction="x">
              <Input.ItemControlled
                name="password"
                placeholder="Seldon"
                control={c}
                className={CSS.BE("input", "password")}
              >
                {(p) => <Input.Text {...p} type="password" />}
              </Input.ItemControlled>
              <Input.ItemControlled<boolean, boolean, Input.SwitchProps>
                name="secure"
                control={c}
              >
                {componentRenderProp(Input.Switch)}
              </Input.ItemControlled>
            </Align.Space>
          </Align.Space>
        </form>
      </Align.Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.Start className={CSS.BE("footer", "start")}>
          {connState != null && (
            <Status.Text variant={statusVariants[connState.status]}>
              {connState.status === "connected"
                ? Case.capitalize(connState.status)
                : connState.message!}
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
            type="submit"
            form="connect-cluster"
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
