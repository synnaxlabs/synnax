import { useState } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { synnaxPropsSchema } from "@synnaxlabs/client";
import type { Connectivity, SynnaxProps } from "@synnaxlabs/client";
import { Button, Header, Input, Nav, Space } from "@synnaxlabs/pluto";
import { FieldValues, useForm } from "react-hook-form";
import { AiFillApi } from "react-icons/ai";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { setActiveCluster, setCluster } from "../store";
import { testConnection } from "../util/testConnection";

import { ConnectionStateBadge } from "./ConnectionStateBadge";

import { ConnectionState } from "@/features/cluster/types";
import { LayoutRendererProps } from "@/features/layout";

import "./ConnectCluster.css";

const formSchema = synnaxPropsSchema.extend({
  name: z.string().optional(),
});

export interface ConnectClusterContentProps {
  clusterKey?: string;
}

export const ConnectCluster = ({ onClose }: LayoutRendererProps): JSX.Element => {
  const dispatch = useDispatch();
  const [connState, setConnState] = useState<ConnectionState | null>(null);

  const {
    getValues,
    trigger,
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(formSchema),
  });

  const _handleSubmit = (data: FieldValues): void => {
    void handleSubmit(async (): Promise<void> => {
      const name = data.name;
      delete data.name;
      data.secure = true;
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
    })();
  };

  const handleTestConnection = (): void => {
    void (async (): Promise<void> => {
      const ok = await trigger();
      if (!ok) return;
      console.log(getValues().secure);
      const { state } = await testConnection(getValues() as SynnaxProps);
      setConnState(state);
    })();
  };

  return (
    <Space direction="vertical" grow>
      <Header level="h4" icon={<AiFillApi />} divided>
        Connect a Cluster
      </Header>
      <Space className="connect-cluster__content" direction="vertical" grow>
        <form onSubmit={_handleSubmit} id="connect-cluster">
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
                className="connect-cluster__input__host"
                {...register("host")}
              />
              <Input.Item
                label="Port"
                type="number"
                placeholder="8080"
                helpText={errors.port?.message?.toString()}
                className="connect-cluster__input__port"
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
                className="connect-cluster__input__password"
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
        <Nav.Bar.Start style={{ padding: "0 2rem" }}>
          {connState != null && <ConnectionStateBadge state={connState} />}
        </Nav.Bar.Start>
        <Nav.Bar.End style={{ padding: "1rem" }}>
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
