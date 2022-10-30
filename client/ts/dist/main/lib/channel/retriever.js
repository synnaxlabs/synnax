"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const zod_1 = require("zod");
const payload_1 = require("./payload");
const requestSchema = zod_1.z.object({
    keys: zod_1.z.string().array().optional(),
    nodeId: zod_1.z.number().optional(),
    names: zod_1.z.string().array().optional(),
});
const responseSchema = zod_1.z.object({
    channels: payload_1.channelPayloadSchema.array(),
});
class Retriever {
    constructor(transport) {
        this.client = transport.getClient();
    }
    async execute(request) {
        const [res, err] = await this.client.send(Retriever.ENDPOINT, request, 
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        responseSchema);
        if (err)
            throw err;
        return res === null || res === void 0 ? void 0 : res.channels;
    }
    async retrieveByKeys(...keys) {
        return await this.execute({ keys });
    }
    async retrieveByNames(...names) {
        return await this.execute({ names });
    }
    async retrieveByNodeID(nodeId) {
        return await this.execute({ nodeId });
    }
    async retrieveAll() {
        return await this.execute({});
    }
}
exports.default = Retriever;
Retriever.ENDPOINT = '/channel/retrieve';
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicmV0cmlldmVyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vLi4vc3JjL2xpYi9jaGFubmVsL3JldHJpZXZlci50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOztBQUNBLDZCQUF3QjtBQUl4Qix1Q0FBaUU7QUFFakUsTUFBTSxhQUFhLEdBQUcsT0FBQyxDQUFDLE1BQU0sQ0FBQztJQUM3QixJQUFJLEVBQUUsT0FBQyxDQUFDLE1BQU0sRUFBRSxDQUFDLEtBQUssRUFBRSxDQUFDLFFBQVEsRUFBRTtJQUNuQyxNQUFNLEVBQUUsT0FBQyxDQUFDLE1BQU0sRUFBRSxDQUFDLFFBQVEsRUFBRTtJQUM3QixLQUFLLEVBQUUsT0FBQyxDQUFDLE1BQU0sRUFBRSxDQUFDLEtBQUssRUFBRSxDQUFDLFFBQVEsRUFBRTtDQUNyQyxDQUFDLENBQUM7QUFJSCxNQUFNLGNBQWMsR0FBRyxPQUFDLENBQUMsTUFBTSxDQUFDO0lBQzlCLFFBQVEsRUFBRSw4QkFBb0IsQ0FBQyxLQUFLLEVBQUU7Q0FDdkMsQ0FBQyxDQUFDO0FBRUgsTUFBcUIsU0FBUztJQUk1QixZQUFZLFNBQW9CO1FBQzlCLElBQUksQ0FBQyxNQUFNLEdBQUcsU0FBUyxDQUFDLFNBQVMsRUFBRSxDQUFDO0lBQ3RDLENBQUM7SUFFTyxLQUFLLENBQUMsT0FBTyxDQUFDLE9BQWdCO1FBQ3BDLE1BQU0sQ0FBQyxHQUFHLEVBQUUsR0FBRyxDQUFDLEdBQUcsTUFBTSxJQUFJLENBQUMsTUFBTSxDQUFDLElBQUksQ0FDdkMsU0FBUyxDQUFDLFFBQVEsRUFDbEIsT0FBTztRQUNQLDZEQUE2RDtRQUM3RCxhQUFhO1FBQ2IsY0FBYyxDQUNmLENBQUM7UUFDRixJQUFJLEdBQUc7WUFBRSxNQUFNLEdBQUcsQ0FBQztRQUNuQixPQUFPLEdBQUcsYUFBSCxHQUFHLHVCQUFILEdBQUcsQ0FBRSxRQUE0QixDQUFDO0lBQzNDLENBQUM7SUFFRCxLQUFLLENBQUMsY0FBYyxDQUFDLEdBQUcsSUFBYztRQUNwQyxPQUFPLE1BQU0sSUFBSSxDQUFDLE9BQU8sQ0FBQyxFQUFFLElBQUksRUFBRSxDQUFDLENBQUM7SUFDdEMsQ0FBQztJQUVELEtBQUssQ0FBQyxlQUFlLENBQUMsR0FBRyxLQUFlO1FBQ3RDLE9BQU8sTUFBTSxJQUFJLENBQUMsT0FBTyxDQUFDLEVBQUUsS0FBSyxFQUFFLENBQUMsQ0FBQztJQUN2QyxDQUFDO0lBRUQsS0FBSyxDQUFDLGdCQUFnQixDQUFDLE1BQWM7UUFDbkMsT0FBTyxNQUFNLElBQUksQ0FBQyxPQUFPLENBQUMsRUFBRSxNQUFNLEVBQUUsQ0FBQyxDQUFDO0lBQ3hDLENBQUM7SUFFRCxLQUFLLENBQUMsV0FBVztRQUNmLE9BQU8sTUFBTSxJQUFJLENBQUMsT0FBTyxDQUFDLEVBQUUsQ0FBQyxDQUFDO0lBQ2hDLENBQUM7O0FBbENILDRCQW1DQztBQWxDZ0Isa0JBQVEsR0FBRyxtQkFBbUIsQ0FBQyJ9