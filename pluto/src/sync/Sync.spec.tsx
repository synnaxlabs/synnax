import { TEST_CLIENT_CONNECTION_PROPS } from "@synnaxlabs/client";
import { createMockWorkers } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import {
  type FC,
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useState,
} from "react";
import { describe, expect, it } from "vitest";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { type AetherMessage, type MainMessage } from "@/aether/message";
import { Status } from "@/status";
import { status } from "@/status/aether";
import { Sync } from "@/sync";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";

const createAetherProvider = (): FC<PropsWithChildren> => {
  const [a, b] = createMockWorkers();
  aether.render({
    comms: a.route("test"),
    registry: { ...synnax.REGISTRY, ...status.REGISTRY },
  });
  const worker = b.route<MainMessage, AetherMessage>("test");
  const AetherProvider = (props: PropsWithChildren): ReactElement => (
    <Aether.Provider {...props} worker={worker} workerKey="test" />
  );
  return AetherProvider;
};

const AetherProvider = createAetherProvider();

const Provider = (props: PropsWithChildren): ReactElement => {
  const [isConnected, setIsConnected] = useState(false);
  const handleConnect = () => setIsConnected((state) => !state);
  return (
    <>
      <button id="connect" onClick={handleConnect} />
      <AetherProvider>
        <Status.Aggregator>
          <Synnax.Provider
            {...props}
            connParams={isConnected ? TEST_CLIENT_CONNECTION_PROPS : undefined}
          />
        </Status.Aggregator>
      </AetherProvider>
    </>
  );
};

const test_channel_name = "sync_test_channel";

const Component = () => {
  const [data, setData] = useState<string[]>([]);
  const addListener = Sync.useAddListener();
  useEffect(() => {
    addListener({
      channels: test_channel_name,
      handler: (frame) => {
        setData((state) => [
          ...state,
          ...frame.series.flatMap((s) => s.toStrings()).flat(),
        ]);
      },
    });
  }, [addListener]);
  return <div>{data.join(", ")}</div>;
};

describe("sync", () => {
  it("should render", () => {
    render(<Provider />);
  });
  it("should add a basic listener", () => {
    render(<Provider />);
    const button = fireEvent.click(button);
    const component = render(<Component />);
    expect(component).toBeDefined();
  });
});
