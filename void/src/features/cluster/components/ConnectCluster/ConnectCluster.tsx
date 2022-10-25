import CoreWindow from "../../../../components/Windows/CoreWindow";
import { createWindow } from "@synnaxlabs/drift";
import { Button, Header, Input, Nav, Space } from "@synnaxlabs/pluto";
import { AiFillApi } from "react-icons/ai";
import { FieldValues, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Connectivity,
  Synnax,
  SynnaxProps,
  synnaxPropsSchema,
} from "@synnaxlabs/client";
import "./ConnectCluster.css";
import { useEffect, useState } from "react";
import { useDispatch } from "react-redux";
import { closeWindow } from "@synnaxlabs/drift";
import { z } from "zod";
import { useParams } from "react-router-dom";

import { ConnectionState } from "@/features/cluster/types";
import { testConnection } from "../../util/testConnection";
import { setCluster, useSelectCluster } from "../../store";
import { ConnectionStatus } from "../ConnectionStatus/ConnectionStatus";

const formSchema = synnaxPropsSchema.extend({
  name: z.string().optional(),
});

export default function ConnectCluster() {
  const dispatch = useDispatch();
  const [key, setKey] = useState("");

  const cluster = useSelectCluster(key);

  const [testConnState, setConnState] = useState<ConnectionState | undefined>(
    undefined
  );

  const {
    getValues,
    trigger,
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(formSchema),
    defaultValues: { name: cluster?.name, ...cluster?.props },
  });

  useEffect(() => {
    if (paramsKey) setKey(paramsKey);
  }, [paramsKey]);

  const onSubmit = async (data: FieldValues) => {
    const name = data.name;
    delete data.name;
    const { clusterKey, state: connState } = await testConnection(
      data as SynnaxProps
    );
    if (connState.status !== Connectivity.CONNECTED) {
      setConnState(connState);
      return;
    }
    dispatch(
      setCluster({
        active: true,
        key: clusterKey as string,
        name: name,
        state: connState,
        props: data as SynnaxProps,
      })
    );
    dispatch(closeWindow());
  };

  const onTestConnection = async () => {
    const ok = await trigger();
    if (!ok) return;
    const { state } = await testConnection(getValues() as SynnaxProps);
    setConnState(state);
  };

  return (
    <CoreWindow>
      <Header level="h4" icon={<AiFillApi />} divided>
        Connect a Cluster
      </Header>
      <Space className="connect-cluster__content" direction="vertical" grow>
        <form onSubmit={handleSubmit(onSubmit)} id="my-form">
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
            <Input.Item
              label="Password"
              placeholder="Seldon"
              type="password"
              helpText={errors.password?.message?.toString()}
              {...register("password")}
            />
          </Space>
        </form>
      </Space>
      <Nav.Bar location="bottom" size={48} style={{ flexShrink: 0 }}>
        <Nav.Bar.Start style={{ padding: "0 12px" }}>
          {testConnState && <ConnectionStatus state={testConnState} />}
        </Nav.Bar.Start>
        <Nav.Bar.End style={{ padding: 6 }}>
          <Button variant="text" size="medium" onClick={onTestConnection}>
            Test Connection
          </Button>
          <Button variant="filled" type="submit" form="my-form">
            Done
          </Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </CoreWindow>
  );
}

export const createConnectClusterWindow = (key?: string) => {
  return createWindow({
    url: `http://localhost:5173/cluster/connect/${key}`,
    resizable: false,
    height: 425,
    width: 650,
    title: "Connect a cluster",
  });
};
