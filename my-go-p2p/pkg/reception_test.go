package pkg_test

// func Test_ReceptionAddRemoveNode(t *testing.T) {
// 	wg := sync.WaitGroup{}
// 	moduleReception := pkg.NewModuleReception()

// 	appState := internal.
// 		NewAppState()

// 	ctx := context.Background()
// 	ctx = context.WithValue(ctx, internal.CTX_Key_AppState, *appState)
// 	ctx, cancel := context.WithCancel(ctx)
// 	go moduleReception.Start(ctx)

// 	doAddNode := func(i int) {
// 		defer wg.Done()
// 		nodeName := fmt.Sprintf("node-%d", i)
// 		addr := net.TCPAddr{
// 			IP:   net.ParseIP("127.0.0.1"),
// 			Port: 5000 + i,
// 		}
// 		appState.Chan_reception_cmd <- internal.CMD_AddNode{
// 			NodeName: nodeName,
// 			Addr:     addr,
// 		}
// 	}

// 	for i := 0; i < 10; i++ {
// 		wg.Add(1)
// 		go doAddNode(i)
// 	}

// 	wg.Wait()

// 	if !moduleReception.HasRemoteNodeName("node-5") {
// 		t.Error("node-5 should exists")
// 	}

// 	ranRemoveAtId := rand.Intn(10)
// 	tmp := fmt.Sprintf("node-%d", ranRemoveAtId)

// 	appState.Chan_reception_cmd <- internal.CMD_RemoveNode{
// 		NodeName: tmp,
// 	}
// 	time.Sleep(10 * time.Millisecond)

// 	if moduleReception.HasRemoteNodeName(tmp) {
// 		t.Errorf("%s should deleted", tmp)
// 	}

// 	cancel()
// }
