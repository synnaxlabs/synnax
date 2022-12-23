import { Space } from "@synnaxlabs/pluto";

import { Logo, LogoProps } from "./Logo";

/**
 * LogoWatermark displays the Synnax logo as a watermark in the center of the
 * screen.
 *
 * @param props - The same props as Logo.
 */
export const LogoWatermark = (props: LogoProps): JSX.Element => (
  <Space.Centered>
    <Logo className="delta-logo-watermark" {...props} />
  </Space.Centered>
);
