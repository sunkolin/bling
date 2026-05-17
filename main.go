package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/yaml.v3"
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

// GUIConfig GUI配置结构体
type GUIConfig struct {
	Title          string `yaml:"title"`
	Width          int    `yaml:"width"`
	Height         int    `yaml:"height"`
	DefaultProcess string `yaml:"default_process"`
}

// AppConfig 应用配置结构体
type AppConfig struct {
	GUI GUIConfig `yaml:"gui"`
}

// loadConfig 加载配置文件
func loadConfig() *AppConfig {
	config := &AppConfig{
		GUI: GUIConfig{
			Title:          "💎 Bling - 游戏修改器",
			Width:          800,
			Height:         600,
			DefaultProcess: "",
		},
	}

	data, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Println("警告: 无法读取配置文件，使用默认配置")
		return config
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		fmt.Println("警告: 配置文件格式错误，使用默认配置")
		return config
	}

	return config
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

	return addresses, nil
}

// 全局游戏修改器实例
var modifier = NewGameModifier()

func main() {
	defer modifier.Close()

	// 加载配置
	config := loadConfig()

	// 自动查找并打开默认进程
	if config.GUI.DefaultProcess != "" {
		processID, err := modifier.FindProcessByName(config.GUI.DefaultProcess)
		if err == nil {
			modifier.OpenProcess(processID)
		}
	}

	// 创建Fyne应用
	myApp := app.New()
	myWindow := myApp.NewWindow(config.GUI.Title)
	myWindow.Resize(fyne.NewSize(float32(config.GUI.Width), float32(config.GUI.Height)))

	// 设置窗口居中显示
	myWindow.CenterOnScreen()

	// 加载窗口图标 + 任务栏图标（使用嵌入的资源）
	myWindow.SetIcon(appIcon) // 设置窗口图标

	// 创建原值输入框
	oldValueEntry := widget.NewEntry()
	oldValueEntry.SetPlaceHolder("输入原值（要搜索的值）")

	// 创建新值输入框
	newValueEntry := widget.NewEntry()
	newValueEntry.SetPlaceHolder("输入新值（要修改成的值）")

	// 创建结果显示
	resultLabel := widget.NewLabel("")
	resultLabel.Wrapping = fyne.TextWrapWord

	// 搜索并修改功能
	searchAndModify := func() {
		oldValueText := oldValueEntry.Text
		newValueText := newValueEntry.Text

		if oldValueText == "" || newValueText == "" {
			resultLabel.SetText("❌ 错误: 请输入原值和新值")
			return
		}

		oldValue, err := strconv.ParseInt(oldValueText, 10, 32)
		if err != nil {
			resultLabel.SetText("❌ 错误: 原值格式不正确")
			return
		}

		newValue, err := strconv.ParseInt(newValueText, 10, 32)
		if err != nil {
			resultLabel.SetText("❌ 错误: 新值格式不正确")
			return
		}

		resultLabel.SetText("⏳ 正在搜索并修改，请稍候...")

		// 搜索所有匹配的地址
		addresses, err := modifier.ScanForValue(int32(oldValue))
		if err != nil {
			resultLabel.SetText("❌ 错误: " + err.Error())
			return
		}

		if len(addresses) == 0 {
			resultLabel.SetText("⚠️ 未找到值为 " + oldValueText + " 的地址")
			return
		}

		// 将所有找到的地址修改为新值
		data := make([]byte, 4)
		data[0] = byte(newValue & 0xFF)
		data[1] = byte((newValue >> 8) & 0xFF)
		data[2] = byte((newValue >> 16) & 0xFF)
		data[3] = byte((newValue >> 24) & 0xFF)

		modifiedCount := 0
		for _, addr := range addresses {
			err := modifier.WriteMemory(addr, data)
			if err == nil {
				modifiedCount++
			}
		}

		resultLabel.SetText(fmt.Sprintf("✅ 成功修改 %d 个地址，值从 %s 改为 %s", modifiedCount, oldValueText, newValueText))
	}

	// 创建搜索并修改按钮
	searchModifyBtn := widget.NewButton("搜索并修改", searchAndModify)
	searchModifyBtn.Importance = widget.HighImportance

	// 创建提示信息
	tipLabel := widget.NewLabel("💡 提示：输入游戏中的当前值和目标值，点击按钮后将自动搜索并修改所有匹配的地址")
	tipLabel.Wrapping = fyne.TextWrapWord

	// 创建主布局
	mainContent := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabel("原值:"), nil, oldValueEntry),
		container.NewBorder(nil, nil, widget.NewLabel("新值:"), nil, newValueEntry),
		widget.NewSeparator(),
		searchModifyBtn,
		widget.NewSeparator(),
		resultLabel,
		tipLabel,
	)

	// 添加滚动容器
	scroll := container.NewScroll(mainContent)

	// 设置窗口内容
	myWindow.SetContent(scroll)

	// 设置窗口关闭事件，强制退出程序
	myWindow.SetOnClosed(func() {
		modifier.Close()
		os.Exit(0)
	})

	// 显示窗口并运行
	myWindow.ShowAndRun()
}
