import grpc from 'k6/net/grpc';
import { check, sleep } from 'k6';

const client = new grpc.Client();

client.load(['pkg/proto/communications'], 'communications-service.proto');

export const options = {
  stages: [
    { duration: '10s', target: 100 },
    { duration: '2m', target: 300 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    'grpc_req_duration': ['p(95)<2000'],
  },
};

const heavyPayload = "X".repeat(1024);

export default () => {
  const target = __ENV.TARGET || 'localhost:50053';
  
  const originNode = __ENV.ORIGIN || 'gn_4c3e3';
  const destNode = __ENV.DESTINATION || 'on_8bbd4';

  client.connect(target, {
    plaintext: true,
    reflect: false,
  });

  const data = {
    origin_id: originNode, 
    destination_id: destNode,
    payload: heavyPayload 
  };

  const response = client.invoke('communications.CommunicationsService/SendMessage', data);

  check(response, {
    'status is OK': (r) => r && r.status === grpc.StatusOK,
  });

  client.close();
};