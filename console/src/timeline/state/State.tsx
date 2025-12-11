import { Dialog, Flex, Form, Header, Icon, Text, Timeline } from "@synnaxlabs/pluto";
import { bounds } from "@synnaxlabs/x";

import { Menu } from "@/timeline/state/Menu";

export interface StateProps {
  itemKey: string;
  path: string;
  onDelete: (key: string) => void;
}

export const State = ({ itemKey, path, onDelete }: StateProps) => (
  <Flex.Box y pack style={{ borderTop: "var(--pluto-border)" }}>
    <Header.Header level="h5" background={1}>
      <Form.TextField
        path={`${path}.name`}
        showLabel={false}
        showHelpText={false}
        inputProps={{ level: "h5", variant: "text", textColor: 9 }}
      />
      <Dialog.Frame>
        <Dialog.Trigger hideCaret variant="text">
          <Icon.KebabMenu />
        </Dialog.Trigger>
        <Dialog.Dialog>
          <Menu itemKey={itemKey} path={path} onDelete={onDelete} />
        </Dialog.Dialog>
      </Dialog.Frame>
    </Header.Header>
    <Timeline.Frame
      style={{
        height: "300px",
        borderTop: "var(--pluto-border)",
        borderBottom: "var(--pluto-border)",
      }}
    >
      <Timeline.Track>
        <Timeline.Item itemKey={path} bounds={bounds.ZERO} onBoundsChange={() => {}}>
          <Text.Text level="p">Hello</Text.Text>
        </Timeline.Item>
      </Timeline.Track>
    </Timeline.Frame>
  </Flex.Box>
);
