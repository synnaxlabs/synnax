"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const freighter_1 = require("@synnaxlabs/freighter");
const ava_1 = __importDefault(require("ava"));
const setupspecs_1 = require("../setupspecs");
const auth_1 = __importDefault(require("./auth"));
const errors_1 = require("./errors");
const transport_1 = __importDefault(require("./transport"));
(0, ava_1.default)('[auth] - valid credentials', async (t) => {
    const transport = new transport_1.default(new freighter_1.URL({ host: setupspecs_1.HOST, port: setupspecs_1.PORT }));
    const client = new auth_1.default(transport.httpFactory, {
        username: 'synnax',
        password: 'seldon',
    });
    await client.authenticating;
    t.assert(client.authenticated);
});
(0, ava_1.default)('[auth] - invalid credentials', async (t) => {
    const transport = new transport_1.default(new freighter_1.URL({ host: setupspecs_1.HOST, port: setupspecs_1.PORT }));
    const client = new auth_1.default(transport.httpFactory, {
        username: 'synnax',
        password: 'wrong',
    });
    try {
        await client.authenticating;
        t.assert(false);
    }
    catch (e) {
        t.assert(!client.authenticated);
        t.assert(e instanceof errors_1.AuthError);
        if (e instanceof errors_1.AuthError) {
            t.is(e.message, '[synnax] - invalid credentials');
        }
    }
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYXV0aC5zcGVjLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vc3JjL2xpYi9hdXRoLnNwZWMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6Ijs7Ozs7QUFBQSxxREFBNEM7QUFDNUMsOENBQXVCO0FBRXZCLDhDQUEyQztBQUUzQyxrREFBMEM7QUFDMUMscUNBQXFDO0FBQ3JDLDREQUFvQztBQUVwQyxJQUFBLGFBQUksRUFBQyw0QkFBNEIsRUFBRSxLQUFLLEVBQUUsQ0FBQyxFQUFFLEVBQUU7SUFDN0MsTUFBTSxTQUFTLEdBQUcsSUFBSSxtQkFBUyxDQUFDLElBQUksZUFBRyxDQUFDLEVBQUUsSUFBSSxFQUFFLGlCQUFJLEVBQUUsSUFBSSxFQUFFLGlCQUFJLEVBQUUsQ0FBQyxDQUFDLENBQUM7SUFDckUsTUFBTSxNQUFNLEdBQUcsSUFBSSxjQUFvQixDQUFDLFNBQVMsQ0FBQyxXQUFXLEVBQUU7UUFDN0QsUUFBUSxFQUFFLFFBQVE7UUFDbEIsUUFBUSxFQUFFLFFBQVE7S0FDbkIsQ0FBQyxDQUFDO0lBQ0gsTUFBTSxNQUFNLENBQUMsY0FBYyxDQUFDO0lBQzVCLENBQUMsQ0FBQyxNQUFNLENBQUMsTUFBTSxDQUFDLGFBQWEsQ0FBQyxDQUFDO0FBQ2pDLENBQUMsQ0FBQyxDQUFDO0FBRUgsSUFBQSxhQUFJLEVBQUMsOEJBQThCLEVBQUUsS0FBSyxFQUFFLENBQUMsRUFBRSxFQUFFO0lBQy9DLE1BQU0sU0FBUyxHQUFHLElBQUksbUJBQVMsQ0FBQyxJQUFJLGVBQUcsQ0FBQyxFQUFFLElBQUksRUFBRSxpQkFBSSxFQUFFLElBQUksRUFBRSxpQkFBSSxFQUFFLENBQUMsQ0FBQyxDQUFDO0lBQ3JFLE1BQU0sTUFBTSxHQUFHLElBQUksY0FBb0IsQ0FBQyxTQUFTLENBQUMsV0FBVyxFQUFFO1FBQzdELFFBQVEsRUFBRSxRQUFRO1FBQ2xCLFFBQVEsRUFBRSxPQUFPO0tBQ2xCLENBQUMsQ0FBQztJQUNILElBQUk7UUFDRixNQUFNLE1BQU0sQ0FBQyxjQUFjLENBQUM7UUFDNUIsQ0FBQyxDQUFDLE1BQU0sQ0FBQyxLQUFLLENBQUMsQ0FBQztLQUNqQjtJQUFDLE9BQU8sQ0FBQyxFQUFFO1FBQ1YsQ0FBQyxDQUFDLE1BQU0sQ0FBQyxDQUFDLE1BQU0sQ0FBQyxhQUFhLENBQUMsQ0FBQztRQUNoQyxDQUFDLENBQUMsTUFBTSxDQUFDLENBQUMsWUFBWSxrQkFBUyxDQUFDLENBQUM7UUFDakMsSUFBSSxDQUFDLFlBQVksa0JBQVMsRUFBRTtZQUMxQixDQUFDLENBQUMsRUFBRSxDQUFDLENBQUMsQ0FBQyxPQUFPLEVBQUUsZ0NBQWdDLENBQUMsQ0FBQztTQUNuRDtLQUNGO0FBQ0gsQ0FBQyxDQUFDLENBQUMifQ==