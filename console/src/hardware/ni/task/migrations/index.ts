import { migrate } from "@synnaxlabs/x";

export * from "@/hardware/ni/task/migrations/v1";
import * as v0 from "@/hardware/ni/task/migrations/v0";
import * as v1 from "@/hardware/ni/task/migrations/v1";

export const ANALOG_READ_CONFIG_MIGRATIONS: migrate.Migrations = {
  "0.0.0": v1.analogReadTaskMigration,
};

type AnyAnalogReadConfig = v0.AnalogReadTaskConfig | v1.AnalogReadTaskConfig;

export const migrateAnalogReadTask = migrate.migrator<
  AnyAnalogReadConfig,
  v1.AnalogReadTaskConfig
>({
  name: "hardware.ni.analogread",
  migrations: ANALOG_READ_CONFIG_MIGRATIONS,
  def: v1.ZERO_ANALOG_READ_CONFIG,
  defaultVersion: "0.0.0",
});
