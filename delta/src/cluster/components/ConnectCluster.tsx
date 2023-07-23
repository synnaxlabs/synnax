// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useState } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { synnaxPropsZ } from "@synnaxlabs/client";
import type { ConnectionState, SynnaxProps } from "@synnaxlabs/client";
import {
  Button,
  Header,
  Input,
  Nav,
  Space,
  componentRenderProp,
} from "@synnaxlabs/pluto";
import type { InputSwitchProps } from "@synnaxlabs/pluto";
import { nanoid } from "nanoid";
import { FieldValues, useForm } from "react-hook-form";
import { AiFillApi } from "react-icons/ai";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { ConnectionStateBadge } from "@/cluster/components/ClusterBadges";
import { setActiveCluster, setCluster } from "@/cluster/store";
import { testConnection } from "@/cluster/util/testConnection";
import { CSS } from "@/css";
import { LayoutState, LayoutRendererProps } from "@/layout";

import "@/cluster/components/ConnectCluster.css";

const formSchema = synnaxPropsZ.extend({ name: z.string() });

export const connectClusterWindowLayout: LayoutState = {
  key: "connectCluster",
  type: "connectCluster",
  name: "Connect a Cluster",
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
export const ConnectCluster = ({ onClose }: LayoutRendererProps): ReactElement => {
  const dispatch = useDispatch();
  const [connState, setConnState] = useState<ConnectionState | null>(null);

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
      setCluster({
        key: state.clusterKey,
        name,
        state,
        props: data as SynnaxProps,
      })
    );
    dispatch(setActiveCluster(state.clusterKey as string));
    onClose();
  });

  const handleTestConnection = (): void => {
    void (async (): Promise<void> => {
      const valid = await trigger();
      if (!valid) return;
      const state = await testConnection(getValues() as SynnaxProps);
      setConnState(state.className);
    })();
  };

  return (
    <Space grow className={CSS.B("connect-cluster")}>
      <Header level="h4" divided>
        <Header.Title startIcon={<AiFillApi />}>Connect a Cluster</Header.Title>
      </Header>
      <Space className="delta-form" grow>
        {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
        <form onSubmit={handleSubmit} id="connect-cluster">
          <Space>
            <Input.ItemC
              name="name"
              placeholder="My Synnax Cluster"
              control={c}
              autoFocus
            />
            <Space direction="x">
              <Input.ItemC name="host" placeholder="localhost" control={c} grow />
              <Input.ItemC
                name="port"
                type="number"
                placeholder="9090"
                control={c}
                className={CSS.BE("input", "port")}
              />
            </Space>
            <Input.ItemC name="username" placeholder="Harry" control={c} />
            <Space direction="x">
              <Input.ItemC
                name="password"
                placeholder="Seldon"
                type="password"
                control={c}
                className={CSS.BE("input", "password")}
              />
              <Input.ItemC<boolean, boolean, InputSwitchProps>
                name="secure"
                control={c}
              >
                {componentRenderProp(Input.Switch)}
              </Input.ItemC>
            </Space>
          </Space>
        </form>
      </Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.Start className={CSS.BE("footer", "start")}>
          {connState != null && <ConnectionStateBadge state={connState} />}
        </Nav.Bar.Start>
        <Nav.Bar.End className={CSS.BE("footer", "end")}>
          <Button variant="text" onClick={handleTestConnection}>
            Test Connection
          </Button>
          <Button type="submit" form="connect-cluster">
            Done
          </Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Space>
  );
};
