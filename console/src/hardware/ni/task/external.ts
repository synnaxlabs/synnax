import {
  ConfigureAnalogRead,
  configureAnalogReadLayout,
} from "@/hardware/ni/task/ConfigureAnalogRead";
import {
  ConfigureDigitalRead,
  configureDigitalReadLayout,
} from "@/hardware/ni/task/ConfigureDigitalRead";
import {
  ConfigureDigitalWrite,
  configureDigitalWriteLayout,
} from "@/hardware/ni/task/ConfigureDigitalWrite";
import { Layout } from "@/layout";

export * from "@/hardware/ni/task/ConfigureAnalogRead";
export * from "@/hardware/ni/task/ConfigureDigitalWrite";
export * from "@/hardware/ni/task/ConfigureDigitalRead";
export * from "@/hardware/ni/task/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [configureAnalogReadLayout.type]: ConfigureAnalogRead,
  [configureDigitalWriteLayout.type]: ConfigureDigitalWrite,
  [configureDigitalReadLayout.type]: ConfigureDigitalRead,
};
