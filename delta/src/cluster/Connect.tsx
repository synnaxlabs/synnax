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
} from "@synnaxlabs/pluto";
import { type FieldValues, useForm } from "react-hook-form";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { ConnectionStateBadge } from "@/cluster/Badges";
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

  const {
    getValues,
    trigger,
    control: c,
    handleSubmit: _handleSubmit,
  } = useForm({ resolver: zodResolver(formSchema) });

  const handleSubmit = _handleSubmit(async (_data: FieldValues): Promise<void> => {
    const { name, ...data } = _data;
    const state = await testConnection(data as SynnaxProps);
    if (state.status !== "connected") return setConnState(state);
    dispatch(
      set({
        key: state.clusterKey,
        name,
        props: data as SynnaxProps,
      })
    );
    dispatch(setActive(state.clusterKey));
    onClose();
  });

  const handleTestConnection = (): void => {
    void (async (): Promise<void> => {
      const valid = await trigger();
      if (!valid) return;
      const state = await testConnection(getValues() as SynnaxProps);
      setConnState(state);
    })();
  };

  return (
    <Align.Space grow className={CSS.B("connect-cluster")}>
      <Header.Header level="h4">
        <Header.Title startIcon={<Icon.Cluster />}>Connect a Cluster</Header.Title>
      </Header.Header>
      <Align.Space className="delta-form" grow>
        {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
        <form onSubmit={handleSubmit} id="connect-cluster">
          <Align.Space>
            <Input.ItemControlled
              name="name"
              placeholder="My Synnax Cluster"
              control={c}
              autoFocus
            />
            <Align.Space direction="x">
              <Input.ItemControlled
                name="host"
                placeholder="localhost"
                control={c}
                grow
              />
              <Input.ItemControlled
                name="port"
                type="number"
                placeholder="9090"
                control={c}
                className={CSS.BE("input", "port")}
              />
            </Align.Space>
            <Input.ItemControlled name="username" placeholder="Harry" control={c} />
            <Align.Space direction="x">
              <Input.ItemControlled
                name="password"
                placeholder="Seldon"
                type="password"
                control={c}
                className={CSS.BE("input", "password")}
              />
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
          {connState != null && <ConnectionStateBadge state={connState} />}
        </Nav.Bar.Start>
        <Nav.Bar.End className={CSS.BE("footer", "end")}>
          <Button.Button variant="text" onClick={handleTestConnection}>
            Test Connection
          </Button.Button>
          <Button.Button type="submit" form="connect-cluster">
            Done
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};
