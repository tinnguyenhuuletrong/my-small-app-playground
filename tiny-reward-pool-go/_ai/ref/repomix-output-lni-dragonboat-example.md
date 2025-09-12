This file is a merged representation of the entire codebase, combined into a single document by Repomix.
The content has been processed where security check has been disabled.

# File Summary

## Purpose
This file contains a packed representation of the entire repository's contents.
It is designed to be easily consumable by AI systems for analysis, code review,
or other automated processes.

## File Format
The content is organized as follows:
1. This summary section
2. Repository information
3. Directory structure
4. Repository files (if enabled)
5. Multiple file entries, each consisting of:
  a. A header with the file path (## File: path/to/file)
  b. The full contents of the file in a code block

## Usage Guidelines
- This file should be treated as read-only. Any changes should be made to the
  original repository files, not this packed version.
- When processing this file, use the file path to distinguish
  between different files in the repository.
- Be aware that this file may contain sensitive information. Handle it with
  the same level of security as you would the original repository.

## Notes
- Some files may have been excluded based on .gitignore rules and Repomix's configuration
- Binary files are not included in this packed representation. Please refer to the Repository Structure section for a complete list of file paths, including binary files
- Files matching patterns in .gitignore are excluded
- Files matching default ignore patterns are excluded
- Security check has been disabled - content may contain sensitive information
- Files are sorted by Git change count (files with more changes are at the bottom)

# Directory Structure
```
helloworld/
  main.go
  README.CHS.md
  README.DS.CHS.md
  README.DS.md
  README.md
  statemachine.go
multigroup/
  main.go
  README.CHS.md
  README.md
  statemachine.go
  statemachine2.go
ondisk/
  diskkv.go
  main.go
  README.CHS.md
  README.md
optimistic-write-lock/
  fsm.go
  handler.go
  main.go
  README.md
.gitignore
go.mod
LICENSE
Makefile
README.CHS.md
README.md
```

# Files

## File: helloworld/main.go
````go
// Copyright 2017,2018 Lei Ni (nilei81@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
helloworld is an example program for dragonboat.
*/
package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/logger"
	"github.com/lni/goutils/syncutil"
)

const (
	exampleShardID uint64 = 128
)

var (
	// initial nodes count is fixed to three, their addresses are also fixed
	// these are the initial member nodes of the Raft cluster.
	addresses = []string{
		"localhost:63001",
		"localhost:63002",
		"localhost:63003",
	}
	errNotMembershipChange = errors.New("not a membership change request")
)

// makeMembershipChange makes membership change request.
func makeMembershipChange(nh *dragonboat.NodeHost,
	cmd string, addr string, replicaID uint64) {
	var rs *dragonboat.RequestState
	var err error
	if cmd == "add" {
		// orderID is ignored in standalone mode
		rs, err = nh.RequestAddReplica(exampleShardID, replicaID, addr, 0, 3*time.Second)
	} else if cmd == "remove" {
		rs, err = nh.RequestDeleteReplica(exampleShardID, replicaID, 0, 3*time.Second)
	} else {
		panic("unknown cmd")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "membership change failed, %v\n", err)
		return
	}
	select {
	case r := <-rs.CompletedC:
		if r.Completed() {
			fmt.Fprintf(os.Stdout, "membership change completed successfully\n")
		} else {
			fmt.Fprintf(os.Stderr, "membership change failed\n")
		}
	}
}

// splitMembershipChangeCmd tries to parse the input string as membership change
// request. ADD node request has the following expected format -
// add localhost:63100 4
// REMOVE node request has the following expected format -
// remove 4
func splitMembershipChangeCmd(v string) (string, string, uint64, error) {
	parts := strings.Split(v, " ")
	if len(parts) == 2 || len(parts) == 3 {
		cmd := strings.ToLower(strings.TrimSpace(parts[0]))
		if cmd != "add" && cmd != "remove" {
			return "", "", 0, errNotMembershipChange
		}
		addr := ""
		var replicaIDStr string
		var replicaID uint64
		var err error
		if cmd == "add" {
			addr = strings.TrimSpace(parts[1])
			replicaIDStr = strings.TrimSpace(parts[2])
		} else {
			replicaIDStr = strings.TrimSpace(parts[1])
		}
		if replicaID, err = strconv.ParseUint(replicaIDStr, 10, 64); err != nil {
			return "", "", 0, errNotMembershipChange
		}
		return cmd, addr, replicaID, nil
	}
	return "", "", 0, errNotMembershipChange
}

func main() {
	replicaID := flag.Int("replicaid", 1, "ReplicaID to use")
	addr := flag.String("addr", "", "Nodehost address")
	join := flag.Bool("join", false, "Joining a new node")
	flag.Parse()
	if len(*addr) == 0 && *replicaID != 1 && *replicaID != 2 && *replicaID != 3 {
		fmt.Fprintf(os.Stderr, "node id must be 1, 2 or 3 when address is not specified\n")
		os.Exit(1)
	}
	// https://github.com/golang/go/issues/17393
	if runtime.GOOS == "darwin" {
		signal.Ignore(syscall.Signal(0xd))
	}
	initialMembers := make(map[uint64]string)
	// when joining a new node which is not an initial members, the initialMembers
	// map should be empty.
	// when restarting a node that is not a member of the initial nodes, you can
	// leave the initialMembers to be empty. we still populate the initialMembers
	// here for simplicity.
	if !*join {
		for idx, v := range addresses {
			// key is the ReplicaID, ReplicaID is not allowed to be 0
			// value is the raft address
			initialMembers[uint64(idx+1)] = v
		}
	}
	var nodeAddr string
	// for simplicity, in this example program, addresses of all those 3 initial
	// raft members are hard coded. when address is not specified on the command
	// line, we assume the node being launched is an initial raft member.
	if len(*addr) != 0 {
		nodeAddr = *addr
	} else {
		nodeAddr = initialMembers[uint64(*replicaID)]
	}
	fmt.Fprintf(os.Stdout, "node address: %s\n", nodeAddr)
	// change the log verbosity
	logger.GetLogger("raft").SetLevel(logger.ERROR)
	logger.GetLogger("rsm").SetLevel(logger.WARNING)
	logger.GetLogger("transport").SetLevel(logger.WARNING)
	logger.GetLogger("grpc").SetLevel(logger.WARNING)
	// config for raft node
	// See GoDoc for all available options
	rc := config.Config{
		// ShardID and ReplicaID of the raft node
		ReplicaID: uint64(*replicaID),
		ShardID:   exampleShardID,
		// In this example, we assume the end-to-end round trip time (RTT) between
		// NodeHost instances (on different machines, VMs or containers) are 200
		// millisecond, it is set in the RTTMillisecond field of the
		// config.NodeHostConfig instance below.
		// ElectionRTT is set to 10 in this example, it determines that the node
		// should start an election if there is no heartbeat from the leader for
		// 10 * RTT time intervals.
		ElectionRTT: 10,
		// HeartbeatRTT is set to 1 in this example, it determines that when the
		// node is a leader, it should broadcast heartbeat messages to its followers
		// every such 1 * RTT time interval.
		HeartbeatRTT: 1,
		CheckQuorum:  true,
		// SnapshotEntries determines how often should we take a snapshot of the
		// replicated state machine, it is set to 10 her which means a snapshot
		// will be captured for every 10 applied proposals (writes).
		// In your real world application, it should be set to much higher values
		// You need to determine a suitable value based on how much space you are
		// willing use on Raft Logs, how fast can you capture a snapshot of your
		// replicated state machine, how often such snapshot is going to be used
		// etc.
		SnapshotEntries: 10,
		// Once a snapshot is captured and saved, how many Raft entries already
		// covered by the new snapshot should be kept. This is useful when some
		// followers are just a little bit left behind, with such overhead Raft
		// entries, the leaders can send them regular entries rather than the full
		// snapshot image.
		CompactionOverhead: 5,
	}
	datadir := filepath.Join(
		"example-data",
		"helloworld-data",
		fmt.Sprintf("node%d", *replicaID))
	// config for the nodehost
	// See GoDoc for all available options
	// by default, insecure transport is used, you can choose to use Mutual TLS
	// Authentication to authenticate both servers and clients. To use Mutual
	// TLS Authentication, set the MutualTLS field in NodeHostConfig to true, set
	// the CAFile, CertFile and KeyFile fields to point to the path of your CA
	// file, certificate and key files.
	nhc := config.NodeHostConfig{
		// WALDir is the directory to store the WAL of all Raft Logs. It is
		// recommended to use Enterprise SSDs with good fsync() performance
		// to get the best performance. A few SSDs we tested or known to work very
		// well
		// Recommended SATA SSDs -
		// Intel S3700, Intel S3710, Micron 500DC
		// Other SATA enterprise class SSDs with power loss protection
		// Recommended NVME SSDs -
		// Most enterprise NVME currently available on the market.
		// SSD to avoid -
		// Consumer class SSDs, no matter whether they are SATA or NVME based, as
		// they usually have very poor fsync() performance.
		//
		// You can use the pg_test_fsync tool shipped with PostgreSQL to test the
		// fsync performance of your WAL disk. It is recommended to use SSDs with
		// fsync latency of well below 1 millisecond.
		//
		// Note that this is only for storing the WAL of Raft Logs, it is size is
		// usually pretty small, 64GB per NodeHost is usually more than enough.
		//
		// If you just have one disk in your system, just set WALDir and NodeHostDir
		// to the same location.
		WALDir: datadir,
		// NodeHostDir is where everything else is stored.
		NodeHostDir: datadir,
		// RTTMillisecond is the average round trip time between NodeHosts (usually
		// on two machines/vms), it is in millisecond. Such RTT includes the
		// processing delays caused by NodeHosts, not just the network delay between
		// two NodeHost instances.
		RTTMillisecond: 200,
		// RaftAddress is used to identify the NodeHost instance
		RaftAddress: nodeAddr,
	}
	nh, err := dragonboat.NewNodeHost(nhc)
	if err != nil {
		panic(err)
	}
	if err := nh.StartReplica(initialMembers, *join, NewExampleStateMachine, rc); err != nil {
		fmt.Fprintf(os.Stderr, "failed to add cluster, %v\n", err)
		os.Exit(1)
	}
	raftStopper := syncutil.NewStopper()
	consoleStopper := syncutil.NewStopper()
	ch := make(chan string, 16)
	consoleStopper.RunWorker(func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			s, err := reader.ReadString('\n')
			if err != nil {
				close(ch)
				return
			}
			if s == "exit\n" {
				raftStopper.Stop()
				// no data will be lost/corrupted if nodehost.Stop() is not called
				nh.Close()
				return
			}
			ch <- s
		}
	})
	raftStopper.RunWorker(func() {
		// this goroutine makes a linearizable read every 10 second. it returns the
		// Count value maintained in IStateMachine. see datastore.go for details.
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				result, err := nh.SyncRead(ctx, exampleShardID, []byte{})
				cancel()
				if err == nil {
					var count uint64
					count = binary.LittleEndian.Uint64(result.([]byte))
					fmt.Fprintf(os.Stdout, "count: %d\n", count)
				}
			case <-raftStopper.ShouldStop():
				return
			}
		}
	})
	raftStopper.RunWorker(func() {
		// use a NO-OP client session here
		// check the example in godoc to see how to use a regular client session
		cs := nh.GetNoOPSession(exampleShardID)
		for {
			select {
			case v, ok := <-ch:
				if !ok {
					return
				}
				// remove the \n char
				msg := strings.Replace(v, "\n", "", 1)
				if cmd, addr, replicaID, err := splitMembershipChangeCmd(msg); err == nil {
					// input is a membership change request
					makeMembershipChange(nh, cmd, addr, replicaID)
				} else {
					// input is a regular message need to be proposed
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					// make a proposal to update the IStateMachine instance
					_, err := nh.SyncPropose(ctx, cs, []byte(msg))
					cancel()
					if err != nil {
						fmt.Fprintf(os.Stderr, "SyncPropose returned error %v\n", err)
					}
				}
			case <-raftStopper.ShouldStop():
				return
			}
		}
	})
	raftStopper.Wait()
}
````

## File: helloworld/README.CHS.md
````markdown
# 示例1 - Hello World #

## 关于 ##
本示例将给您一个Dragonboat的基本功能的概览，包括：

* 如何配置并开始一个新的NodeHost实例
* 如何开始一个Raft组
* 如何发起一个proposal以改变Raft状态
* 如何做强一致读
* 如何触发一个snapshot快照
* 如何完成Raft组的成员变更

## 编译 ##
执行下面的命令以编译helloworld程序：
```
cd $HOME/src/dragonboat-example
make helloworld
```

## 首次运行 ##
在同一主机上三个不同的终端terminals上启动三个helloworld进程：

```
./example-helloworld -replicaid 1
```
```
./example-helloworld -replicaid 2
```
```
./example-helloworld -replicaid 3
```
这将组建一个三节点的Raft集群，每个节点都由上述命令行命令中-replicaid所指定的NodeID值来标示。为求简易，本示例被设定为需用三个节点且NodeID值必须为1, 2, 3。

在任意一个终端terminal中输入一个字符串并按下键盘回车，这样的外部输入通常被称为一个提议（proposal），它会被复制到其它节点中。在不同的终端内重复几次这样的操作，此时请注意各个terminal上显示的计数信息和回显的消息的次序，在三个不同的terminals上它们应该是完全一致对应的。这演示了Dragonboat所实现的Raft的最核心功能：在分布的节点上达成共识。

```
2017-09-09 00:02:08.873755 I | transport: raft gRPC stream from [00128:00002] to [00128:00001] established
from Update() : msg: hi there, count:1
```
In the log message above, it mentions "[00128:00002]" and "[00128:00001]". They are used to identify two Raft nodes in log messages, the first one means it is a node from Raft Cluster with ShardID 128, its NodeID is 2.

在上述log消息中提到了"[00128:00002]"和"[00128:00001]"。它们是用来在log消息中指代Raft节点的，比如第一个的意义是一个来自ShardID为128的Raft组的NodeID为2的节点。

每个helloworld进程都有一个后台的goroutine以每10秒一次的频率做强一致读（Linearizable Read）。它查询已经应用的提议个数并将此结果显示到terminal上。

## 多数派 Quorum ##
只要任意多数的节点在正常工作，那么该Raft集群被认为具备多数派Quorum。对于一个三节点集群，只要至少两个任意节点在正常工作，那么系统便具备多数派Quorum。

现在选择任意一个terminal中按Ctrl+C以中止它的运行。在剩余的两个terminal中的任意一个输入消息，所输入的消息应该依旧可以被复制到另外一个节点上并被回显到terminal上。此时，使用Ctrl+C再次中止一个节点后再次输入一些消息，此时便不能继续看到所输入的消息被回显到terminal上，并且应该很快看到一个如下的超时错误提示消息：

```
failed to get a proposal session, timeout
2017-09-09 00:04:09.039257 W | transport: breaker localhost:63003 failed, connect and process failed: context deadline exceeded
```

## 重启节点 ##
我们选定一个已经被停止的节点，然后用首次启动它的时候一样的命令来重启它。比如，那个首次启动时候用了含有*-replicaid 2*字样命令的节点，可以用下面的命令在其原来的terminal中重启：
```
./example-helloworld -replicaid 2
```

此时，之前被复制的消息再次被应用并回显，并且它们的次序和之前完全一致。这是因为Dragonboat内部记录所有已提交commit成功的Proposals，在重启的时候它们将再次被应用到重启以后的进程中，以确保重启后的节点能恢复到之前失效时候的状态。

## 状态快照 Snapshot ##
在提交一定数量提议后，终端上会提示"snapshot captured"字样的消息。快照snapshot这一行为由raft.Config中的SnapshotEntries参数来控制。快照使得一个Raft程序的状态可以被快速整体捕获保存，从而可以被用于整体恢复程序的状态，而不需要逐一、大量地应用所有已经应用的提议。

## 成员变更 ##
在本例程中，Raft group的成员可以通过输入特殊消息来完成变更。下列输入到helloworld例程中的特殊消息将请求使得位于localhost:63100地址的具有replica ID 4的一个新节点被加入到Raft group中。
```
add localhost:63100 4
```
一旦成员变更完成，便可以在一个新的终端中使用下列shell命令启动这个节点
```
./example-helloworld -replicaid 4 -addr localhost:63100 -join
```
命令行中的-join使得在调用nodehost.StartCluster时的join参数被设为true，以表示这是一个新的节点加入。

新节点将首先追赶上其余节点的进度，再开始接收新的复制过来的消息。

下列特殊消息将使得replica ID为4的节点被从Raft group被移除掉。
```
remove 4
```
一旦被移除，该节点将无法继续接收复制过来的消息。

请注意，将一个已经移除的节点再次加入回到Raft group中是不被允许的。

## 重新开始 ##
所有保存的数据均位于example-data的子目录内，可以手工删除这个example-data目录从而重新开始本例程。

## 代码 ##
在[statemachine.go](statemachine.go)中，ExampleStateMachine结构体用来实现statemachine.IStateMachine接口。这个data store结构体用来实现应用程序自己的状态机逻辑。具体的内容将在[下一示例](README.DS.CHS.md)中展开。

[main.go](main.go)含有main()入口函数，在里面我们实例化了一个NodeHost实例，把所创建的Raft集群节点加入了这个实例。它同时使用多个goroutine来做用户输入消息和Ctrl+C信号的处理。同时请留意MakeProposal()函数的错误处理部分代码和注释。

makeMembershipChange()函数完成成员的变更。
````

## File: helloworld/README.DS.CHS.md
````markdown
# 示例2 - IStateMachine #

## 关于 ##
本示例介绍如何实现一个[statemachine.IStateMachine](https://godoc.org/github.com/lni/dragonboat/statemachine#IStateMachine)。

## 代码 ##
[statemachine.go](statemachine.go)实现了Dragonboat应用所需要的用于管理用户数据的[statemachine.IStateMachine](https://godoc.org/github.com/lni/dragonboat/statemachine#IStateMachine)接口。在本例子中，我们介绍上一个Helloworld示例中所使用的IStateMachine实例是如何实现的。

首先需要实现Update()与Lookup()方法，用于处理收到的更新与查询请求。在本示例IStateMachine中，它只有一个名为Count的整数计数器，每当Update()被调用时，该计数器会被递增。在Update()方法中，我们同时打印出所收到的输入参数用以演示目的。在一个实际应用里，用户可根据这样的输入参数相应更新IStateMachine的状态。

Lookup()是一个只读的用于查询IStateMachine的方法，在本示例的实现中，仅简单的把计数器的值放入一个byte slice中并返回。名为query的byte slice参数通常是应用提供的用于表明需查询内容的输入参数。

SaveSnapshot()和RecoverFromSnapshot() 用来实现快照的保存与读取。快照需要能包含IStateMachine的完整状态。IStateMachine所维护的内存内的数据可以通过所提供的以磁盘文件为后台存储的io.Writer和io.Reader来保存与恢复。请注意，SaveSnapshot()也是一个只读的方法，它不应该改变IStateMachine的状态。

Close()可以被认为是可选的。因为系统并不保证Close会被最终调用，因此IStateMachine的数据完整性不能依赖于Close()方法。
````

## File: helloworld/README.DS.md
````markdown
# Example 2 - IStateMachine #

## About ##
This example demonstrates how to implement your own [statemachine.IStateMachine](https://godoc.org/github.com/lni/dragonboat/statemachine#IStateMachine). 

## Code ##
[statemachine.go](statemachine.go) implements the [statemachine.IStateMachine](https://godoc.org/github.com/lni/dragonboat/statemachine#IStateMachine) interface required for managing application data in Dragonboat based applications. In this example, we show how the IStateMachine instance used in the previous Helloworld example is implemented.

We first implement the Update() and Lookup() methods for handling incoming updates and queries. In this example IStateMachine, there is a single integer named Count for representing the state of the IStateMachine, the Count integer is increased every time when Update() is invoked. In the Update() method, we also print out the input payload for demonstration purposes. In a real application, users are free to interpret the input data slice and update the state of their IStateMachine accordingly. 

Lookup() is a read-only method for querying the state of the IStateMachine. In the implementation of this example, we just put the Count value into a byte slice and return it. The input byte slice named query is usually used by applications to specify what to query. 

SaveSnapshot() and RecoverFromSnapshot() are used to implement snapshot save and load operations. Those in memory data maintained by your IStateMachine can be saved to or read from the disk file backed io.Writer and io.Reader. The SaveSnapshot() method is also read-only, which means it is not suppose to change the state of the IStateMachine. 

The Close() method should be considered as optional, as there is no guarantee that the Close() method will always be called before a node is stopped or killed, data integrity of the IStateMachine instance must not rely on the Close() method.
````

## File: helloworld/README.md
````markdown
# Example 1 - Hello World #

## About ##
This example aims to give you an overview of Dragonboat's basic features, such as:

* how to config and start a NodeHost instance
* how to start a Raft group (also known as Raft shard)
* how to make proposals to update Raft group state
* how to perform linearizable read from Raft group
* how to trigger snapshot to be generated
* how to perform Raft group membership change

## Build ##
To build the helloworld executable -
```
cd $HOME/src/dragonboat-example
make helloworld
```

## First Run ##
Start three instances of the helloworld program on the same machine in three different terminals:

```
./example-helloworld -replicaid 1
```
```
./example-helloworld -replicaid 2
```
```
./example-helloworld -replicaid 3
```
This forms a Raft group with 3 nodes, each of them is identified by the NodeID specified on the command line. For simplicity, this example program has been hardcoded to use 3 nodes and their node id values are 1, 2 and 3.

Type in a message (also known as a proposal) and press enter in any one of those terminals, the message will be replicated to other nodes. You can try repeating this a few times, probably in different terminals. Note the count numbers and the order of messages, they should be identical across all three terminals. See example log messages below. This demonstrates the core feature of Raft implemented by Dragonboat - reaching consensus in a distributed environment.

```
2017-09-09 00:02:08.873755 I | transport: raft gRPC stream from [00128:00002] to [00128:00001] established
from Update() : msg: hi there, count:1
```
In the log message above, it mentions "[00128:00002]" and "[00128:00001]". They are used to identify two Raft nodes in log messages, the first one means it is a node from Raft group with ShardID 128, its NodeID is 2. 

Each helloworld process has a background goroutine performing linearizable read every 10 seconds. It queries the number of applied proposals and print out the result to the terminal. 

## Quorum ##
As long as the majority of nodes in the Raft group are available, the group is said to has the quorum. For such a 3-nodes Raft group, any two nodes need to be available to have the quorum.

Now press CTRL+C in any one of the terminal to stop the running exmple-helloworld program. Type in some messsages again in any of the remaining two terminals, you should still be able to have the message replicated across to the other node as the Raft group still has the quorum. Let's press CTRL+C to stop another instance, you should notice that further input messages will no longer be printed back on the remaining terminal and you are expected to see a timeout message. 

```
failed to get a proposal session, timeout
2017-09-09 00:04:09.039257 W | transport: breaker localhost:63003 failed, connect and process failed: context deadline exceeded
```

## Restart a node ##
Let's pick a stopped instance and restart it using the exact same command, e.g. for the one which we previously started with the *-replicaid 2* command line option, it can be restarted using the command below - 
```
./example-helloworld -replicaid 2
```

Previously replicated messages are printed back onto the terminal again in the same order as they were initially replicated across. Dragonboat internally records the state of the node and all updates to make sure it can be correctly restored after restart. 

## Snapshotting ##
After proposing several more messages, there will be logs mentioning that "snapshot captured". This is controled by the SnapshotEntries parameter specified in the raft.Config object. Snapshots can be used to restore the state of the program without requiring every single proposed messages to be applied one by one.

## Membership Change ##
In this example program, Raft group membership can be changed by inputing some messages with special format. The following special message causes a new node with node ID 4 running at localhost:63100 to be added to the Raft group.

```
add localhost:63100 4
```
Once the membership change is completed, you can start this recently added node in a new terminal using the following shell command - 
```
./example-helloworld -replicaid 4 -addr localhost:63100 -join
```
The -join option tells the progress to set the join parameter to be true when calling the nodehost.StartReplica() function. 

The new node will catch up with other nodes and start to receive replicated messages.

The following special message removes the node with node ID 4 from the cluster.
```
remove 4
```
Once removed, node will stop receiving further replicated messages.

Note that adding a previously removed node back to the cluster is not allowed.

## Start Over ##
All saved data is saved into the example-data folder, you can delete this example-data folder and restart all processes to start over again.

## Code! ##
In [statemachine.go](statemachine.go), we have this ExampleStateMachine struct which implements the statemachine.IStateMachine interface. This is the data store struct for implementing the application state machine. We will leave the details to the [next example](README.DS.md). 

[main.go](main.go) contains the main() function, it is the place where we instantiated the NodeHost instance, added the created example Raft group to it. It uses multiple goroutines for input and signal handling. Updates to the IStateMachine instance is achieved by making proposals.

makeMembershipChange() shows how to make membership changes, including add or remove nodes.
````

## File: helloworld/statemachine.go
````go
// Copyright 2017,2018 Lei Ni (nilei81@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	sm "github.com/lni/dragonboat/v4/statemachine"
)

// ExampleStateMachine is the IStateMachine implementation used in the
// helloworld example.
// See https://github.com/lni/dragonboat/blob/master/statemachine/rsm.go for
// more details of the IStateMachine interface.
type ExampleStateMachine struct {
	ShardID   uint64
	ReplicaID uint64
	Count     uint64
}

// NewExampleStateMachine creates and return a new ExampleStateMachine object.
func NewExampleStateMachine(shardID uint64,
	replicaID uint64) sm.IStateMachine {
	return &ExampleStateMachine{
		ShardID:   shardID,
		ReplicaID: replicaID,
		Count:     0,
	}
}

// Lookup performs local lookup on the ExampleStateMachine instance. In this example,
// we always return the Count value as a little endian binary encoded byte
// slice.
func (s *ExampleStateMachine) Lookup(query interface{}) (interface{}, error) {
	result := make([]byte, 8)
	binary.LittleEndian.PutUint64(result, s.Count)
	return result, nil
}

// Update updates the object using the specified committed raft entry.
func (s *ExampleStateMachine) Update(e sm.Entry) (sm.Result, error) {
	// in this example, we print out the following hello world message for each
	// incoming update request. we also increase the counter by one to remember
	// how many updates we have applied
	s.Count++
	fmt.Printf("from ExampleStateMachine.Update(), msg: %s, count:%d\n",
		string(e.Cmd), s.Count)
	return sm.Result{Value: uint64(len(e.Cmd))}, nil
}

// SaveSnapshot saves the current IStateMachine state into a snapshot using the
// specified io.Writer object.
func (s *ExampleStateMachine) SaveSnapshot(w io.Writer,
	fc sm.ISnapshotFileCollection, done <-chan struct{}) error {
	// as shown above, the only state that can be saved is the Count variable
	// there is no external file in this IStateMachine example, we thus leave
	// the fc untouched
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, s.Count)
	_, err := w.Write(data)
	return err
}

// RecoverFromSnapshot recovers the state using the provided snapshot.
func (s *ExampleStateMachine) RecoverFromSnapshot(r io.Reader,
	files []sm.SnapshotFile,
	done <-chan struct{}) error {
	// restore the Count variable, that is the only state we maintain in this
	// example, the input files is expected to be empty
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	v := binary.LittleEndian.Uint64(data)
	s.Count = v
	return nil
}

// Close closes the IStateMachine instance. There is nothing for us to cleanup
// or release as this is a pure in memory data store. Note that the Close
// method is not guaranteed to be called as node can crash at any time.
func (s *ExampleStateMachine) Close() error { return nil }
````

## File: multigroup/main.go
````go
// Copyright 2017,2018 Lei Ni (nilei81@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
multigroup is an example program for dragonboat demonstrating how multiple
raft groups can be used in an user application.
*/
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/logger"
	"github.com/lni/goutils/syncutil"
)

const (
	// we use two raft groups in this example, they are identified by the cluster
	// ID values below
	shardID1 uint64 = 100
	shardID2 uint64 = 101
)

var (
	// initial nodes count is three, their addresses are also fixed
	// this is for simplicity
	addresses = []string{
		"localhost:63001",
		"localhost:63002",
		"localhost:63003",
	}
)

func main() {
	replicaID := flag.Int("nodeid", 1, "ReplicaID to use")
	flag.Parse()
	if *replicaID > 3 || *replicaID < 1 {
		fmt.Fprintf(os.Stderr, "invalid nodeid %d, it must be 1, 2 or 3", *replicaID)
		os.Exit(1)
	}
	// https://github.com/golang/go/issues/17393
	if runtime.GOOS == "darwin" {
		signal.Ignore(syscall.Signal(0xd))
	}
	initialMembers := make(map[uint64]string)
	for idx, v := range addresses {
		// key is the ReplicaID, ReplicaID is not allowed to be 0
		// value is the raft address
		initialMembers[uint64(idx+1)] = v
	}
	nodeAddr := initialMembers[uint64(*replicaID)]
	fmt.Fprintf(os.Stdout, "node address: %s\n", nodeAddr)
	// change the log verbosity
	logger.GetLogger("raft").SetLevel(logger.ERROR)
	logger.GetLogger("rsm").SetLevel(logger.WARNING)
	logger.GetLogger("transport").SetLevel(logger.WARNING)
	logger.GetLogger("grpc").SetLevel(logger.WARNING)
	// config for raft
	// note the ShardID value is not specified here
	rc := config.Config{
		ReplicaID:          uint64(*replicaID),
		ElectionRTT:        5,
		HeartbeatRTT:       1,
		CheckQuorum:        true,
		SnapshotEntries:    10,
		CompactionOverhead: 5,
	}
	datadir := filepath.Join(
		"example-data",
		"multigroup-data",
		fmt.Sprintf("node%d", *replicaID))
	// config for the nodehost
	// by default, insecure transport is used, you can choose to use Mutual TLS
	// Authentication to authenticate both servers and clients. To use Mutual
	// TLS Authentication, set the MutualTLS field in NodeHostConfig to true, set
	// the CAFile, CertFile and KeyFile fields to point to the path of your CA
	// file, certificate and key files.
	// by default, TCP based RPC module is used, set the RaftRPCFactory field in
	// NodeHostConfig to rpc.NewRaftGRPC (github.com/lni/dragonboat/plugin/rpc) to
	// use gRPC based transport. To use gRPC based RPC module, you need to install
	// the gRPC library first -
	//
	// $ go get -u google.golang.org/grpc
	//
	nhc := config.NodeHostConfig{
		WALDir:         datadir,
		NodeHostDir:    datadir,
		RTTMillisecond: 200,
		RaftAddress:    nodeAddr,
		// RaftRPCFactory: rpc.NewRaftGRPC,
	}
	// create a NodeHost instance. it is a facade interface allowing access to
	// all functionalities provided by dragonboat.
	nh, err := dragonboat.NewNodeHost(nhc)
	if err != nil {
		panic(err)
	}
	defer nh.Close()
	// start the first cluster
	// we use ExampleStateMachine as the IStateMachine for this cluster, its
	// behaviour is identical to the one used in the Hello World example.
	rc.ShardID = shardID1
	if err := nh.StartReplica(initialMembers, false, NewExampleStateMachine, rc); err != nil {
		fmt.Fprintf(os.Stderr, "failed to add cluster, %v\n", err)
		os.Exit(1)
	}
	// start the second cluster
	// we use SecondStateMachine as the IStateMachine for the second cluster
	rc.ShardID = shardID2
	if err := nh.StartReplica(initialMembers, false, NewSecondStateMachine, rc); err != nil {
		fmt.Fprintf(os.Stderr, "failed to add cluster, %v\n", err)
		os.Exit(1)
	}
	raftStopper := syncutil.NewStopper()
	consoleStopper := syncutil.NewStopper()
	ch := make(chan string, 16)
	consoleStopper.RunWorker(func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			s, err := reader.ReadString('\n')
			if err != nil {
				close(ch)
				return
			}
			if s == "exit\n" {
				raftStopper.Stop()
				// no data will be lost/corrupted if nodehost.Stop() is not called
				nh.Close()
				return
			}
			ch <- s
		}
	})
	raftStopper.RunWorker(func() {
		// use NO-OP client session here
		// check the example in godoc to see how to use a regular client session
		cs1 := nh.GetNoOPSession(shardID1)
		cs2 := nh.GetNoOPSession(shardID2)
		for {
			select {
			case v, ok := <-ch:
				if !ok {
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				// remove the \n char
				msg := strings.Replace(strings.TrimSpace(v), "\n", "", 1)
				var err error
				// In this example, the strategy on how data is sharded across different
				// Raft groups is based on whether the input message ends with a "?".
				// In your application, you are free to choose strategies suitable for
				// your application.
				if strings.HasSuffix(msg, "?") {
					// user message ends with "?", make a proposal to update the second
					// raft group
					_, err = nh.SyncPropose(ctx, cs2, []byte(msg))
				} else {
					// message not ends with "?", make a proposal to update the first
					// raft group
					_, err = nh.SyncPropose(ctx, cs1, []byte(msg))
				}
				cancel()
				if err != nil {
					fmt.Fprintf(os.Stderr, "SyncPropose returned error %v\n", err)
				}
			case <-raftStopper.ShouldStop():
				return
			}
		}
	})
	raftStopper.Wait()
}
````

## File: multigroup/README.CHS.md
````markdown
# 示例3 - 多个Raft组 #

## 关于 ##
本示例展示如何使用多个Raft组。

## 编译 ##
执行下面的命令以编译示例程序：
```
cd $HOME/src/dragonboat-example
make multigroup
```

## 运行 ##
在同一主机上三个不同的终端terminals上启动三个示例程序的进程：

```
./example-multigroup -replicaid 1
```
```
./example-multigroup -replicaid 2
```
```
./example-multigroup -replicaid 3
```
这将组建两个三节点的Raft集群，每个节点都由上述命令行命令中-replicaid所指定的ReplicaID值来标示。为求简易，本示例被设定为需用三个节点且ReplicaID值必须为1, 2, 3。

与之前的helloworld示例一样，你可以在一个终端上输入一条消息，它将会被复制到别的节点上。在本示例中，如果你输入一个以问好"?"结尾的消息，那么它会被提议到第二个raft组中，其余不以"?"结尾的消息均提议至第一个raft组中。

本例中，我们仅使用两个Raft组，这是为了使得例子程序尽可能简单。在一个实际的应用中，用户可以轻易使用大量的Raft组。

## 重新开始 ##
所有保存的数据均位于example-data的子目录内，可以手工删除这个example-data目录从而重新开始本例程。

## 代码 ##
[main.go](main.go)含有main()入口函数，在里面我们实例化了一个NodeHost实例，把所创建的两个Raft集群节点加入了这个实例。它同时实现了根据用户输入不同而对不同Raft组来提交提议的逻辑。注意代码中各阶段是如何指定ShardID值的。
````

## File: multigroup/README.md
````markdown
# Example 3 - Multiple Raft Groups #

## About ##
This example aims to give you an overview on how to use multiple raft groups in Dragonboat.

## Build ##
To build the executable -
```
cd $HOME/src/dragonboat-example
make multigroup
```

## Let's try it ##
Start three instances of the example program on the same machine in three different terminals:

```
./example-multigroup -replicaid 1
```
```
./example-multigroup -replicaid 2
```
```
./example-multigroup -replicaid 3
```
This forms two Raft groups each with 3 nodes, nodes are identified by the ReplicaID values specified on the command line while two Raft groups are identified by the ShardID values harded coded in main.go. For simplicity, this example program has been hardcoded to use 3 nodes for each raft group and their replica id values are 1, 2 and 3.

Similar to the previous helloworld example, you can type in a message in one of the terminals and your input message will be replicated to other nodes. In this example, if you type something ends with a question mark "?", then that particular message is going to be proposed to the second raft group, while messages without the question mark at the end are proposed to the first raft group. 

Note that we use two Raft groups here for simplicity, in a real world application, users can scale to much larger number of Raft groups.  

## Start Over ##
All saved data is saved into the example-data folder, you can delete this example-data folder and restart all processes to start over again.

## Code! ##
[main.go](main.go) contains the main() function, it is the place where we instantiated the NodeHost instance, added the created two raft groups to it. It also implements the logic of making proposals to one of the two available raft groups based on user input. Note how ShardID value is specified during different stages of this demo.
````

## File: multigroup/statemachine.go
````go
// Copyright 2017,2018 Lei Ni (nilei81@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	sm "github.com/lni/dragonboat/v4/statemachine"
)

// ExampleStateMachine is the IStateMachine implementation used in the example
// for handling all inputs not ends with "?".
// See https://github.com/lni/dragonboat/blob/master/statemachine/rsm.go for
// more details of the IStateMachine interface.
type ExampleStateMachine struct {
	ShardID   uint64
	ReplicaID uint64
	Count     uint64
}

// NewExampleStateMachine creates and return a new ExampleStateMachine object.
func NewExampleStateMachine(shardID uint64, replicaID uint64) sm.IStateMachine {
	return &ExampleStateMachine{
		ShardID:   shardID,
		ReplicaID: replicaID,
		Count:     0,
	}
}

// Lookup performs local lookup on the ExampleStateMachine instance. In this example,
// we always return the Count value as a little endian binary encoded byte
// slice.
func (s *ExampleStateMachine) Lookup(query interface{}) (interface{}, error) {
	result := make([]byte, 8)
	binary.LittleEndian.PutUint64(result, s.Count)
	return result, nil
}

// Update updates the object using the specified committed raft entry.
func (s *ExampleStateMachine) Update(e sm.Entry) (sm.Result, error) {
	// in this example, we print out the following message for each
	// incoming update request. we also increase the counter by one to remember
	// how many updates we have applied
	s.Count++
	fmt.Printf("from ExampleStateMachine.Update(), msg: %s, count:%d\n",
		string(e.Cmd), s.Count)
	return sm.Result{Value: uint64(len(e.Cmd))}, nil
}

// SaveSnapshot saves the current IStateMachine state into a snapshot using the
// specified io.Writer object.
func (s *ExampleStateMachine) SaveSnapshot(w io.Writer,
	fc sm.ISnapshotFileCollection, done <-chan struct{}) error {
	// as shown above, the only state that can be saved is the Count variable
	// there is no external file in this IStateMachine example, we thus leave
	// the fc untouched
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, s.Count)
	_, err := w.Write(data)
	return err
}

// RecoverFromSnapshot recovers the state using the provided snapshot.
func (s *ExampleStateMachine) RecoverFromSnapshot(r io.Reader,
	files []sm.SnapshotFile, done <-chan struct{}) error {
	// restore the Count variable, that is the only state we maintain in this
	// example, the input files is expected to be empty
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	v := binary.LittleEndian.Uint64(data)
	s.Count = v
	return nil
}

// Close closes the IStateMachine instance. There is nothing for us to cleanup
// or release as this is a pure in memory data store. Note that the Close
// method is not guaranteed to be called as node can crash at any time.
func (s *ExampleStateMachine) Close() error { return nil }
````

## File: multigroup/statemachine2.go
````go
// Copyright 2017,2018 Lei Ni (nilei81@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	sm "github.com/lni/dragonboat/v4/statemachine"
)

// SecondStateMachine is the IStateMachine implementation used in the
// multigroup example for handling all inputs ends with "?".
// See https://github.com/lni/dragonboat/blob/master/statemachine/rsm.go for
// more details of the IStateMachine interface.
// The behaviour of SecondStateMachine is similar to the ExampleStateMachine.
// The biggest difference is that its Update() method has different print
// out messages. See Update() for details.
type SecondStateMachine struct {
	ShardID   uint64
	ReplicaID uint64
	Count     uint64
}

// NewSecondStateMachine creates and return a new SecondStateMachine object.
func NewSecondStateMachine(shardID uint64, replicaID uint64) sm.IStateMachine {
	return &SecondStateMachine{
		ShardID:   shardID,
		ReplicaID: replicaID,
		Count:     0,
	}
}

// Lookup performs local lookup on the SecondStateMachine instance. In this example,
// we always return the Count value as a little endian binary encoded byte
// slice.
func (s *SecondStateMachine) Lookup(query interface{}) (interface{}, error) {
	result := make([]byte, 8)
	binary.LittleEndian.PutUint64(result, s.Count)
	return result, nil
}

// Update updates the object using the specified committed raft entry.
func (s *SecondStateMachine) Update(e sm.Entry) (sm.Result, error) {
	// in this example, we regard the input as a question.
	s.Count++
	fmt.Printf("got a question from user: %s, count:%d\n", string(e.Cmd), s.Count)
	return sm.Result{Value: uint64(len(e.Cmd))}, nil
}

// SaveSnapshot saves the current IStateMachine state into a snapshot using the
// specified io.Writer object.
func (s *SecondStateMachine) SaveSnapshot(w io.Writer,
	fc sm.ISnapshotFileCollection, done <-chan struct{}) error {
	// as shown above, the only state that can be saved is the Count variable
	// there is no external file in this IStateMachine example, we thus leave
	// the fc untouched
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, s.Count)
	_, err := w.Write(data)
	return err
}

// RecoverFromSnapshot recovers the state using the provided snapshot.
func (s *SecondStateMachine) RecoverFromSnapshot(r io.Reader,
	files []sm.SnapshotFile, done <-chan struct{}) error {
	// restore the Count variable, that is the only state we maintain in this
	// example, the input files is expected to be empty
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	v := binary.LittleEndian.Uint64(data)
	s.Count = v
	return nil
}

// Close closes the IStateMachine instance. There is nothing for us to cleanup
// or release as this is a pure in memory data store. Note that the Close
// method is not guaranteed to be called as node can crash at any time.
func (s *SecondStateMachine) Close() error { return nil }
````

## File: ondisk/diskkv.go
````go
// Copyright 2017-2019 Lei Ni (nilei81@gmail.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/cockroachdb/pebble"

	sm "github.com/lni/dragonboat/v4/statemachine"
)

const (
	appliedIndexKey    string = "disk_kv_applied_index"
	testDBDirName      string = "example-data"
	currentDBFilename  string = "current"
	updatingDBFilename string = "current.updating"
)

//
// Note: this is an example demonstrating how to use the on disk state machine
// feature in Dragonboat. it assumes the underlying db only supports Get, Put
// and TakeSnapshot operations. this is not a demonstration on how to build a
// distributed key-value database.
//

func syncDir(dir string) (err error) {
	if runtime.GOOS == "windows" {
		return nil
	}
	fileInfo, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		panic("not a dir")
	}
	df, err := os.Open(filepath.Clean(dir))
	if err != nil {
		return err
	}
	defer func() {
		if cerr := df.Close(); err == nil {
			err = cerr
		}
	}()
	return df.Sync()
}

type KVData struct {
	Key string
	Val string
}

// pebbledb is a wrapper to ensure lookup() and close() can be concurrently
// invoked. IOnDiskStateMachine.Update() and close() will never be concurrently
// invoked.
type pebbledb struct {
	mu     sync.RWMutex
	db     *pebble.DB
	ro     *pebble.IterOptions
	wo     *pebble.WriteOptions
	syncwo *pebble.WriteOptions
	closed bool
}

func (r *pebbledb) lookup(query []byte) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return nil, errors.New("db already closed")
	}
	val, closer, err := r.db.Get(query)
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	if len(val) == 0 {
		return []byte(""), nil
	}
	buf := make([]byte, len(val))
	copy(buf, val)
	return buf, nil
}

func (r *pebbledb) close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	if r.db != nil {
		r.db.Close()
	}
}

// createDB creates a PebbleDB DB in the specified directory.
func createDB(dbdir string) (*pebbledb, error) {
	ro := &pebble.IterOptions{}
	wo := &pebble.WriteOptions{Sync: false}
	syncwo := &pebble.WriteOptions{Sync: true}
	cache := pebble.NewCache(0)
	opts := &pebble.Options{
		MaxManifestFileSize: 1024 * 32,
		MemTableSize:        1024 * 32,
		Cache:               cache,
	}
	if err := os.MkdirAll(dbdir, 0755); err != nil {
		return nil, err
	}
	db, err := pebble.Open(dbdir, opts)
	if err != nil {
		return nil, err
	}
	cache.Unref()
	return &pebbledb{
		db:     db,
		ro:     ro,
		wo:     wo,
		syncwo: syncwo,
	}, nil
}

// functions below are used to manage the current data directory of Pebble DB.
func isNewRun(dir string) bool {
	fp := filepath.Join(dir, currentDBFilename)
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		return true
	}
	return false
}

func getNodeDBDirName(clusterID uint64, nodeID uint64) string {
	part := fmt.Sprintf("%d_%d", clusterID, nodeID)
	return filepath.Join(testDBDirName, part)
}

func getNewRandomDBDirName(dir string) string {
	part := "%d_%d"
	rn := rand.Uint64()
	ct := time.Now().UnixNano()
	return filepath.Join(dir, fmt.Sprintf(part, rn, ct))
}

func replaceCurrentDBFile(dir string) error {
	fp := filepath.Join(dir, currentDBFilename)
	tmpFp := filepath.Join(dir, updatingDBFilename)
	if err := os.Rename(tmpFp, fp); err != nil {
		return err
	}
	return syncDir(dir)
}

func saveCurrentDBDirName(dir string, dbdir string) error {
	h := md5.New()
	if _, err := h.Write([]byte(dbdir)); err != nil {
		return err
	}
	fp := filepath.Join(dir, updatingDBFilename)
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
		if err := syncDir(dir); err != nil {
			panic(err)
		}
	}()
	if _, err := f.Write(h.Sum(nil)[:8]); err != nil {
		return err
	}
	if _, err := f.Write([]byte(dbdir)); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return nil
}

func getCurrentDBDirName(dir string) (string, error) {
	fp := filepath.Join(dir, currentDBFilename)
	f, err := os.OpenFile(fp, os.O_RDONLY, 0755)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	if len(data) <= 8 {
		panic("corrupted content")
	}
	crc := data[:8]
	content := data[8:]
	h := md5.New()
	if _, err := h.Write(content); err != nil {
		return "", err
	}
	if !bytes.Equal(crc, h.Sum(nil)[:8]) {
		panic("corrupted content with not matched crc")
	}
	return string(content), nil
}

func createNodeDataDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return syncDir(filepath.Dir(dir))
}

func cleanupNodeDataDir(dir string) error {
	os.RemoveAll(filepath.Join(dir, updatingDBFilename))
	dbdir, err := getCurrentDBDirName(dir)
	if err != nil {
		return err
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, fi := range files {
		if !fi.IsDir() {
			continue
		}
		fmt.Printf("dbdir %s, fi.name %s, dir %s\n", dbdir, fi.Name(), dir)
		toDelete := filepath.Join(dir, fi.Name())
		if toDelete != dbdir {
			fmt.Printf("removing %s\n", toDelete)
			if err := os.RemoveAll(toDelete); err != nil {
				return err
			}
		}
	}
	return nil
}

// DiskKV is a state machine that implements the IOnDiskStateMachine interface.
// DiskKV stores key-value pairs in the underlying PebbleDB key-value store. As
// it is used as an example, it is implemented using the most basic features
// common in most key-value stores. This is NOT a benchmark program.
type DiskKV struct {
	clusterID   uint64
	nodeID      uint64
	lastApplied uint64
	db          unsafe.Pointer
	closed      bool
	aborted     bool
}

// NewDiskKV creates a new disk kv test state machine.
func NewDiskKV(clusterID uint64, nodeID uint64) sm.IOnDiskStateMachine {
	d := &DiskKV{
		clusterID: clusterID,
		nodeID:    nodeID,
	}
	return d
}

func (d *DiskKV) queryAppliedIndex(db *pebbledb) (uint64, error) {
	val, closer, err := db.db.Get([]byte(appliedIndexKey))
	if err != nil && err != pebble.ErrNotFound {
		return 0, err
	}
	defer func() {
		if closer != nil {
			closer.Close()
		}
	}()
	if len(val) == 0 {
		return 0, nil
	}
	return binary.LittleEndian.Uint64(val), nil
}

// Open opens the state machine and return the index of the last Raft Log entry
// already updated into the state machine.
func (d *DiskKV) Open(stopc <-chan struct{}) (uint64, error) {
	dir := getNodeDBDirName(d.clusterID, d.nodeID)
	if err := createNodeDataDir(dir); err != nil {
		panic(err)
	}
	var dbdir string
	if !isNewRun(dir) {
		if err := cleanupNodeDataDir(dir); err != nil {
			return 0, err
		}
		var err error
		dbdir, err = getCurrentDBDirName(dir)
		if err != nil {
			return 0, err
		}
		if _, err := os.Stat(dbdir); err != nil {
			if os.IsNotExist(err) {
				panic("db dir unexpectedly deleted")
			}
		}
	} else {
		dbdir = getNewRandomDBDirName(dir)
		if err := saveCurrentDBDirName(dir, dbdir); err != nil {
			return 0, err
		}
		if err := replaceCurrentDBFile(dir); err != nil {
			return 0, err
		}
	}
	db, err := createDB(dbdir)
	if err != nil {
		return 0, err
	}
	atomic.SwapPointer(&d.db, unsafe.Pointer(db))
	appliedIndex, err := d.queryAppliedIndex(db)
	if err != nil {
		panic(err)
	}
	d.lastApplied = appliedIndex
	return appliedIndex, nil
}

// Lookup queries the state machine.
func (d *DiskKV) Lookup(key interface{}) (interface{}, error) {
	db := (*pebbledb)(atomic.LoadPointer(&d.db))
	if db != nil {
		v, err := db.lookup(key.([]byte))
		if err == nil && d.closed {
			panic("lookup returned valid result when DiskKV is already closed")
		}
		if err == pebble.ErrNotFound {
			return v, nil
		}
		return v, err
	}
	return nil, errors.New("db closed")
}

// Update updates the state machine. In this example, all updates are put into
// a PebbleDB write batch and then atomically written to the DB together with
// the index of the last Raft Log entry. For simplicity, we always Sync the
// writes (db.wo.Sync=True). To get higher throughput, you can implement the
// Sync() method below and choose not to synchronize for every Update(). Sync()
// will periodically called by Dragonboat to synchronize the state.
func (d *DiskKV) Update(ents []sm.Entry) ([]sm.Entry, error) {
	if d.aborted {
		panic("update() called after abort set to true")
	}
	if d.closed {
		panic("update called after Close()")
	}
	db := (*pebbledb)(atomic.LoadPointer(&d.db))
	wb := db.db.NewBatch()
	defer wb.Close()
	for idx, e := range ents {
		dataKV := &KVData{}
		if err := json.Unmarshal(e.Cmd, dataKV); err != nil {
			panic(err)
		}
		wb.Set([]byte(dataKV.Key), []byte(dataKV.Val), db.wo)
		ents[idx].Result = sm.Result{Value: uint64(len(ents[idx].Cmd))}
	}
	// save the applied index to the DB.
	appliedIndex := make([]byte, 8)
	binary.LittleEndian.PutUint64(appliedIndex, ents[len(ents)-1].Index)
	wb.Set([]byte(appliedIndexKey), appliedIndex, db.wo)
	if err := db.db.Apply(wb, db.syncwo); err != nil {
		return nil, err
	}
	if d.lastApplied >= ents[len(ents)-1].Index {
		panic("lastApplied not moving forward")
	}
	d.lastApplied = ents[len(ents)-1].Index
	return ents, nil
}

// Sync synchronizes all in-core state of the state machine. Since the Update
// method in this example already does that every time when it is invoked, the
// Sync method here is a NoOP.
func (d *DiskKV) Sync() error {
	return nil
}

type diskKVCtx struct {
	db       *pebbledb
	snapshot *pebble.Snapshot
}

// PrepareSnapshot prepares snapshotting. PrepareSnapshot is responsible to
// capture a state identifier that identifies a point in time state of the
// underlying data. In this example, we use Pebble's snapshot feature to
// achieve that.
func (d *DiskKV) PrepareSnapshot() (interface{}, error) {
	if d.closed {
		panic("prepare snapshot called after Close()")
	}
	if d.aborted {
		panic("prepare snapshot called after abort")
	}
	db := (*pebbledb)(atomic.LoadPointer(&d.db))
	return &diskKVCtx{
		db:       db,
		snapshot: db.db.NewSnapshot(),
	}, nil
}

func iteratorIsValid(iter *pebble.Iterator) bool {
	return iter.Valid()
}

// saveToWriter saves all existing key-value pairs to the provided writer.
// As an example, we use the most straight forward way to implement this.
func (d *DiskKV) saveToWriter(db *pebbledb,
	ss *pebble.Snapshot, w io.Writer) error {
	iter := ss.NewIter(db.ro)
	defer iter.Close()
	values := make([]*KVData, 0)
	for iter.First(); iteratorIsValid(iter); iter.Next() {
		kv := &KVData{
			Key: string(iter.Key()),
			Val: string(iter.Value()),
		}
		values = append(values, kv)
	}
	count := uint64(len(values))
	sz := make([]byte, 8)
	binary.LittleEndian.PutUint64(sz, count)
	if _, err := w.Write(sz); err != nil {
		return err
	}
	for _, dataKv := range values {
		data, err := json.Marshal(dataKv)
		if err != nil {
			panic(err)
		}
		binary.LittleEndian.PutUint64(sz, uint64(len(data)))
		if _, err := w.Write(sz); err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	return nil
}

// SaveSnapshot saves the state machine state identified by the state
// identifier provided by the input ctx parameter. Note that SaveSnapshot
// is not suppose to save the latest state.
func (d *DiskKV) SaveSnapshot(ctx interface{},
	w io.Writer, done <-chan struct{}) error {
	if d.closed {
		panic("prepare snapshot called after Close()")
	}
	if d.aborted {
		panic("prepare snapshot called after abort")
	}
	ctxdata := ctx.(*diskKVCtx)
	db := ctxdata.db
	db.mu.RLock()
	defer db.mu.RUnlock()
	ss := ctxdata.snapshot
	defer ss.Close()
	return d.saveToWriter(db, ss, w)
}

// RecoverFromSnapshot recovers the state machine state from snapshot. The
// snapshot is recovered into a new DB first and then atomically swapped with
// the existing DB to complete the recovery.
func (d *DiskKV) RecoverFromSnapshot(r io.Reader,
	done <-chan struct{}) error {
	if d.closed {
		panic("recover from snapshot called after Close()")
	}
	dir := getNodeDBDirName(d.clusterID, d.nodeID)
	dbdir := getNewRandomDBDirName(dir)
	oldDirName, err := getCurrentDBDirName(dir)
	if err != nil {
		return err
	}
	db, err := createDB(dbdir)
	if err != nil {
		return err
	}
	sz := make([]byte, 8)
	if _, err := io.ReadFull(r, sz); err != nil {
		return err
	}
	total := binary.LittleEndian.Uint64(sz)
	wb := db.db.NewBatch()
	defer wb.Close()
	for i := uint64(0); i < total; i++ {
		if _, err := io.ReadFull(r, sz); err != nil {
			return err
		}
		toRead := binary.LittleEndian.Uint64(sz)
		data := make([]byte, toRead)
		if _, err := io.ReadFull(r, data); err != nil {
			return err
		}
		dataKv := &KVData{}
		if err := json.Unmarshal(data, dataKv); err != nil {
			panic(err)
		}
		wb.Set([]byte(dataKv.Key), []byte(dataKv.Val), db.wo)
	}
	if err := db.db.Apply(wb, db.syncwo); err != nil {
		return err
	}
	if err := saveCurrentDBDirName(dir, dbdir); err != nil {
		return err
	}
	if err := replaceCurrentDBFile(dir); err != nil {
		return err
	}
	newLastApplied, err := d.queryAppliedIndex(db)
	if err != nil {
		panic(err)
	}
	// when d.lastApplied == newLastApplied, it probably means there were some
	// dummy entries or membership change entries as part of the new snapshot
	// that never reached the SM and thus never moved the last applied index
	// in the SM snapshot.
	if d.lastApplied > newLastApplied {
		panic("last applied not moving forward")
	}
	d.lastApplied = newLastApplied
	old := (*pebbledb)(atomic.SwapPointer(&d.db, unsafe.Pointer(db)))
	if old != nil {
		old.close()
	}
	parent := filepath.Dir(oldDirName)
	if err := os.RemoveAll(oldDirName); err != nil {
		return err
	}
	return syncDir(parent)
}

// Close closes the state machine.
func (d *DiskKV) Close() error {
	db := (*pebbledb)(atomic.SwapPointer(&d.db, unsafe.Pointer(nil)))
	if db != nil {
		d.closed = true
		db.close()
	} else {
		if d.closed {
			panic("close called twice")
		}
	}
	return nil
}
````

## File: ondisk/main.go
````go
// Copyright 2017,2018 Lei Ni (nilei81@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
ondisk is an example program for dragonboat's on disk state machine.
*/
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/logger"
	"github.com/lni/goutils/syncutil"
)

type RequestType uint64

const (
	exampleShardID uint64 = 128
)

const (
	PUT RequestType = iota
	GET
)

var (
	// initial nodes count is fixed to three, their addresses are also fixed
	addresses = []string{
		"localhost:63001",
		"localhost:63002",
		"localhost:63003",
	}
)

func parseCommand(msg string) (RequestType, string, string, bool) {
	parts := strings.Split(strings.TrimSpace(msg), " ")
	if len(parts) == 0 || (parts[0] != "put" && parts[0] != "get") {
		return PUT, "", "", false
	}
	if parts[0] == "put" {
		if len(parts) != 3 {
			return PUT, "", "", false
		}
		return PUT, parts[1], parts[2], true
	}
	if len(parts) != 2 {
		return GET, "", "", false
	}
	return GET, parts[1], "", true
}

func printUsage() {
	fmt.Fprintf(os.Stdout, "Usage - \n")
	fmt.Fprintf(os.Stdout, "put key value\n")
	fmt.Fprintf(os.Stdout, "get key\n")
}

func main() {
	replicaID := flag.Int("replicaid", 1, "ReplicaID to use")
	addr := flag.String("addr", "", "Nodehost address")
	join := flag.Bool("join", false, "Joining a new node")
	flag.Parse()
	if len(*addr) == 0 && *replicaID != 1 && *replicaID != 2 && *replicaID != 3 {
		fmt.Fprintf(os.Stderr, "replica id must be 1, 2 or 3 when address is not specified\n")
		os.Exit(1)
	}
	// https://github.com/golang/go/issues/17393
	if runtime.GOOS == "darwin" {
		signal.Ignore(syscall.Signal(0xd))
	}
	initialMembers := make(map[uint64]string)
	if !*join {
		for idx, v := range addresses {
			initialMembers[uint64(idx+1)] = v
		}
	}
	var nodeAddr string
	if len(*addr) != 0 {
		nodeAddr = *addr
	} else {
		nodeAddr = initialMembers[uint64(*replicaID)]
	}
	fmt.Fprintf(os.Stdout, "node address: %s\n", nodeAddr)
	logger.GetLogger("raft").SetLevel(logger.ERROR)
	logger.GetLogger("rsm").SetLevel(logger.WARNING)
	logger.GetLogger("transport").SetLevel(logger.WARNING)
	logger.GetLogger("grpc").SetLevel(logger.WARNING)
	rc := config.Config{
		ReplicaID:          uint64(*replicaID),
		ShardID:            exampleShardID,
		ElectionRTT:        10,
		HeartbeatRTT:       1,
		CheckQuorum:        true,
		SnapshotEntries:    10,
		CompactionOverhead: 5,
	}
	datadir := filepath.Join(
		"example-data",
		"helloworld-data",
		fmt.Sprintf("node%d", *replicaID))
	nhc := config.NodeHostConfig{
		WALDir:         datadir,
		NodeHostDir:    datadir,
		RTTMillisecond: 200,
		RaftAddress:    nodeAddr,
	}
	nh, err := dragonboat.NewNodeHost(nhc)
	if err != nil {
		panic(err)
	}
	if err := nh.StartOnDiskReplica(initialMembers, *join, NewDiskKV, rc); err != nil {
		fmt.Fprintf(os.Stderr, "failed to add cluster, %v\n", err)
		os.Exit(1)
	}
	raftStopper := syncutil.NewStopper()
	consoleStopper := syncutil.NewStopper()
	ch := make(chan string, 16)
	consoleStopper.RunWorker(func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			s, err := reader.ReadString('\n')
			if err != nil {
				close(ch)
				return
			}
			if s == "exit\n" {
				raftStopper.Stop()
				nh.Close()
				return
			}
			ch <- s
		}
	})
	printUsage()
	raftStopper.RunWorker(func() {
		cs := nh.GetNoOPSession(exampleShardID)
		for {
			select {
			case v, ok := <-ch:
				if !ok {
					return
				}
				msg := strings.Replace(v, "\n", "", 1)
				// input message must be in the following formats -
				// put key value
				// get key
				rt, key, val, ok := parseCommand(msg)
				if !ok {
					fmt.Fprintf(os.Stderr, "invalid input\n")
					printUsage()
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				if rt == PUT {
					kv := &KVData{
						Key: key,
						Val: val,
					}
					data, err := json.Marshal(kv)
					if err != nil {
						panic(err)
					}
					_, err = nh.SyncPropose(ctx, cs, data)
					if err != nil {
						fmt.Fprintf(os.Stderr, "SyncPropose returned error %v\n", err)
					}
				} else {
					result, err := nh.SyncRead(ctx, exampleShardID, []byte(key))
					if err != nil {
						fmt.Fprintf(os.Stderr, "SyncRead returned error %v\n", err)
					} else {
						fmt.Fprintf(os.Stdout, "query key: %s, result: %s\n", key, result)
					}
				}
				cancel()
			case <-raftStopper.ShouldStop():
				return
			}
		}
	})
	raftStopper.Wait()
}
````

## File: ondisk/README.CHS.md
````markdown
# 示例4 基于磁盘的状态机 #

## 关于 ##
本例用一个基于(Pebble)[https://github.com/cockroachdb/pebble]的分布式Key-Value数据库来展示Dragonboat的基于磁盘的状态机支持。

## 编译 ##
用下列命令编译本例的可执行文件 - 
```
cd $HOME/src/dragonboat-example
make ondisk
```

## 运行 ##
使用下列命令在同一台计算机的三个终端上启动三个本例程的实例。

```
./example-ondisk -replicaid 1
```
```
./example-ondisk -replicaid 2
```
```
./example-ondisk -replicaid 3
```

这将组建一个三节点的Raft集群，每个节点都由上述命令行命令中-replicaid所指定的ReplicaID值来标示。为求简易，本示例被
设定为需用三个节点且ReplicaID值必须为1, 2, 3。

您可以以下列两种格式输入一个命令以使用本例程 -

```
put key value
```
或 
```
get key
```

第一个命令将所指定的输入值Value设入所指定的键值Key，第二个命令通过查询底层的基于磁盘的状态机以返回键值Key所指向的值。

## 重新开始 ##
所有保存的数据均位于example-data的子目录内，可以手工删除这个example-data目录从而重新开始本例程。

## 代码 ##
在[diskkv.go](diskkv.go)中，DiskKV类型实现了statemachine.IOnDiskStateMachine这一接口。它使用Pebble作为存储引擎以在磁盘上存储所有状态机内容，这使它无需在每次重启后用快照Snapshot和已保存的Raft Log来恢复状态。这同时使得状态机管理下的数据量受磁盘大小制约，而不再受限于内存大小。

statemachine.IOnDiskStateMachine接口中的Open方法用以打开一个在磁盘上已存在的状态机并返回其最后一个已处理的Raft Log的index值。所有的实现了statemachine.IOnDiskStateMachine接口的类型均必须在保存状态机状态的同时，原子的同时保存其最后一个已处理的Raft Log的index值。内存那缓存的状态与磁盘同步，比如通过使用fsync()。本例中，我们始终使用Pebble的WriteBatch来原子地写入多个记录到底层的Pebble数据库中，包括最后一个已处理的Raft Log的index值。fsync()也始终在每次写以后被调用。

与基于statemachine.IStateMachine的状态机相比较，本例另一主要区别在于基于statemachine.IOnDiskStateMachine的状态机支持并发的读写。在状态机正在被Update更新时，Lookup和SaveSnapshot方法可以被同时并发的调用。而Lookup方法也可以在状态机正在被RecoverFromSnapshot方法恢复的时候被并发调用。

为了支持这样的并发访问，状态机快照的产生方法也与基于statemachine.IStateMachine的状态机有所不同。PrepareSnapshot方法首先被执行，它保存一个称为状态ID的对象，它用来标示状态机在某一具体时间点的状态。本例中，我们使用Pebble的快照功能来创建这样一个状态ID，并将其作为所产生的diskKVCtx对象的一部分返回。Update方法不会与PrepareSnapshot方法并发执行。接着，SaveSnapshot方法便可以与Update方法并发的执行了，SaveSnapshot会根据所提供的状态ID来产生状态机快照。本例中，我们遍历Pebble的快照中所涵盖的所有key-value对并将它们写入所提供的io.Writer中。

请参考[diskkv.go](diskkv.go)代码了解更详细的实现。

main函数在[main.go](main.go)中，它是我们创建NodeHost对象并启动所用的Raft组的地方。用户输入也在其中被处理，以允许用户通过put和get命令操作key-value数据。
````

## File: ondisk/README.md
````markdown
# Example 4 - On Disk State Machine #

## About ##
This example uses a [Pebble](https://github.com/cockroachdb/pebble) based distributed key-value store to demonstrate the on disk state machine support in Dragonboat.

## Build ##
To build the executable -
```
cd $HOME/src/dragonboat-example
make ondisk
```

## Run This Example ##
Start three instances of the example program on the same machine in three different terminals:

```
./example-ondisk -replicaid 1
```
```
./example-ondisk -replicaid 2
```
```
./example-ondisk -replicaid 3
```
This forms a Raft group with 3 replicas, each of them is identified by the ReplicaID specified on the command line. For simplicity, this example program has been hardcoded to use 3 nodes and their node id values are 1, 2 and 3.

You can type in a message in one of the two following formats - 
```
put key value
```
or 
```
get key
```

The first command above sets the specified input value to key, the second command queries the underlying on disk state machine and returns the value associated with key. 

## Start Over ##
All saved data is saved into the example-data folder, you can delete this example-data folder and restart all processes to start over again.

## Code ##
In [diskkv.go](diskkv.go), the DiskKV struct implements the statemachine.IOnDiskStateMachine interface. It employs Pebble as its on disk storage engine to store all state machine managed data, it thus doesn't need to be restored from snapshot or saved Raft logs after each reboot. This also ensures that the total amount of data that can be managed by the state machine is limited by available disk capacity rather than memory size. 

The Open method of the statemachine.IOnDiskStateMachine interface opens existing on disk state machine and returns the index of the last updated Raft log entry. It is important for all statemachine.IOnDiskStateMachine implmentations to atomically persist the index of the last updated Raft log entry together with the outcome of the update operation when updating such on disk state machines. In-core state should also be synchronized with disk, e.g. using fsync(). In this example, we always use Pebble's WriteBatch type to atomically write incoming records, including the the index of the last updated Raft log entry, to the underlying Pebble database. fsync() is invoked by Pebble at the end of each write.

Compared with statemachine.IStateMachine based state machine, another major difference is that concurrent read and write are supported by statemachine.IOnDiskStateMachine based on disk state machines. The Lookup and the SaveSnapshot method can be concurrently invoked when the state machine is being updated by the Update method. The Lookup method can also be invoked when the state machine is being resotred by the RecoverFromSnapshot method. 

To support the above described concurrent access to statemachine.IOnDiskStateMachine types, the way how state machine snapshot is saved is also different from previously described statemachine.IStateMachine types. The PrepareSnapshot method will be first invoked to capture and return a so called state identifier object that can describe the point in time state of the state machine. In this example, we take a Pebble snapshot and return it as a part of the generated diskKVCtx instance. Update is not allowed by the system when PrepareSnapshot is being invoked. SaveSnapshot is then invoked concurrent to the Update method to actually save the point in time state of the state machine identified by the provided state identifier. In this example, we iterate over all key-value pairs covered in the Pebble snapshot and write all of them to the provided io.Writer.

See godoc in [diskkv.go](diskkv.go) for more detials.

The main function can be found in [main.go](main.go), it is the place where we instantiated the NodeHost instance, added the created example Ragroupp to it. User inputs are also handled here to allow users to put or get key-value pairs.
````

## File: optimistic-write-lock/fsm.go
````go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	dbsm "github.com/lni/dragonboat/v4/statemachine"
)

const (
	ResultCodeFailure = iota
	ResultCodeSuccess
	ResultCodeVersionMismatch
)

type Query struct {
	Key string
}

type Entry struct {
	Key string `json:"key"`
	Ver uint64 `json:"ver"`
	Val string `json:"val"`
}

func NewLinearizableFSM() dbsm.CreateConcurrentStateMachineFunc {
	return dbsm.CreateConcurrentStateMachineFunc(func(shardID, replicaID uint64) dbsm.IConcurrentStateMachine {
		return &linearizableFSM{
			shardID:   shardID,
			replicaID: replicaID,
			data:      map[string]interface{}{},
		}
	})
}

type linearizableFSM struct {
	shardID   uint64
	replicaID uint64
	data      map[string]interface{}
}

func (fsm *linearizableFSM) Update(entries []dbsm.Entry) ([]dbsm.Entry, error) {
	for i, ent := range entries {
		var entry Entry
		if err := json.Unmarshal(ent.Cmd, &entry); err != nil {
			return entries, fmt.Errorf("Invalid entry %#v, %w", ent, err)
		}
		if v, ok := fsm.data[entry.Key]; ok {
			// Reject entries with mismatched versions
			if v.(Entry).Ver != entry.Ver {
				data, _ := json.Marshal(v)
				entries[i].Result = dbsm.Result{
					Value: ResultCodeVersionMismatch,
					Data:  data,
				}
				continue
			}
		}
		entry.Ver = ent.Index
		fsm.data[entry.Key] = entry
		b, _ := json.Marshal(entry)
		entries[i].Result = dbsm.Result{
			Value: ResultCodeSuccess,
			Data:  b,
		}
	}

	return entries, nil
}

func (fsm *linearizableFSM) Lookup(e interface{}) (val interface{}, err error) {
	query, ok := e.(Query)
	if !ok {
		return nil, fmt.Errorf("Invalid query %#v", e)
	}
	val, _ = fsm.data[query.Key]

	return
}

func (fsm *linearizableFSM) PrepareSnapshot() (ctx interface{}, err error) {
	return
}

func (fsm *linearizableFSM) SaveSnapshot(ctx interface{}, w io.Writer, sfc dbsm.ISnapshotFileCollection, stopc <-chan struct{}) (err error) {
	b, err := json.Marshal(fsm.data)
	if err == nil {
		_, err = io.Copy(w, bytes.NewReader(b))
	}

	return
}

func (fsm *linearizableFSM) RecoverFromSnapshot(r io.Reader, sfc []dbsm.SnapshotFile, stopc <-chan struct{}) (err error) {
	return json.NewDecoder(r).Decode(fsm.data)
}

func (fsm *linearizableFSM) Close() (err error) {
	return
}
````

## File: optimistic-write-lock/handler.go
````go
// Copyright 2017,2018 Lei Ni (nilei81@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/lni/dragonboat/v4"
)

type handler struct {
	nh *dragonboat.NodeHost
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer w.Write([]byte("\n"))
	var err error
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	if r.Method == "GET" {
		query := Query{
			Key: r.URL.Path,
		}
		res, err := h.nh.SyncRead(ctx, shardID, query)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		if _, ok := res.(Entry); !ok {
			w.WriteHeader(404)
			w.Write([]byte("Not Found"))
			return
		}
		b, _ := json.Marshal(res.(Entry))
		w.WriteHeader(200)
		w.Write(b)
	} else if r.Method == "PUT" {
		var ver int
		if len(r.FormValue("ver")) > 0 {
			ver, err = strconv.Atoi(r.FormValue("ver"))
			if err != nil {
				w.WriteHeader(400)
				w.Write([]byte("Version must be uint64"))
				return
			}
		}
		var entry = Entry{
			Key: r.URL.Path,
			Ver: uint64(ver),
			Val: r.FormValue("val"),
		}
		b, err := json.Marshal(entry)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}
		res, err := h.nh.SyncPropose(ctx, h.nh.GetNoOPSession(shardID), b)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		if res.Value == ResultCodeFailure {
			w.WriteHeader(400)
			w.Write(res.Data)
			return
		}
		if res.Value == ResultCodeVersionMismatch {
			var result Entry
			json.Unmarshal(res.Data, &result)
			w.WriteHeader(409)
			w.Write([]byte(fmt.Sprintf("Version mismatch (%d != %d)", entry.Ver, result.Ver)))
			return
		}
		w.WriteHeader(200)
		w.Write(res.Data)
	} else {
		w.WriteHeader(405)
		w.Write([]byte("Method not supported"))
	}
}
````

## File: optimistic-write-lock/main.go
````go
// Copyright 2017,2018 Lei Ni (nilei81@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
linearizable is an example program for building a linearizable state machine using dragonboat.
*/
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
)

var (
	datadir = "/tmp/dragonboat-example-linearizable"
	members = map[uint64]string{
		1: "localhost:61001",
		2: "localhost:61002",
		3: "localhost:61003",
	}
	httpAddr = []string{
		":8001",
		":8002",
		":8003",
	}
	shardID uint64 = 128
)

func main() {
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)
	for i, nodeAddr := range members {
		dir := fmt.Sprintf("%s/%d", datadir, i)
		if err := os.MkdirAll(dir, 0777); err != nil {
			panic(err)
		}
		log.Printf("Starting node %s", nodeAddr)
		nh, err := dragonboat.NewNodeHost(config.NodeHostConfig{
			RaftAddress:    nodeAddr,
			NodeHostDir:    dir,
			RTTMillisecond: 100,
		})
		if err != nil {
			panic(err)
		}
		fsm := NewLinearizableFSM()
		err = nh.StartConcurrentReplica(members, false, fsm, config.Config{
			ReplicaID:          uint64(i),
			ShardID:            shardID,
			ElectionRTT:        10,
			HeartbeatRTT:       1,
			CheckQuorum:        true,
			SnapshotEntries:    10,
			CompactionOverhead: 5,
		})
		if err != nil {
			panic(err)
		}
		go func(s *http.Server) {
			log.Fatal(s.ListenAndServe())
		}(&http.Server{
			Addr:    httpAddr[i-1],
			Handler: &handler{nh},
		})
	}
	<-stop
}
````

## File: optimistic-write-lock/README.md
````markdown
## Optimistic Write Lock

This example illustrates use of optimistic write locks to implement a consistent finite state machine.

The example starts an HTTP server which performs queries on GET and proposes updates on PUT.

Any proposed update with an invalid version will be rejected.

Clients must read the version of the key and supply it during update in order to modify the value.

```go
> make run *.go
```

```
> curl -X PUT "http://localhost:8001/testkey?val=testvalue"
{"key":"/testkey","ver":6,"val":"testvalue"}

> curl -X PUT "http://localhost:8001/testkey?val=testvalue2"
Version mismatch (0 != 6)

> curl -X PUT "http://localhost:8001/testkey?val=testvalue2&ver=6"
{"key":"/testkey","ver":8,"val":"testvalue2"}

> curl -X PUT "http://localhost:8001/testkey?val=testvalue3&ver=6"
Version mismatch (6 != 8)

> curl -X PUT "http://localhost:8001/testkey?val=testvalue3&ver=8"
{"key":"/testkey","ver":10,"val":"testvalue3"}
```

Optimistic write locks can be used to implement [CP](https://en.wikipedia.org/wiki/CAP_theorem)
systems using dragonboat.

While this example demonstrates key-level write locks, similar write locks can be implemented
at any level of granularity within a finite state machine to linearize anything from individual
keys to entire datasets.
````

## File: .gitignore
````
example-data
/example-helloworld
/example-multigroup
/example-ondisk
/example-optimistic-write-lock
cpphelloworld/cpphelloworld
*.o
*.so
````

## File: go.mod
````
module github.com/lni/dragonboat-example/v3

require (
	github.com/cockroachdb/pebble v0.0.0-20221207173255-0f086d933dac
	github.com/lni/dragonboat/v4 v4.0.0-20230917160253-d9f49378cd2d
	github.com/lni/goutils v1.3.1-0.20220604063047-388d67b4dbc4
)

require (
	github.com/DataDog/zstd v1.4.5 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.1.2 // indirect
	github.com/VictoriaMetrics/metrics v1.18.1 // indirect
	github.com/armon/go-metrics v0.0.0-20180917152333-f0300d1749da // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cockroachdb/errors v1.9.0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20211118104740-dabe8e521a4f // indirect
	github.com/cockroachdb/redact v1.1.3 // indirect
	github.com/getsentry/sentry-go v0.12.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.0.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.3 // indirect
	github.com/hashicorp/go-multierror v1.0.0 // indirect
	github.com/hashicorp/go-sockaddr v1.0.0 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/hashicorp/memberlist v0.3.1 // indirect
	github.com/klauspost/compress v1.11.13 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lni/vfs v0.2.1-0.20220616104132-8852fd867376 // indirect
	github.com/miekg/dns v1.1.26 // indirect
	github.com/pierrec/lz4/v4 v4.1.14 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	github.com/valyala/fastrand v1.1.0 // indirect
	github.com/valyala/histogram v1.2.0 // indirect
	golang.org/x/crypto v0.13.0 // indirect
	golang.org/x/exp v0.0.0-20200513190911-00229845015e // indirect
	golang.org/x/net v0.15.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
)

go 1.17
````

## File: LICENSE
````
Apache License
                           Version 2.0, January 2004
                        http://www.apache.org/licenses/

   TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION

   1. Definitions.

      "License" shall mean the terms and conditions for use, reproduction,
      and distribution as defined by Sections 1 through 9 of this document.

      "Licensor" shall mean the copyright owner or entity authorized by
      the copyright owner that is granting the License.

      "Legal Entity" shall mean the union of the acting entity and all
      other entities that control, are controlled by, or are under common
      control with that entity. For the purposes of this definition,
      "control" means (i) the power, direct or indirect, to cause the
      direction or management of such entity, whether by contract or
      otherwise, or (ii) ownership of fifty percent (50%) or more of the
      outstanding shares, or (iii) beneficial ownership of such entity.

      "You" (or "Your") shall mean an individual or Legal Entity
      exercising permissions granted by this License.

      "Source" form shall mean the preferred form for making modifications,
      including but not limited to software source code, documentation
      source, and configuration files.

      "Object" form shall mean any form resulting from mechanical
      transformation or translation of a Source form, including but
      not limited to compiled object code, generated documentation,
      and conversions to other media types.

      "Work" shall mean the work of authorship, whether in Source or
      Object form, made available under the License, as indicated by a
      copyright notice that is included in or attached to the work
      (an example is provided in the Appendix below).

      "Derivative Works" shall mean any work, whether in Source or Object
      form, that is based on (or derived from) the Work and for which the
      editorial revisions, annotations, elaborations, or other modifications
      represent, as a whole, an original work of authorship. For the purposes
      of this License, Derivative Works shall not include works that remain
      separable from, or merely link (or bind by name) to the interfaces of,
      the Work and Derivative Works thereof.

      "Contribution" shall mean any work of authorship, including
      the original version of the Work and any modifications or additions
      to that Work or Derivative Works thereof, that is intentionally
      submitted to Licensor for inclusion in the Work by the copyright owner
      or by an individual or Legal Entity authorized to submit on behalf of
      the copyright owner. For the purposes of this definition, "submitted"
      means any form of electronic, verbal, or written communication sent
      to the Licensor or its representatives, including but not limited to
      communication on electronic mailing lists, source code control systems,
      and issue tracking systems that are managed by, or on behalf of, the
      Licensor for the purpose of discussing and improving the Work, but
      excluding communication that is conspicuously marked or otherwise
      designated in writing by the copyright owner as "Not a Contribution."

      "Contributor" shall mean Licensor and any individual or Legal Entity
      on behalf of whom a Contribution has been received by Licensor and
      subsequently incorporated within the Work.

   2. Grant of Copyright License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      copyright license to reproduce, prepare Derivative Works of,
      publicly display, publicly perform, sublicense, and distribute the
      Work and such Derivative Works in Source or Object form.

   3. Grant of Patent License. Subject to the terms and conditions of
      this License, each Contributor hereby grants to You a perpetual,
      worldwide, non-exclusive, no-charge, royalty-free, irrevocable
      (except as stated in this section) patent license to make, have made,
      use, offer to sell, sell, import, and otherwise transfer the Work,
      where such license applies only to those patent claims licensable
      by such Contributor that are necessarily infringed by their
      Contribution(s) alone or by combination of their Contribution(s)
      with the Work to which such Contribution(s) was submitted. If You
      institute patent litigation against any entity (including a
      cross-claim or counterclaim in a lawsuit) alleging that the Work
      or a Contribution incorporated within the Work constitutes direct
      or contributory patent infringement, then any patent licenses
      granted to You under this License for that Work shall terminate
      as of the date such litigation is filed.

   4. Redistribution. You may reproduce and distribute copies of the
      Work or Derivative Works thereof in any medium, with or without
      modifications, and in Source or Object form, provided that You
      meet the following conditions:

      (a) You must give any other recipients of the Work or
          Derivative Works a copy of this License; and

      (b) You must cause any modified files to carry prominent notices
          stating that You changed the files; and

      (c) You must retain, in the Source form of any Derivative Works
          that You distribute, all copyright, patent, trademark, and
          attribution notices from the Source form of the Work,
          excluding those notices that do not pertain to any part of
          the Derivative Works; and

      (d) If the Work includes a "NOTICE" text file as part of its
          distribution, then any Derivative Works that You distribute must
          include a readable copy of the attribution notices contained
          within such NOTICE file, excluding those notices that do not
          pertain to any part of the Derivative Works, in at least one
          of the following places: within a NOTICE text file distributed
          as part of the Derivative Works; within the Source form or
          documentation, if provided along with the Derivative Works; or,
          within a display generated by the Derivative Works, if and
          wherever such third-party notices normally appear. The contents
          of the NOTICE file are for informational purposes only and
          do not modify the License. You may add Your own attribution
          notices within Derivative Works that You distribute, alongside
          or as an addendum to the NOTICE text from the Work, provided
          that such additional attribution notices cannot be construed
          as modifying the License.

      You may add Your own copyright statement to Your modifications and
      may provide additional or different license terms and conditions
      for use, reproduction, or distribution of Your modifications, or
      for any such Derivative Works as a whole, provided Your use,
      reproduction, and distribution of the Work otherwise complies with
      the conditions stated in this License.

   5. Submission of Contributions. Unless You explicitly state otherwise,
      any Contribution intentionally submitted for inclusion in the Work
      by You to the Licensor shall be under the terms and conditions of
      this License, without any additional terms or conditions.
      Notwithstanding the above, nothing herein shall supersede or modify
      the terms of any separate license agreement you may have executed
      with Licensor regarding such Contributions.

   6. Trademarks. This License does not grant permission to use the trade
      names, trademarks, service marks, or product names of the Licensor,
      except as required for reasonable and customary use in describing the
      origin of the Work and reproducing the content of the NOTICE file.

   7. Disclaimer of Warranty. Unless required by applicable law or
      agreed to in writing, Licensor provides the Work (and each
      Contributor provides its Contributions) on an "AS IS" BASIS,
      WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
      implied, including, without limitation, any warranties or conditions
      of TITLE, NON-INFRINGEMENT, MERCHANTABILITY, or FITNESS FOR A
      PARTICULAR PURPOSE. You are solely responsible for determining the
      appropriateness of using or redistributing the Work and assume any
      risks associated with Your exercise of permissions under this License.

   8. Limitation of Liability. In no event and under no legal theory,
      whether in tort (including negligence), contract, or otherwise,
      unless required by applicable law (such as deliberate and grossly
      negligent acts) or agreed to in writing, shall any Contributor be
      liable to You for damages, including any direct, indirect, special,
      incidental, or consequential damages of any character arising as a
      result of this License or out of the use or inability to use the
      Work (including but not limited to damages for loss of goodwill,
      work stoppage, computer failure or malfunction, or any and all
      other commercial damages or losses), even if such Contributor
      has been advised of the possibility of such damages.

   9. Accepting Warranty or Additional Liability. While redistributing
      the Work or Derivative Works thereof, You may choose to offer,
      and charge a fee for, acceptance of support, warranty, indemnity,
      or other liability obligations and/or rights consistent with this
      License. However, in accepting such obligations, You may act only
      on Your own behalf and on Your sole responsibility, not on behalf
      of any other Contributor, and only if You agree to indemnify,
      defend, and hold each Contributor harmless for any liability
      incurred by, or claims asserted against, such Contributor by reason
      of your accepting any such warranty or additional liability.

   END OF TERMS AND CONDITIONS

   APPENDIX: How to apply the Apache License to your work.

      To apply the Apache License to your work, attach the following
      boilerplate notice, with the fields enclosed by brackets "[]"
      replaced with your own identifying information. (Don't include
      the brackets!)  The text should be enclosed in the appropriate
      comment syntax for the file format. We also recommend that a
      file or class name and description of purpose be included on the
      same "printed page" as the copyright notice for easier
      identification within third-party archives.

   Copyright [yyyy] [name of copyright owner]

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
````

## File: Makefile
````
# Copyright 2017-2020 Lei Ni (nilei81@gmail.com).
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GOCMD=go build -v

all: helloworld multigroup ondisk optimistic-write-lock

helloworld:
	$(GOCMD) -o example-helloworld github.com/lni/dragonboat-example/v3/helloworld

multigroup:
	$(GOCMD) -o example-multigroup github.com/lni/dragonboat-example/v3/multigroup

ondisk:
	$(GOCMD) -o example-ondisk github.com/lni/dragonboat-example/v3/ondisk

optimistic-write-lock:
	$(GOCMD) -o example-optimistic-write-lock github.com/lni/dragonboat-example/v3/optimistic-write-lock

clean:
	@rm -f example-helloworld example-multigroup example-ondisk example-optimistic-write-lock

.PHONY: helloworld multigroup ondisk optimistic-write-lock clean
````

## File: README.CHS.md
````markdown
## 关于 ##
本repo含[dragonboat](http://github.com/lni/dragonboat)项目的示例程序

本repo的master branch和release-3.3 branch针对dragonboat repo的master branch和各v3.3.x发布版。

需Go 1.17或更新的带[Go module](https://github.com/golang/go/wiki/Modules)支持的Go版本。

## 注意事项 ##
本repo中的程序均为示例，为了便于向用户展现dragonboat的基本用途，它们被刻意以最简单的方式实现而忽略了基本所有性能考虑。这些示例程序不能用于跑分用途。

## 安装 ##
假设计划下载例程代码到$HOME/src/dragonboat-example：
```
$ cd $HOME/src
$ git clone https://github.com/lni/dragonboat-example
```
编译所有例程：
```
$ cd $HOME/src/dragonboat-example
$ make
```

## 示例 ##

点选下列链接以获取具体例程信息。

* [示例 1](helloworld) - Hello World
* [示例 2](helloworld/README.DS.md) - State Machine 状态机
* [示例 3](multigroup/README.CHS.md) - 多个Raft组
* [示例 4](ondisk/README.CHS.md) - 基于磁盘的State Machine 状态机

## 下一步 ##
* [godoc](https://godoc.org/github.com/lni/dragonboat)
* 为[dragonboat](http://github.com/lni/dragonboat)项目贡献代码或报告bug！
````

## File: README.md
````markdown
## About / [中文版](README.CHS.md) ##
This repo contains examples for [dragonboat](http://github.com/lni/dragonboat).

The master branch and the release-3.3 branch of this repo target Dragonboat's master and v3.3.x releases.

Go 1.17 or later releases with [Go module](https://github.com/golang/go/wiki/Modules) support is required.

## Notice ##

Programs provided here in this repo are examples - they are intentionally created in a more straight forward way to help users to understand the basics of the dragonboat library. They are not benchmark programs.

## Install ##

To download the example code to say $HOME/src/dragonboat-example:
```
$ cd $HOME/src
$ git clone https://github.com/lni/dragonboat-example
```
Build all examples:
```
$ cd $HOME/src/dragonboat-example
$ make
```

## Examples ##

Click links below for more details.

* [Example 1](helloworld) - Hello World
* [Example 2](helloworld/README.DS.md) - State Machine
* [Example 3](multigroup) - Multiple Raft Groups
* [Example 4](ondisk) - On Disk State Machine
* [Example 5](optimistic-write-lock) - Optimistic Write Lock

## Next Step ##
* [godoc](https://godoc.org/github.com/lni/dragonboat)
* Contribute code or report bugs for [dragonboat](http://github.com/lni/dragonboat)
````
