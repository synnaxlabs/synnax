import {
  ConfigureAnalogRead,
  configureAnalogReadLayout,
} from "@/hardware/ni/task/ConfigureAnalogRead";
import {
  ConfigureDigitalWrite,
  configureDigitalWriteLayout,
} from "@/hardware/ni/task/ConfigureDigitalWrite";
import { Layout } from "@/layout";

export * from "@/hardware/ni/task/ConfigureAnalogRead";
export * from "@/hardware/ni/task/ConfigureDigitalWrite";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [configureAnalogReadLayout.type]: ConfigureAnalogRead,
  [configureDigitalWriteLayout.type]: ConfigureDigitalWrite,
};
