package main

import (
	"fmt"
	"strconv"
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2"
)

// 定义Windows API函数
var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess = kernel32.NewProc("OpenProcess")
	procReadProcessMemory = kernel32.NewProc("ReadProcessMemory")
	procWriteProcessMemory = kernel32.NewProc("WriteProcessMemory")
	procCloseHandle = kernel32.NewProc("CloseHandle")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First = kernel32.NewProc("Process32FirstW")
	procProcess32Next = kernel32.NewProc("Process32NextW")
)

const (
	PROCESS_ALL_ACCESS = 0x1F0FFF
	TH32CS_SNAPPROCESS = 0x00000002
)

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
	// 这里是一个简化的实现，实际应用中需要更复杂的内存扫描逻辑
	// 由于Windows内存管理的复杂性，完整实现会非常复杂
	// 此示例仅演示基本概念
	
	var addresses []uintptr
	
	// 在实际应用中，这里应该遍历进程的内存空间
	// 但由于安全和系统限制，这需要更高级的技术
	
	fmt.Println("注意：完整的内存扫描功能需要更复杂的实现")
	fmt.Println("此示例仅展示基本架构")
	
	return addresses, nil
}

func main() {
	// 创建Fyne应用
	myApp := app.New()
	myWindow := myApp.NewWindow("游戏修改器")

	// 创建游戏修改器实例
	modifier := NewGameModifier()
	defer modifier.Close()

	// 创建UI组件
	processEntry := widget.NewEntry()
	processEntry.SetPlaceHolder("输入进程名称 (例如: notepad.exe)")

	valueEntry := widget.NewEntry()
	valueEntry.SetPlaceHolder("输入要搜索的值")

	addressEntry := widget.NewEntry()
	addressEntry.SetPlaceHolder("输入内存地址 (十六进制, 例如: 0x12345678)")

	newValueEntry := widget.NewEntry()
	newValueEntry.SetPlaceHolder("输入新值")

	resultLabel := widget.NewLabel("")

	// 查找进程按钮
	findProcessBtn := widget.NewButton("查找进程", func() {
		processName := processEntry.Text
		if processName == "" {
			resultLabel.SetText("错误: 请输入进程名称")
			return
		}

		processID, err := modifier.FindProcessByName(processName)
		if err != nil {
			resultLabel.SetText(fmt.Sprintf("错误: %v", err))
		} else {
			err = modifier.OpenProcess(processID)
			if err != nil {
				resultLabel.SetText(fmt.Sprintf("错误: %v", err))
			} else {
				resultLabel.SetText(fmt.Sprintf("成功打开进程 ID: %d", processID))
			}
		}
	})

	// 搜索值按钮
	searchValueBtn := widget.NewButton("搜索值", func() {
		valueStr := valueEntry.Text
		if valueStr == "" {
			resultLabel.SetText("错误: 请输入要搜索的值")
			return
		}

		value, err := strconv.ParseInt(valueStr, 10, 32)
		if err != nil {
			resultLabel.SetText(fmt.Sprintf("错误: 无效的数字 - %v", err))
			return
		}

		addresses, err := modifier.ScanForValue(int32(value))
		if err != nil {
			resultLabel.SetText(fmt.Sprintf("错误: %v", err))
		} else {
			if len(addresses) == 0 {
				resultLabel.SetText("未找到匹配的值")
			} else {
				resultLabel.SetText(fmt.Sprintf("找到 %d 个匹配地址", len(addresses)))
			}
		}
	})

	// 修改值按钮
	modifyValueBtn := widget.NewButton("修改值", func() {
		addressStr := addressEntry.Text
		newValueStr := newValueEntry.Text

		if addressStr == "" || newValueStr == "" {
			resultLabel.SetText("错误: 请输入内存地址和新值")
			return
		}

		// 解析地址（假设是十六进制格式）
		var address uint64
		_, err := fmt.Sscanf(addressStr, "0x%x", &address)
		if err != nil {
			// 尝试十进制解析
			addr, parseErr := strconv.ParseUint(addressStr, 10, 64)
			if parseErr != nil {
				resultLabel.SetText(fmt.Sprintf("错误: 无效的内存地址格式 - %v", err))
				return
			}
			address = addr
		}

		// 解析新值
		newValue, err := strconv.ParseInt(newValueStr, 10, 32)
		if err != nil {
			resultLabel.SetText(fmt.Sprintf("错误: 无效的新值 - %v", err))
			return
		}

		// 将值转换为字节数组
		data := make([]byte, 4)
		data[0] = byte(newValue & 0xFF)
		data[1] = byte((newValue >> 8) & 0xFF)
		data[2] = byte((newValue >> 16) & 0xFF)
		data[3] = byte((newValue >> 24) & 0xFF)

		err = modifier.WriteMemory(uintptr(address), data)
		if err != nil {
			resultLabel.SetText(fmt.Sprintf("错误: %v", err))
		} else {
			resultLabel.SetText(fmt.Sprintf("成功修改地址 0x%X 的值为 %d", address, newValue))
		}
	})

	// 布局
	content := container.NewVBox(
		widget.NewLabel("游戏修改器"),
		container.NewHBox(
			processEntry,
			findProcessBtn,
		),
		container.NewHBox(
			valueEntry,
			searchValueBtn,
		),
		container.NewHBox(
			addressEntry,
			newValueEntry,
			modifyValueBtn,
		),
		resultLabel,
	)

	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(600, 300))
	myWindow.ShowAndRun()
}