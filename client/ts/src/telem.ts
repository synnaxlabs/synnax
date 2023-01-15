import { registerCustomTypeEncoder } from "@synnaxlabs/freighter";
import { DataType, Density, Rate, TimeSpan, TimeStamp } from "@synnaxlabs/x";

const valueOfEncoder = (value: unknown): unknown => value?.valueOf();

registerCustomTypeEncoder({ Class: TimeStamp, write: valueOfEncoder });
registerCustomTypeEncoder({ Class: TimeSpan, write: valueOfEncoder });
registerCustomTypeEncoder({
  Class: DataType,
  write: (v: unknown) => (v as DataType).toString,
});
registerCustomTypeEncoder({ Class: Rate, write: valueOfEncoder });
registerCustomTypeEncoder({ Class: Density, write: valueOfEncoder });
