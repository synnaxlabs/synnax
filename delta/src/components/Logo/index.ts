import { Logo as CoreLogo } from "./Logo";
import { LogoWatermark } from "./LogoWatermark";

type CoreLogoType = typeof CoreLogo;

export interface LogoType extends CoreLogoType {
  Watermark: typeof LogoWatermark;
}

export const Logo = CoreLogo as LogoType;

Logo.Watermark = LogoWatermark;
