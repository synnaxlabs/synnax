import { Align, Synnax, Text } from "@synnaxlabs/pluto";
import {
  type MouseEventHandler,
  type PropsWithChildren,
  type ReactElement,
} from "react";

import { connectWindowLayout } from "@/cluster/remote/Connect";
import { Layout } from "@/layout";

export interface NoneConnectedProps extends PropsWithChildren {}

export const Boundary = ({ children }: NoneConnectedProps): ReactElement => {
  const client = Synnax.use();
  if (client != null) return <>{children}</>;
  return <NoneConnected />;
};

export const NoneConnected = (): ReactElement => {
  const placer = Layout.usePlacer();

  const handleCluster: MouseEventHandler<HTMLParagraphElement> = (e) => {
    e.stopPropagation();
    placer(connectWindowLayout);
  };

  return (
    <Align.Space empty style={{ height: "100%", position: "relative" }}>
      <Align.Center direction="y" style={{ height: "100%" }} size="small">
        <Text.Text level="p">No cluster connected.</Text.Text>
        <Text.Link level="p" onClick={handleCluster}>
          Connect a cluster
        </Text.Link>
      </Align.Center>
    </Align.Space>
  );
};
