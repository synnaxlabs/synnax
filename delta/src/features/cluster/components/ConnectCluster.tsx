import { useState } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { synnaxPropsSchema } from "@synnaxlabs/client";
import type { SynnaxProps } from "@synnaxlabs/client";
import { Button, Header, Input, Nav, Space } from "@synnaxlabs/pluto";
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

const formSchema = synnaxPropsSchema.extend({ name: z.string().optional() });

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
    register,
    handleSubmit: _handleSubmit,
    formState: { errors },
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
    <Space direction="vertical" grow>
      <Header level="h4" icon={<AiFillApi />} divided>
        Connect a Cluster
      </Header>
      <Space className="delta-form" direction="vertical" grow>
        {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
        <form onSubmit={handleSubmit} id="connect-cluster">
          <Space direction="vertical">
            <Input.Item
              label="Name"
              placeholder="My Synnax Cluster"
              {...register("name")}
            />
            <Space direction="horizontal">
              <Input.Item
                label="Host"
                placeholder="localhost"
                helpText={errors.host?.message?.toString()}
                className="delta-connect-cluster__input--host"
                {...register("host")}
              />
              <Input.Item
                label="Port"
                type="number"
                placeholder="9090"
                helpText={errors.port?.message?.toString()}
                className="delta-connect-cluster__input--port"
                {...register("port")}
              />
            </Space>
            <Input.Item
              label="Username"
              placeholder="Harry"
              helpText={errors.username?.message?.toString()}
              {...register("username")}
            />
            <Space direction="horizontal">
              <Input.Item
                label="Password"
                placeholder="Seldon"
                type="password"
                helpText={errors.password?.message?.toString()}
                className="delta-connect-cluster__input--password"
                {...register("password")}
              />
              <Input.Item label="Secure" {...register("secure")}>
                {Input.Switch}
              </Input.Item>
            </Space>
          </Space>
        </form>
      </Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.Start className="delta-connect-cluster-footer__left">
          {connState != null && <ConnectionStateBadge state={connState} />}
        </Nav.Bar.Start>
        <Nav.Bar.End className="delta-connect-cluster-footer__right">
          <Button variant="text" size="medium" onClick={handleTestConnection}>
            Test Connection
          </Button>
          <Button variant="filled" type="submit" form="connect-cluster">
            Done
          </Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Space>
  );
};
