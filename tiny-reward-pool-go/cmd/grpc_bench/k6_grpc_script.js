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


  // draw 10 times per request
  const batchedSzie = 10
  const data = { count: batchedSzie };
  const stream = new grpc.Stream(client, 'rewardpool.RewardPoolService/Draw')
  
  let success = true
  stream.on('data', data => {
    // console.log(JSON.stringify(data));
  })

  stream.on('error', () => {
    success = false
  })

  stream.on('end', (data) =>{
    // console.log("end", JSON.stringify(data));
    check(success, {
      [`status is OK - batchedSzie=${batchedSzie} / call`]: success,
    });
    client.close();
  })

  // finished write
  stream.write(data)
  stream.end()
  sleep(0.5);
};