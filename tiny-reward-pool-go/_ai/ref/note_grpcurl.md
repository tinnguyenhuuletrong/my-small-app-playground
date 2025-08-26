
- https://github.com/fullstorydev/grpcurl
```sh

# List with reflection enabled
grpcurl -plaintext localhost:50051 list

grpcurl -plaintext localhost:50051 describe rewardpool.RewardPoolService.GetState

# List
grpcurl -plaintext -proto ./pkg/rewardpool-grpc-service/rewardpool.proto localhost:50051 list

# Call State
grpcurl -plaintext \
-proto ./pkg/rewardpool-grpc-service/rewardpool.proto \
localhost:50051 rewardpool.RewardPoolService/GetState

# Call Draw
grpcurl -plaintext \
-d @ \
-proto ./pkg/rewardpool-grpc-service/rewardpool.proto \
localhost:50051 rewardpool.RewardPoolService/Draw <<EOM
{
  "count": 10
}
EOM
```