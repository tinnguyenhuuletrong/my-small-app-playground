import grpc from 'k6/net/grpc';
import { check, sleep } from 'k6';

// Download quickpizza.proto for grpc-quickpizza.grafana.com, located at:
// https://raw.githubusercontent.com/grafana/quickpizza/refs/heads/main/proto/quickpizza.proto
// and put it in the same folder as this script.
const client = new grpc.Client();
client.load(['../../pkg/rewardpool-grpc-service'], 'rewardpool.proto');

export default () => {
  client.connect('localhost:50051', {
    plaintext: true
  });

  const data = { count: 1 };
  const response = client.invoke('rewardpool.RewardPoolService/Draw', data);

  check(response, {
    'status is OK': (r) => r && r.status === grpc.StatusOK,
  });

//   console.log(JSON.stringify(response.message));

  client.close();
  sleep(1);
};