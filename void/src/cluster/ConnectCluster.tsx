import CoreWindow from "../lib/Windows/CoreWindow";
import { Dispatch, AnyAction } from "@reduxjs/toolkit";
import { createWindow } from "@synnaxlabs/drift";
import { Button, Header, Input, Nav, Space } from "@synnaxlabs/pluto";
import { AiFillApi, AiOutlineCheck } from "react-icons/ai";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import "./ConnectCluster.css";

const connParams = z.object({
  host: z.string().min(1),
  port: z.string(),
  username: z.string(),
  password: z.string(),
});

export default function ConnectCluster() {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({ resolver: zodResolver(connParams) });
  return (
    <CoreWindow>
      <Header level="h4" icon={<AiFillApi />} divided>
        Connect a Cluster
      </Header>
      <Space className="connect-cluster__content" direction="vertical" grow>
        <form onSubmit={handleSubmit(() => {})} id="my-form">
          <Space direction="vertical">
            <Space direction="horizontal">
              <Input.Item
                label="Host"
                placeholder="localhost"
                helpText={errors.host?.message?.toString()}
                style={{ flexGrow: 1 }}
                {...register("host")}
              />
              <Input.Item
                label="Port"
                type="number"
                placeholder="8080"
                helpText={errors.port?.message?.toString()}
                style={{
                  width: 100,
                }}
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
        <Nav.Bar.End style={{ padding: 6 }}>
          <Button
            type="submit"
            size="medium"
            startIcon={<AiOutlineCheck />}
            form="my-form"
          >
            Connect
          </Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </CoreWindow>
  );
}

export const createConnectClusterWindow = (dispatch: Dispatch<AnyAction>) => {
  dispatch(
    createWindow({
      url: "http://localhost:5173/cluster/connect",
      resizable: false,
      height: 425,
      width: 550,
      title: "Connect a cluster",
    })
  );
};
