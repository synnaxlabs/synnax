"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const ava_1 = __importDefault(require("ava"));
const endpoint_1 = __importDefault(require("./endpoint"));
(0, ava_1.default)('[endpoint] - test path', (t) => {
    const endpoint = new endpoint_1.default({
        host: 'localhost',
        port: 8080,
        protocol: 'http',
        pathPrefix: 'api',
    });
    t.is(endpoint.path('test'), 'http://localhost:8080/api/test');
});
(0, ava_1.default)('[endpoint] - child', (t) => {
    const endpoint = new endpoint_1.default({
        host: 'localhost',
        port: 8080,
        protocol: 'http',
        pathPrefix: 'api',
    });
    const child = endpoint.child({ path: 'test' });
    t.is(child.path('test'), 'http://localhost:8080/api/test/test');
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZW5kcG9pbnQuc3BlYy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uL3NyYy9saWIvZW5kcG9pbnQuc3BlYy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOzs7OztBQUFBLDhDQUF1QjtBQUV2QiwwREFBa0M7QUFFbEMsSUFBQSxhQUFJLEVBQUMsd0JBQXdCLEVBQUUsQ0FBQyxDQUFDLEVBQUUsRUFBRTtJQUNuQyxNQUFNLFFBQVEsR0FBRyxJQUFJLGtCQUFRLENBQUM7UUFDNUIsSUFBSSxFQUFFLFdBQVc7UUFDakIsSUFBSSxFQUFFLElBQUk7UUFDVixRQUFRLEVBQUUsTUFBTTtRQUNoQixVQUFVLEVBQUUsS0FBSztLQUNsQixDQUFDLENBQUM7SUFDSCxDQUFDLENBQUMsRUFBRSxDQUFDLFFBQVEsQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUFDLEVBQUUsZ0NBQWdDLENBQUMsQ0FBQztBQUNoRSxDQUFDLENBQUMsQ0FBQztBQUVILElBQUEsYUFBSSxFQUFDLG9CQUFvQixFQUFFLENBQUMsQ0FBQyxFQUFFLEVBQUU7SUFDL0IsTUFBTSxRQUFRLEdBQUcsSUFBSSxrQkFBUSxDQUFDO1FBQzVCLElBQUksRUFBRSxXQUFXO1FBQ2pCLElBQUksRUFBRSxJQUFJO1FBQ1YsUUFBUSxFQUFFLE1BQU07UUFDaEIsVUFBVSxFQUFFLEtBQUs7S0FDbEIsQ0FBQyxDQUFDO0lBQ0gsTUFBTSxLQUFLLEdBQUcsUUFBUSxDQUFDLEtBQUssQ0FBQyxFQUFFLElBQUksRUFBRSxNQUFNLEVBQUUsQ0FBQyxDQUFDO0lBQy9DLENBQUMsQ0FBQyxFQUFFLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQyxNQUFNLENBQUMsRUFBRSxxQ0FBcUMsQ0FBQyxDQUFDO0FBQ2xFLENBQUMsQ0FBQyxDQUFDIn0=