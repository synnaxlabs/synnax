import { UnparsedDataType, UnparsedRate } from '../telem';
import Transport from '../transport';
import { ChannelPayload } from './payload';
export declare type CreateChannelProps = {
    rate: UnparsedRate;
    dataType: UnparsedDataType;
    name?: string;
    nodeId?: number;
};
export default class Creator {
    private static ENDPOINT;
    private client;
    constructor(transport: Transport);
    create(props: CreateChannelProps): Promise<ChannelPayload>;
    createMany({ rate, dataType, name, nodeId, count, }: CreateChannelProps & {
        count: number;
    }): Promise<ChannelPayload[]>;
    private execute;
}
