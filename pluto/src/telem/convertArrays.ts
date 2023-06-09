import { DataType, LazyArray, SampleValue } from "@synnaxlabs/x";

export const convertArrays = (arrs: LazyArray[]): LazyArray[] =>
  arrs.map((a) => {
    let offset: SampleValue = 0;
    if (a.dataType.equals(DataType.TIMESTAMP)) offset = BigInt(-a.data[0]);
    return a.convert(DataType.FLOAT32, offset);
  });
