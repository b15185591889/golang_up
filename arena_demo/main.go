package main

import (
	"arena_demo/pkg/core"
	"fmt"
	"net/http"
	"strconv"
)

var engine *core.Engine

func main() {
	// 1. 启动 Core (C World)
	engine = core.NewEngine()
	engine.Start()

	// 2. 启动 HTTP Server (Go World)
	http.HandleFunc("/calc", handleCalc)
	http.HandleFunc("/order", handleOrder)

	fmt.Println("Hybrid Server listening on :8080")
	fmt.Println("  - /calc?val=10  -> Calc Task")
	fmt.Println("  - /order?p=100&q=5 -> Order Task")

	http.ListenAndServe(":8080", nil)
}

func handleCalc(w http.ResponseWriter, r *http.Request) {
	valStr := r.URL.Query().Get("val")
	val, _ := strconv.Atoi(valStr)

	// 创建一个 channel 用于接收 C World 的结果
	respChan := make(chan any, 1)

	// 3. 跨界投递：Go -> C
	task := core.Task{
		Type:  core.TaskTypeCalc,
		Value: val,
		Resp:  respChan,
	}

	// 如果队列满了，这里可以选择阻塞或者报错
	if !engine.Queue.Push(task) {
		http.Error(w, "Core Busy", 503)
		return
	}

	// 4. 等待结果：Go <- C
	result := <-respChan

	fmt.Fprintf(w, "Calc Result: %v\n", result)
}

func handleOrder(w http.ResponseWriter, r *http.Request) {
	price, _ := strconv.ParseFloat(r.URL.Query().Get("p"), 64)
	qty, _ := strconv.Atoi(r.URL.Query().Get("q"))
	uid, _ := strconv.Atoi(r.URL.Query().Get("uid"))
	if uid == 0 {
		uid = 1 // default user
	}

	respChan := make(chan any, 1)

	// 预分配 Log Buffer (可以使用 sync.Pool 复用)
	logBuf := make([]byte, 0, 1024)

	task := core.Task{
		Type:     core.TaskTypeOrder,
		Price:    price,
		Quantity: qty,
		Value:    uid, // Reuse Value as UserID
		Resp:     respChan,
		LogBuf:   logBuf,
	}

	if !engine.Queue.Push(task) {
		http.Error(w, "Core Busy", 503)
		return
	}

	// 4. 获取结果
	resAny := <-respChan
	result := resAny.(core.OrderResult)

	// 5. 打印 Core 返回的日志 (异步打印，不影响 Core)
	if len(result.Log) > 0 {
		fmt.Printf("[AsyncLog] %s", result.Log)
	}

	fmt.Fprintf(w, "Order Total: %.2f\nProcessed At: %d\nLog: %s", result.Total, result.ProcessedAt, result.Log)
}
