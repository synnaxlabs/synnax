import { z } from 'zod';
import { DataType, Rate } from '../telem';
import { channelPayloadSchema } from './payload';
const RequestSchema = z.object({
    channel: channelPayloadSchema,
    count: z.number(),
});
const ResponseSchema = z.object({
    channels: channelPayloadSchema.array(),
});
export default class Creator {
    static ENDPOINT = '/channel/create';
    client;
    constructor(transport) {
        this.client = transport.postClient();
    }
    async create(props) {
        const [channel] = await this.createMany({ ...props, count: 1 });
        return channel;
    }
    async createMany({ rate, dataType, name = '', nodeId = 0, count = 1, }) {
        return (await this.execute({
            channel: {
                name,
                nodeId,
                rate: new Rate(rate),
                dataType: new DataType(dataType),
            },
            count,
        })).channels;
    }
    async execute(request) {
        const [res, err] = await this.client.send(Creator.ENDPOINT, request, 
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        ResponseSchema);
        if (err)
            throw err;
        return res;
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY3JlYXRvci5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uLy4uL3NyYy9saWIvY2hhbm5lbC9jcmVhdG9yLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUNBLE9BQU8sRUFBRSxDQUFDLEVBQUUsTUFBTSxLQUFLLENBQUM7QUFFeEIsT0FBTyxFQUFFLFFBQVEsRUFBRSxJQUFJLEVBQWtDLE1BQU0sVUFBVSxDQUFDO0FBRzFFLE9BQU8sRUFBa0Isb0JBQW9CLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFFakUsTUFBTSxhQUFhLEdBQUcsQ0FBQyxDQUFDLE1BQU0sQ0FBQztJQUM3QixPQUFPLEVBQUUsb0JBQW9CO0lBQzdCLEtBQUssRUFBRSxDQUFDLENBQUMsTUFBTSxFQUFFO0NBQ2xCLENBQUMsQ0FBQztBQUlILE1BQU0sY0FBYyxHQUFHLENBQUMsQ0FBQyxNQUFNLENBQUM7SUFDOUIsUUFBUSxFQUFFLG9CQUFvQixDQUFDLEtBQUssRUFBRTtDQUN2QyxDQUFDLENBQUM7QUFXSCxNQUFNLENBQUMsT0FBTyxPQUFPLE9BQU87SUFDbEIsTUFBTSxDQUFDLFFBQVEsR0FBRyxpQkFBaUIsQ0FBQztJQUNwQyxNQUFNLENBQWM7SUFFNUIsWUFBWSxTQUFvQjtRQUM5QixJQUFJLENBQUMsTUFBTSxHQUFHLFNBQVMsQ0FBQyxVQUFVLEVBQUUsQ0FBQztJQUN2QyxDQUFDO0lBRUQsS0FBSyxDQUFDLE1BQU0sQ0FBQyxLQUF5QjtRQUNwQyxNQUFNLENBQUMsT0FBTyxDQUFDLEdBQUcsTUFBTSxJQUFJLENBQUMsVUFBVSxDQUFDLEVBQUUsR0FBRyxLQUFLLEVBQUUsS0FBSyxFQUFFLENBQUMsRUFBRSxDQUFDLENBQUM7UUFDaEUsT0FBTyxPQUFPLENBQUM7SUFDakIsQ0FBQztJQUVELEtBQUssQ0FBQyxVQUFVLENBQUMsRUFDZixJQUFJLEVBQ0osUUFBUSxFQUNSLElBQUksR0FBRyxFQUFFLEVBQ1QsTUFBTSxHQUFHLENBQUMsRUFDVixLQUFLLEdBQUcsQ0FBQyxHQUM4QjtRQUN2QyxPQUFPLENBQ0wsTUFBTSxJQUFJLENBQUMsT0FBTyxDQUFDO1lBQ2pCLE9BQU8sRUFBRTtnQkFDUCxJQUFJO2dCQUNKLE1BQU07Z0JBQ04sSUFBSSxFQUFFLElBQUksSUFBSSxDQUFDLElBQUksQ0FBQztnQkFDcEIsUUFBUSxFQUFFLElBQUksUUFBUSxDQUFDLFFBQVEsQ0FBQzthQUNqQztZQUNELEtBQUs7U0FDTixDQUFDLENBQ0gsQ0FBQyxRQUFRLENBQUM7SUFDYixDQUFDO0lBRU8sS0FBSyxDQUFDLE9BQU8sQ0FBQyxPQUFnQjtRQUNwQyxNQUFNLENBQUMsR0FBRyxFQUFFLEdBQUcsQ0FBQyxHQUFHLE1BQU0sSUFBSSxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQ3ZDLE9BQU8sQ0FBQyxRQUFRLEVBQ2hCLE9BQU87UUFDUCw2REFBNkQ7UUFDN0QsYUFBYTtRQUNiLGNBQWMsQ0FDZixDQUFDO1FBQ0YsSUFBSSxHQUFHO1lBQUUsTUFBTSxHQUFHLENBQUM7UUFDbkIsT0FBTyxHQUFlLENBQUM7SUFDekIsQ0FBQyJ9