"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const zod_1 = require("zod");
const telem_1 = require("../telem");
const payload_1 = require("./payload");
const RequestSchema = zod_1.z.object({
    channel: payload_1.channelPayloadSchema,
    count: zod_1.z.number(),
});
const ResponseSchema = zod_1.z.object({
    channels: payload_1.channelPayloadSchema.array(),
});
class Creator {
    constructor(transport) {
        this.client = transport.postClient();
    }
    async create(props) {
        const [channel] = await this.createMany(Object.assign(Object.assign({}, props), { count: 1 }));
        return channel;
    }
    async createMany({ rate, dataType, name = '', nodeId = 0, count = 1, }) {
        return (await this.execute({
            channel: {
                name,
                nodeId,
                rate: new telem_1.Rate(rate),
                dataType: new telem_1.DataType(dataType),
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
exports.default = Creator;
Creator.ENDPOINT = '/channel/create';
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY3JlYXRvci5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uLy4uL3NyYy9saWIvY2hhbm5lbC9jcmVhdG9yLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiI7O0FBQ0EsNkJBQXdCO0FBRXhCLG9DQUEwRTtBQUcxRSx1Q0FBaUU7QUFFakUsTUFBTSxhQUFhLEdBQUcsT0FBQyxDQUFDLE1BQU0sQ0FBQztJQUM3QixPQUFPLEVBQUUsOEJBQW9CO0lBQzdCLEtBQUssRUFBRSxPQUFDLENBQUMsTUFBTSxFQUFFO0NBQ2xCLENBQUMsQ0FBQztBQUlILE1BQU0sY0FBYyxHQUFHLE9BQUMsQ0FBQyxNQUFNLENBQUM7SUFDOUIsUUFBUSxFQUFFLDhCQUFvQixDQUFDLEtBQUssRUFBRTtDQUN2QyxDQUFDLENBQUM7QUFXSCxNQUFxQixPQUFPO0lBSTFCLFlBQVksU0FBb0I7UUFDOUIsSUFBSSxDQUFDLE1BQU0sR0FBRyxTQUFTLENBQUMsVUFBVSxFQUFFLENBQUM7SUFDdkMsQ0FBQztJQUVELEtBQUssQ0FBQyxNQUFNLENBQUMsS0FBeUI7UUFDcEMsTUFBTSxDQUFDLE9BQU8sQ0FBQyxHQUFHLE1BQU0sSUFBSSxDQUFDLFVBQVUsaUNBQU0sS0FBSyxLQUFFLEtBQUssRUFBRSxDQUFDLElBQUcsQ0FBQztRQUNoRSxPQUFPLE9BQU8sQ0FBQztJQUNqQixDQUFDO0lBRUQsS0FBSyxDQUFDLFVBQVUsQ0FBQyxFQUNmLElBQUksRUFDSixRQUFRLEVBQ1IsSUFBSSxHQUFHLEVBQUUsRUFDVCxNQUFNLEdBQUcsQ0FBQyxFQUNWLEtBQUssR0FBRyxDQUFDLEdBQzhCO1FBQ3ZDLE9BQU8sQ0FDTCxNQUFNLElBQUksQ0FBQyxPQUFPLENBQUM7WUFDakIsT0FBTyxFQUFFO2dCQUNQLElBQUk7Z0JBQ0osTUFBTTtnQkFDTixJQUFJLEVBQUUsSUFBSSxZQUFJLENBQUMsSUFBSSxDQUFDO2dCQUNwQixRQUFRLEVBQUUsSUFBSSxnQkFBUSxDQUFDLFFBQVEsQ0FBQzthQUNqQztZQUNELEtBQUs7U0FDTixDQUFDLENBQ0gsQ0FBQyxRQUFRLENBQUM7SUFDYixDQUFDO0lBRU8sS0FBSyxDQUFDLE9BQU8sQ0FBQyxPQUFnQjtRQUNwQyxNQUFNLENBQUMsR0FBRyxFQUFFLEdBQUcsQ0FBQyxHQUFHLE1BQU0sSUFBSSxDQUFDLE1BQU0sQ0FBQyxJQUFJLENBQ3ZDLE9BQU8sQ0FBQyxRQUFRLEVBQ2hCLE9BQU87UUFDUCw2REFBNkQ7UUFDN0QsYUFBYTtRQUNiLGNBQWMsQ0FDZixDQUFDO1FBQ0YsSUFBSSxHQUFHO1lBQUUsTUFBTSxHQUFHLENBQUM7UUFDbkIsT0FBTyxHQUFlLENBQUM7SUFDekIsQ0FBQzs7QUEzQ0gsMEJBNENDO0FBM0NnQixnQkFBUSxHQUFHLGlCQUFpQixDQUFDIn0=