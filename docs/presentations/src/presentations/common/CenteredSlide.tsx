import { Space } from "@synnaxlabs/pluto";
import { PropsWithChildren } from "react";

export default function CenteredSlide({children}: PropsWithChildren<{}>) {
  return (
    <Space direction="vertical" justify="center" align="center" size={20} grow>
      {children}
    </Space>

  )
}