package pkg

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/my-go-p2p/internal"
)

type ModuleTerminal struct {
}

func NewModuleTerminal() *ModuleTerminal {
	return &ModuleTerminal{}
}

func (s *ModuleTerminal) Start(ctx context.Context) {
	appState, ok := ctx.Value(internal.CTX_Key_AppState).(internal.AppState)
	if !ok {
		log.Fatalln("ctx.appstate not exists")
		return
	}
	log.Println("[ModuleTerminal]", "start")
	appState.AppWaitGroup.Add(1)
	defer func() {
		appState.AppWaitGroup.Done()
		log.Println("[ModuleTerminal]", "stop")
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		s.startReplLoop(ctx)
		defer wg.Done()
	}()

	wg.Wait()

}

func (s *ModuleTerminal) startReplLoop(ctx context.Context) {
	lines := make(chan string)
	go func() {
		time.Sleep(2 * time.Second)
		fmt.Println(">cmd")
		s := bufio.NewScanner(os.Stdin)
		s.Split(bufio.ScanLines)
		for s.Scan() {
			lines <- s.Text()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			close(lines)
			return
		case cmd := <-lines:
			s.handleTerminalCmd(ctx, cmd)
		}
	}
}

func (s *ModuleTerminal) handleTerminalCmd(ctx context.Context, cmdLine string) {
	appState, ok := ctx.Value(internal.CTX_Key_AppState).(internal.AppState)
	if !ok {
		log.Fatalln("ctx.appstate not exists")
		return
	}

	tmp := strings.Split(cmdLine, " ")
	if len(tmp) == 0 {
		return
	}
	cmd, _ := tmp[0], tmp[1:]

	switch cmd {
	case ".info":
		fmt.Println(formatToDisplay(map[string]any{
			"nodeName":   appState.Config.NodeName,
			"tcpAddress": appState.Config.Tcp_address,
		}))
	case ".list":
		replyChan := make(chan []internal.CMD_CmdAdminListNodeReplyItem, 1)
		appState.Chan_reception_cmd <- internal.CMD_CmdAdminListNode{
			Reply: replyChan,
		}

		data := <-replyChan
		fmt.Println(formatToDisplay(data))
		close(replyChan)

	default:
		fmt.Println(`
Help:
	.info	print node info
	.list	print remote node list
		`)
	}
}

func formatToDisplay(val any) string {
	s, err := json.Marshal(val)
	if err != nil {
		return "unknown"
	}
	return string(s)
}
