import { Flex, Text } from "@synnaxlabs/pluto";

export interface EmptyActionProps
  extends Omit<Flex.BoxProps<"div">, "onClick">,
    Pick<Text.TextProps, "onClick"> {
  message: string;
  action: string;
}

export const EmptyAction = ({
  message,
  action,
  onClick,
  direction,
  x,
  y = true,
  ...rest
}: EmptyActionProps) => (
  <Flex.Box center {...rest}>
    <Text.Text y={y} x={x} center color={9} direction={direction} gap="tiny">
      {message}
      <Text.Text onClick={onClick} variant="link">
        {action}
      </Text.Text>
    </Text.Text>
  </Flex.Box>
);
