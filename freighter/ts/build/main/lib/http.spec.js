"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const ava_1 = __importDefault(require("ava"));
const encoder_1 = require("./encoder");
const endpoint_1 = __importDefault(require("./endpoint"));
const http_1 = __importDefault(require("./http"));
const ENDPOINT = new endpoint_1.default({
    host: '127.0.0.1',
    port: 8080,
    pathPrefix: 'http',
});
(0, ava_1.default)('[http] - post echo', async (t) => {
    const client = new http_1.default(ENDPOINT, new encoder_1.JSONEncoderDecoder());
    const post = client.post();
    const [response, error] = await post.send('/echo', {
        id: 1,
        message: 'hello',
    });
    t.is(error, undefined);
    t.deepEqual(response, { id: 2, message: 'hello' });
});
(0, ava_1.default)('[http] - get echo', async (t) => {
    const client = new http_1.default(ENDPOINT, new encoder_1.JSONEncoderDecoder());
    const get = client.get();
    const [response, error] = await get.send('/echo', {
        id: 1,
        message: 'hello',
    });
    t.is(error, undefined);
    t.deepEqual(response, { id: 2, message: 'hello' });
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiaHR0cC5zcGVjLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vc3JjL2xpYi9odHRwLnNwZWMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6Ijs7Ozs7QUFBQSw4Q0FBdUI7QUFFdkIsdUNBQStDO0FBQy9DLDBEQUFrQztBQUNsQyxrREFBZ0M7QUFFaEMsTUFBTSxRQUFRLEdBQUcsSUFBSSxrQkFBUSxDQUFDO0lBQzVCLElBQUksRUFBRSxXQUFXO0lBQ2pCLElBQUksRUFBRSxJQUFJO0lBQ1YsVUFBVSxFQUFFLE1BQU07Q0FDbkIsQ0FBQyxDQUFDO0FBRUgsSUFBQSxhQUFJLEVBQUMsb0JBQW9CLEVBQUUsS0FBSyxFQUFFLENBQUMsRUFBRSxFQUFFO0lBQ3JDLE1BQU0sTUFBTSxHQUFHLElBQUksY0FBVSxDQUFDLFFBQVEsRUFBRSxJQUFJLDRCQUFrQixFQUFFLENBQUMsQ0FBQztJQUNsRSxNQUFNLElBQUksR0FBRyxNQUFNLENBQUMsSUFBSSxFQUFFLENBQUM7SUFDM0IsTUFBTSxDQUFDLFFBQVEsRUFBRSxLQUFLLENBQUMsR0FBRyxNQUFNLElBQUksQ0FBQyxJQUFJLENBQUMsT0FBTyxFQUFFO1FBQ2pELEVBQUUsRUFBRSxDQUFDO1FBQ0wsT0FBTyxFQUFFLE9BQU87S0FDakIsQ0FBQyxDQUFDO0lBQ0gsQ0FBQyxDQUFDLEVBQUUsQ0FBQyxLQUFLLEVBQUUsU0FBUyxDQUFDLENBQUM7SUFDdkIsQ0FBQyxDQUFDLFNBQVMsQ0FBQyxRQUFRLEVBQUUsRUFBRSxFQUFFLEVBQUUsQ0FBQyxFQUFFLE9BQU8sRUFBRSxPQUFPLEVBQUUsQ0FBQyxDQUFDO0FBQ3JELENBQUMsQ0FBQyxDQUFDO0FBRUgsSUFBQSxhQUFJLEVBQUMsbUJBQW1CLEVBQUUsS0FBSyxFQUFFLENBQUMsRUFBRSxFQUFFO0lBQ3BDLE1BQU0sTUFBTSxHQUFHLElBQUksY0FBVSxDQUFDLFFBQVEsRUFBRSxJQUFJLDRCQUFrQixFQUFFLENBQUMsQ0FBQztJQUNsRSxNQUFNLEdBQUcsR0FBRyxNQUFNLENBQUMsR0FBRyxFQUFFLENBQUM7SUFDekIsTUFBTSxDQUFDLFFBQVEsRUFBRSxLQUFLLENBQUMsR0FBRyxNQUFNLEdBQUcsQ0FBQyxJQUFJLENBQUMsT0FBTyxFQUFFO1FBQ2hELEVBQUUsRUFBRSxDQUFDO1FBQ0wsT0FBTyxFQUFFLE9BQU87S0FDakIsQ0FBQyxDQUFDO0lBQ0gsQ0FBQyxDQUFDLEVBQUUsQ0FBQyxLQUFLLEVBQUUsU0FBUyxDQUFDLENBQUM7SUFDdkIsQ0FBQyxDQUFDLFNBQVMsQ0FBQyxRQUFRLEVBQUUsRUFBRSxFQUFFLEVBQUUsQ0FBQyxFQUFFLE9BQU8sRUFBRSxPQUFPLEVBQUUsQ0FBQyxDQUFDO0FBQ3JELENBQUMsQ0FBQyxDQUFDIn0=