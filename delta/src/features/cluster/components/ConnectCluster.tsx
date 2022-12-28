import { useState } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { synnaxPropsSchema } from "@synnaxlabs/client";
import type { SynnaxProps } from "@synnaxlabs/client";
import { Button, Header, Input, Nav, Space } from "@synnaxlabs/pluto";
import type { InputSwitchProps } from "@synnaxlabs/pluto";
import { FieldValues, useForm } from "react-hook-form";
import { AiFillApi } from "react-icons/ai";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { setActiveCluster, setCluster } from "../store";
import { testConnection } from "../util/testConnection";

import { ConnectionStateBadge } from "./ClusterBadges";

import type { ConnectionState } from "@/features/cluster";
import { LayoutRendererProps } from "@/features/layout";

import "./ConnectCluster.css";

const formSchema = synnaxPropsSchema.extend({ name: z.string() });

/**
 * ConnectCluster implements the LayoutRenderer component type to provide a form for
 * connecting to a cluster.
 *
 * @param props - The standard LayoutRendererProps.
 */
export const ConnectCluster = ({ onClose }: LayoutRendererProps): JSX.Element => {
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
    const { clusterKey, state } = await testConnection(data as SynnaxProps);
    if (state.status !== "connected") return setConnState(state);
    dispatch(
      setCluster({
        key: clusterKey as string,
        name,
        state,
        props: data as SynnaxProps,
      })
    );
    dispatch(setActiveCluster(clusterKey as string));
    onClose();
  });

  const handleTestConnection = (): void => {
    void (async (): Promise<void> => {
      const valid = await trigger();
      if (!valid) return;
      const { state } = await testConnection(getValues() as SynnaxProps);
      setConnState(state);
    })();
  };

  return (
    <Space grow>
      <Header level="h4" divided>
        <Header.Title startIcon={<AiFillApi />}>Connect a Cluster</Header.Title>
      </Header>
      <Space className="delta-form" grow>
        {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
        <form onSubmit={handleSubmit} id="connect-cluster">
          <Space>
            <Input.ItemC name="name" placeholder="My Synnax Cluster" control={c} />
            <Space direction="horizontal">
              <Input.ItemC name="host" placeholder="localhost" control={c} grow />
              <Input.ItemC
                name="port"
                type="number"
                placeholder="9090"
                control={c}
                className="delta-connect-cluster__input--port"
              />
            </Space>
            <Input.ItemC name="username" placeholder="Harry" control={c} />
            <Space direction="horizontal">
              <Input.ItemC
                name="password"
                placeholder="Seldon"
                type="password"
                control={c}
                className="delta-connect-cluster__input--password"
              />
              <Input.ItemC<boolean, InputSwitchProps> name="secure" control={c}>
                {Input.Switch}
              </Input.ItemC>
            </Space>
          </Space>
        </form>
      </Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.Start className="delta-connect-cluster-footer__left">
          {connState != null && <ConnectionStateBadge state={connState} />}
        </Nav.Bar.Start>
        <Nav.Bar.End className="delta-connect-cluster-footer__right">
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
