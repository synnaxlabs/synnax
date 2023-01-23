import { Text, Button, Space } from "@synnaxlabs/pluto";

import { Logo } from "@/components";
import { ClusterIcon } from "@/features/cluster";
import { VisCreateButton } from "@/features/vis";

import "./GetStarted.css";

export const GetStarted = (): JSX.Element => {
  return (
    <Space.Centered className="delta-get-started" align="center" size={6}>
      <Logo variant="title" className="delta-get-started__logo" />
      <Text level="h1">Get Started</Text>
      <Space direction="x" size="large" justify="center" wrap>
        <Button startIcon={<ClusterIcon />} size="large">
          Connect a Cluster
        </Button>
        <VisCreateButton size="large" />
      </Space>
      <Text.Link href="https://docs.synnaxlabs.com" target="_blank" level="h4">
        Read the Documentation
      </Text.Link>
    </Space.Centered>
  );
};
