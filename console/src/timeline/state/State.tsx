import { Flex, Form, Header, Text, Timeline } from "@synnaxlabs/pluto";

export interface StateProps {
  path: string;
  onDelete: (key: string) => void;
}

export const State = ({ path }: StateProps) => (
  <Flex.Box y pack style={{ borderTop: "var(--pluto-border)" }}>
    <Header.Header level="h5" background={1}>
      <Form.TextField
        path={`${path}.name`}
        showLabel={false}
        showHelpText={false}
        inputProps={{ level: "h5", variant: "text", textColor: 9 }}
      />
    </Header.Header>
    <Timeline.Frame
      style={{
        height: "300px",
        borderTop: "var(--pluto-border)",
        borderBottom: "var(--pluto-border)",
      }}
    >
      <Timeline.Track>
        <Timeline.Item>
          <Text.Text level="p">Hello</Text.Text>
        </Timeline.Item>
      </Timeline.Track>
    </Timeline.Frame>
  </Flex.Box>
);
