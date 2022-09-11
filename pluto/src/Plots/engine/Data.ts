export interface DataProvider {
  read(channel: string, timeRange: [number, number]): number[];
}
