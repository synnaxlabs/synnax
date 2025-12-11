import { Dialog, Form, Header, Icon, Text, Timeline } from "@synnaxlabs/pluto";
import { bounds } from "@synnaxlabs/x";

import { Menu } from "@/timeline/phase/Menu";
import { State } from "@/timeline/state";

export interface PhaseProps {
  path: string;
  index: number;
  onDelete: (path: string) => void;
}

export const Phase = ({ path, index, onDelete }: PhaseProps) => (
  <Timeline.Item
    y
    bordered
    itemKey={path}
    bounds={bounds.ZERO}
    background={0}
    rounded
    pack
    onBoundsChange={() => {}}
  >
    <Header.Header level="h4">
      <Header.Title align="center">
        <Text.Text variant="code" color={8} level="h4">
          {index + 1}
        </Text.Text>
        <Form.TextField
          path={`${path}.name`}
          showLabel={false}
          showHelpText={false}
          inputProps={{ level: "h4", variant: "text", textColor: 10 }}
        />
      </Header.Title>
      <Dialog.Frame>
        <Dialog.Trigger hideCaret variant="text">
          <Icon.KebabMenu />
        </Dialog.Trigger>
        <Dialog.Dialog>
          <Menu path={path} onDelete={onDelete} />
        </Dialog.Dialog>
      </Dialog.Frame>
    </Header.Header>
    <State.States path={`${path}.states`} />
  </Timeline.Item>
);
