package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"syscall"
	"time"
	"unsafe"
)

// 定义Windows API函数
var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procReadProcessMemory        = kernel32.NewProc("ReadProcessMemory")
	procWriteProcessMemory       = kernel32.NewProc("WriteProcessMemory")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = kernel32.NewProc("Process32FirstW")
	procProcess32Next            = kernel32.NewProc("Process32NextW")
	procVirtualQueryEx           = kernel32.NewProc("VirtualQueryEx")
)

const (
	PROCESS_ALL_ACCESS = 0x1F0FFF
	TH32CS_SNAPPROCESS = 0x00000002
)

// MEMORY_BASIC_INFORMATION 内存区域信息结构
type MEMORY_BASIC_INFORMATION struct {
	BaseAddress       uintptr
	AllocationBase    uintptr
	AllocationProtect uint32
	RegionSize        uintptr
	State             uint32
	Protect           uint32
	Type              uint32
}

type PROCESSENTRY32 struct {
	Size              uint32
	CntUsage          uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	CntThreads        uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	ExeFile           [260]uint16
}

// GameModifier 游戏修改器结构体
type GameModifier struct {
	processHandle syscall.Handle
	processID     uint32
}

// NewGameModifier 创建新的游戏修改器实例
func NewGameModifier() *GameModifier {
	return &GameModifier{}
}

// FindProcessByName 根据进程名查找进程ID
func (gm *GameModifier) FindProcessByName(processName string) (uint32, error) {
	snapshot, _, err := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == ^uintptr(0) {
		return 0, fmt.Errorf("无法创建进程快照: %v", err)
	}
	defer procCloseHandle.Call(snapshot)

	var pe32 PROCESSENTRY32
	pe32.Size = uint32(unsafe.Sizeof(pe32))

	ret, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&pe32)))
	if ret == 0 {
		return 0, fmt.Errorf("无法获取第一个进程")
	}

	for {
		exeFile := syscall.UTF16ToString(pe32.ExeFile[:])
		if exeFile == processName {
			return pe32.ProcessID, nil
		}

		ret, _, _ = procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&pe32)))
		if ret == 0 {
			break
		}
	}

	return 0, fmt.Errorf("未找到进程: %s", processName)
}

// OpenProcess 打开目标进程
func (gm *GameModifier) OpenProcess(processID uint32) error {
	handle, _, err := procOpenProcess.Call(PROCESS_ALL_ACCESS, 0, uintptr(processID))
	if handle == 0 {
		// 提供更详细的错误信息
		errMsg := err.Error()
		if errMsg == "Access is denied." {
			return fmt.Errorf("访问被拒绝: 请以管理员身份运行此程序")
		}
		return fmt.Errorf("无法打开进程 %d: %v", processID, err)
	}

	gm.processHandle = syscall.Handle(handle)
	gm.processID = processID
	return nil
}

// ReadMemory 读取指定地址的内存值
func (gm *GameModifier) ReadMemory(address uintptr, buffer []byte) error {
	var bytesRead uintptr
	ret, _, err := procReadProcessMemory.Call(
		uintptr(gm.processHandle),
		address,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
		uintptr(unsafe.Pointer(&bytesRead)),
	)

	if ret == 0 {
		return fmt.Errorf("读取内存失败: %v", err)
	}

	return nil
}

// WriteMemory 写入指定地址的内存值
func (gm *GameModifier) WriteMemory(address uintptr, data []byte) error {
	var bytesWritten uintptr
	ret, _, err := procWriteProcessMemory.Call(
		uintptr(gm.processHandle),
		address,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(unsafe.Pointer(&bytesWritten)),
	)

	if ret == 0 {
		return fmt.Errorf("写入内存失败: %v", err)
	}

	return nil
}

// Close 关闭进程句柄
func (gm *GameModifier) Close() {
	if gm.processHandle != 0 {
		procCloseHandle.Call(uintptr(gm.processHandle))
	}
}

// ScanForValue 扫描进程中指定值的内存地址
func (gm *GameModifier) ScanForValue(targetValue int32) ([]uintptr, error) {
	if gm.processHandle == 0 {
		return nil, fmt.Errorf("请先打开一个进程")
	}

	var addresses []uintptr
	var mbi MEMORY_BASIC_INFORMATION
	var address uintptr = 0
	const maxResults = 1000 // 增加最大结果数，提高找到目标的概率
	var scannedRegions int = 0
	var readableRegions int = 0
	var totalBytesScanned uint64 = 0

	fmt.Printf("开始扫描进程 %d，搜索值: %d...\n", gm.processID, targetValue)
	startTime := time.Now()

	// 遍历进程的整个虚拟地址空间
	for {
		// 查询当前地址的内存区域信息
		ret, _, _ := procVirtualQueryEx.Call(
			uintptr(gm.processHandle),
			address,
			uintptr(unsafe.Pointer(&mbi)),
			uintptr(unsafe.Sizeof(mbi)),
		)

		// 如果查询失败，退出循环
		if ret == 0 {
			break
		}

		scannedRegions++

		// 检查内存区域是否可读且已提交
		// State: 0x1000 = MEM_COMMIT, 0x10000 = MEM_FREE, 0x200000 = MEM_RESERVE
		if mbi.State == 0x1000 && mbi.RegionSize > 0 {
			// 移除大小限制，扫描所有已提交的内存区域

			// 检查保护属性是否可读
			// PAGE_READONLY=0x02, PAGE_READWRITE=0x04, PAGE_WRITECOPY=0x08
			// PAGE_EXECUTE_READ=0x20, PAGE_EXECUTE_READWRITE=0x40, PAGE_EXECUTE_WRITECOPY=0x80
			isReadable := (mbi.Protect&0x02 != 0) || (mbi.Protect&0x04 != 0) ||
				(mbi.Protect&0x08 != 0) || (mbi.Protect&0x20 != 0) || (mbi.Protect&0x40 != 0) ||
				(mbi.Protect&0x80 != 0)

			if isReadable {
				readableRegions++

				// 分配缓冲区读取内存 - 分批处理大内存区域
				regionSize := mbi.RegionSize
				const chunkSize = 1024 * 1024 // 1MB chunks

				for offset := uintptr(0); offset < regionSize; offset += chunkSize {
					// 计算当前块的大小
					currentChunkSize := chunkSize
					if offset+uintptr(currentChunkSize) > regionSize {
						currentChunkSize = int(regionSize - offset)
					}

					// 分配缓冲区读取内存
					buffer := make([]byte, currentChunkSize)
					err := gm.ReadMemory(mbi.BaseAddress+offset, buffer)

					if err == nil {
						totalBytesScanned += uint64(currentChunkSize)

						// 在缓冲区中搜索目标值
						for i := 0; i <= len(buffer)-4; i += 4 {
							// 将4个字节转换为int32（小端序）
							value := int32(buffer[i]) |
								int32(buffer[i+1])<<8 |
								int32(buffer[i+2])<<16 |
								int32(buffer[i+3])<<24

							if value == targetValue {
								addresses = append(addresses, mbi.BaseAddress+offset+uintptr(i))

								// 如果结果太多，提前退出
								if len(addresses) >= maxResults {
									elapsed := time.Since(startTime)
									fmt.Printf("已达到最大结果数限制 (%d)，耗时: %v\n", maxResults, elapsed)
									return addresses, nil
								}
							}
						}
					}
				}
			}
		}

		// 移动到下一个内存区域
		nextAddress := mbi.BaseAddress + mbi.RegionSize

		// 防止无限循环（地址空间上限或回绕）
		if nextAddress <= address {
			break
		}
		address = nextAddress
	}

	elapsed := time.Since(startTime)
	fmt.Printf("扫描完成！\n")
	fmt.Printf("  扫描区域数: %d\n", scannedRegions)
	fmt.Printf("  可读区域数: %d\n", readableRegions)
	fmt.Printf("  扫描总字节数: %.2f MB\n", float64(totalBytesScanned)/(1024*1024))
	fmt.Printf("  找到匹配地址: %d\n", len(addresses))
	fmt.Printf("  总耗时: %v\n", elapsed)

	return addresses, nil
}

// refineSearch 在上次搜索结果中再次搜索指定值
func refineSearch(targetValue int32) ([]uintptr, error) {
	if len(lastSearchAddresses) == 0 {
		return nil, fmt.Errorf("没有上次的搜索结果，请先进行首次搜索")
	}

	var addresses []uintptr
	fmt.Printf("在上次 %d 个结果中再次搜索值: %d...\n", len(lastSearchAddresses), targetValue)
	startTime := time.Now()

	// 遍历上次搜索的所有地址，检查当前值是否匹配
	for _, addr := range lastSearchAddresses {
		buffer := make([]byte, 4)
		err := modifier.ReadMemory(addr, buffer)

		if err == nil {
			// 将4个字节转换为int32（小端序）
			value := int32(buffer[0]) |
				int32(buffer[1])<<8 |
				int32(buffer[2])<<16 |
				int32(buffer[3])<<24

			if value == targetValue {
				addresses = append(addresses, addr)
			}
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("再次搜索完成！找到 %d 个匹配地址，耗时: %v\n", len(addresses), elapsed)

	return addresses, nil
}

// 全局游戏修改器实例
var modifier = NewGameModifier()

// 存储上次搜索的地址列表，用于再次搜索
var lastSearchAddresses []uintptr

// API响应结构
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 查找进程API
func findProcessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProcessName string `json:"process_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "无效的请求"})
		return
	}

	processID, err := modifier.FindProcessByName(req.ProcessName)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
		return
	}

	err = modifier.OpenProcess(processID)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Message: fmt.Sprintf("成功打开进程 ID: %d", processID),
		Data:    map[string]interface{}{"process_id": processID},
	})
}

// 搜索值API
func searchValueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Value        int32 `json:"value"`
		RefineSearch bool  `json:"refine_search,omitempty"` // 是否在上次结果中再次搜索
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "无效的请求"})
		return
	}

	var addresses []uintptr
	var err error

	if req.RefineSearch && len(lastSearchAddresses) > 0 {
		// 在上次搜索结果中再次搜索
		addresses, err = refineSearch(req.Value)
	} else {
		// 全新搜索
		addresses, err = modifier.ScanForValue(req.Value)
		if err == nil {
			// 保存搜索结果供下次使用
			lastSearchAddresses = addresses
		}
	}

	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
		return
	}

	// 更新上次搜索结果
	lastSearchAddresses = addresses

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Message: fmt.Sprintf("找到 %d 个匹配地址", len(addresses)),
		Data:    map[string]interface{}{"addresses": addresses},
	})
}

// 修改值API
func modifyValueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Address  uint64 `json:"address"`
		NewValue int32  `json:"new_value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: "无效的请求"})
		return
	}

	// 将值转换为字节数组
	data := make([]byte, 4)
	data[0] = byte(req.NewValue & 0xFF)
	data[1] = byte((req.NewValue >> 8) & 0xFF)
	data[2] = byte((req.NewValue >> 16) & 0xFF)
	data[3] = byte((req.NewValue >> 24) & 0xFF)

	err := modifier.WriteMemory(uintptr(req.Address), data)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Success: false, Message: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Message: fmt.Sprintf("成功修改地址 0x%X 的值为 %d", req.Address, req.NewValue),
	})
}

// Web界面
const htmlPage = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>游戏修改器</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: 'Microsoft YaHei', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 15px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            padding: 40px;
            max-width: 600px;
            width: 100%;
        }
        h1 {
            text-align: center;
            color: #333;
            margin-bottom: 30px;
            font-size: 28px;
        }
        .section {
            margin-bottom: 25px;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 10px;
        }
        .section h2 {
            color: #667eea;
            margin-bottom: 15px;
            font-size: 18px;
        }
        .input-group {
            display: flex;
            gap: 10px;
            margin-bottom: 10px;
        }
        input[type="text"], input[type="number"] {
            flex: 1;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 14px;
            transition: border-color 0.3s;
        }
        input[type="text"]:focus, input[type="number"]:focus {
            outline: none;
            border-color: #667eea;
        }
        button {
            padding: 12px 24px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 14px;
            font-weight: bold;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
        }
        button:active {
            transform: translateY(0);
        }
        .result {
            margin-top: 20px;
            padding: 15px;
            border-radius: 8px;
            background: #e8f5e9;
            border-left: 4px solid #4caf50;
            display: none;
        }
        .result.error {
            background: #ffebee;
            border-left-color: #f44336;
        }
        .result.warning {
            background: #fff3e0;
            border-left-color: #ff9800;
        }
        .result.show {
            display: block;
        }
        .address-list {
            margin-top: 10px;
            max-height: 200px;
            overflow-y: auto;
            background: #f5f5f5;
            padding: 10px;
            border-radius: 5px;
            font-family: 'Courier New', monospace;
            font-size: 12px;
        }
        .address-item {
            padding: 5px;
            margin: 2px 0;
            background: white;
            border-radius: 3px;
            cursor: pointer;
            transition: background 0.2s;
        }
        .address-item:hover {
            background: #e3f2fd;
        }
        .info {
            text-align: center;
            color: #666;
            margin-top: 20px;
            font-size: 12px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎮 游戏修改器</h1>
        
        <div class="section">
            <h2>1. 选择进程</h2>
            <div class="input-group">
                <input type="text" id="processName" placeholder="输入进程名称 (例如: notepad.exe)">
                <button onclick="findProcess()">查找进程</button>
            </div>
        </div>

        <div class="section">
            <h2>2. 搜索数值</h2>
            <div class="input-group">
                <input type="number" id="searchValue" placeholder="输入要搜索的值">
                <button onclick="searchValue()">首次搜索</button>
                <button onclick="refineSearch()" style="background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);">再次搜索</button>
            </div>
            <p style="font-size: 12px; color: #666; margin-top: 8px;">
                💡 提示：首次搜索会得到很多结果，改变游戏数值后点击“再次搜索”来缩小范围
            </p>
        </div>

        <div class="section">
            <h2>3. 修改数值</h2>
            <div class="input-group">
                <input type="text" id="address" placeholder="内存地址 (例如: 0x12345678)">
                <input type="number" id="newValue" placeholder="新值">
                <button onclick="modifyValue()">修改值</button>
            </div>
        </div>

        <div id="result" class="result"></div>
        <div id="addressList" class="address-list" style="display: none;"></div>
        
        <div class="info">
            <p>⚠️ 本程序仅用于学习和研究目的</p>
        </div>
    </div>

    <script>
        function showResult(message, type = 'success') {
            const resultDiv = document.getElementById('result');
            resultDiv.textContent = message;
            resultDiv.className = 'result show ' + type;
        }

        async function findProcess() {
            const processName = document.getElementById('processName').value;
            if (!processName) {
                showResult('错误: 请输入进程名称', 'error');
                return;
            }

            try {
                const response = await fetch('/api/find-process', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({process_name: processName})
                });
                const data = await response.json();
                
                // 检查是否是权限错误
                if (data.message && data.message.includes('管理员')) {
                    showResult(data.message, 'warning');
                } else {
                    showResult(data.message, data.success ? 'success' : 'error');
                }
            } catch (error) {
                showResult('错误: ' + error.message, 'error');
            }
        }

        async function searchValue() {
            const value = document.getElementById('searchValue').value;
            if (!value) {
                showResult('错误: 请输入要搜索的值', 'error');
                return;
            }

            showResult('正在扫描内存，请稍候...', 'success');
            document.getElementById('addressList').style.display = 'none';

            try {
                const response = await fetch('/api/search-value', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({value: parseInt(value), refine_search: false})
                });
                const data = await response.json();
                showResult(data.message, data.success ? 'success' : 'error');
                
                // 显示地址列表
                if (data.success && data.data && data.data.addresses && data.data.addresses.length > 0) {
                    displayAddresses(data.data.addresses);
                } else if (data.success && data.data.addresses.length === 0) {
                    showResult('未找到匹配的地址，请尝试其他数值', 'warning');
                }
            } catch (error) {
                showResult('错误: ' + error.message, 'error');
            }
        }

        async function refineSearch() {
            const value = document.getElementById('searchValue').value;
            if (!value) {
                showResult('错误: 请输入要搜索的值', 'error');
                return;
            }

            showResult('正在上次结果中再次搜索...', 'success');
            document.getElementById('addressList').style.display = 'none';

            try {
                const response = await fetch('/api/search-value', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({value: parseInt(value), refine_search: true})
                });
                const data = await response.json();
                showResult(data.message, data.success ? 'success' : 'error');
                
                // 显示地址列表
                if (data.success && data.data && data.data.addresses && data.data.addresses.length > 0) {
                    displayAddresses(data.data.addresses);
                } else if (data.success && data.data.addresses.length === 0) {
                    showResult('未找到匹配的地址，可能目标地址已变化', 'warning');
                }
            } catch (error) {
                showResult('错误: ' + error.message, 'error');
            }
        }

        function displayAddresses(addresses) {
            const addressListDiv = document.getElementById('addressList');
            addressListDiv.innerHTML = '<strong>找到的地址（点击复制到修改框）：</strong><br>';
            
            addresses.forEach(addr => {
                const div = document.createElement('div');
                div.className = 'address-item';
                div.textContent = '0x' + addr.toString(16).toUpperCase();
                div.onclick = () => {
                    document.getElementById('address').value = '0x' + addr.toString(16).toUpperCase();
                };
                addressListDiv.appendChild(div);
            });
            
            addressListDiv.style.display = 'block';
        }

        async function modifyValue() {
            const address = document.getElementById('address').value;
            const newValue = document.getElementById('newValue').value;

            if (!address || !newValue) {
                showResult('错误: 请输入内存地址和新值', 'error');
                return;
            }

            try {
                const response = await fetch('/api/modify-value', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({
                        address: parseInt(address.startsWith('0x') ? address : '0x' + address, 16),
                        new_value: parseInt(newValue)
                    })
                });
                const data = await response.json();
                
                // 检查是否是权限错误
                if (data.message && data.message.includes('管理员')) {
                    showResult(data.message, 'warning');
                } else {
                    showResult(data.message, data.success ? 'success' : 'error');
                }
            } catch (error) {
                showResult('错误: ' + error.message, 'error');
            }
        }
    </script>
</body>
</html>
`

// 主页处理器
func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlPage))
}

func main() {
	defer modifier.Close()

	// 设置路由
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/find-process", findProcessHandler)
	http.HandleFunc("/api/search-value", searchValueHandler)
	http.HandleFunc("/api/modify-value", modifyValueHandler)

	// 启动服务器
	port := "8080" // 更改端口避免冲突
	fmt.Printf("🎮 游戏修改器已启动!\n")
	fmt.Printf("🌐 请在浏览器中打开: http://localhost:%s\n", port)
	fmt.Printf("⚠️  按 Ctrl+C 停止服务器\n\n")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("错误: %v\n", err)
	}
}
