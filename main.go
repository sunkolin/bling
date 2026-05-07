package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Config 定义配置文件结构
type Config struct {
	App struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Version     string `yaml:"version"`
	} `yaml:"app"`
}

// calculateMD5 计算字符串的MD5值
func calculateMD5(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

// calculateSHA1 计算字符串的SHA1值
func calculateSHA1(input string) string {
	hash := sha1.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

// calculateSHA256 计算字符串的SHA256值
func calculateSHA256(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// splitString 使用多个分隔符分割字符串
func splitString(input string, delimiters string) []string {
	// 将分隔符字符串拆分为单个字符
	delimRunes := []rune(delimiters)
	
	// 创建一个替换函数，将所有分隔符替换为同一个特殊字符
	replacer := strings.NewReplacer(func() []string {
		var replacements []string
		for _, r := range delimRunes {
			replacements = append(replacements, string(r), "\x00")
		}
		return replacements
	}()...)
	
	// 替换所有分隔符
	replaced := replacer.Replace(input)
	
	// 按照特殊字符分割
	parts := strings.Split(replaced, "\x00")
	
	// 过滤空字符串
	var result []string
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	
	return result
}

// joinStrings 将多行字符串用分隔符合并
func joinStrings(input string, delimiter string) string {
	// 按换行符分割
	lines := strings.Split(input, "\n")
	
	// 过滤空行
	var nonEmptyLines []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			nonEmptyLines = append(nonEmptyLines, trimmed)
		}
	}
	
	// 用分隔符连接
	return strings.Join(nonEmptyLines, delimiter)
}

// calculateSHA512 计算字符串的SHA512值
func calculateSHA512(input string) string {
	hash := sha512.Sum512([]byte(input))
	return hex.EncodeToString(hash[:])
}

// getNextCronTimes 获取cron表达式接下来10次执行时间
func getNextCronTimes(expression string) ([]string, error) {
	schedule, err := cron.ParseStandard(expression)
	if err != nil {
		return nil, fmt.Errorf("解析cron表达式失败: %v", err)
	}

	var times []string
	now := time.Now()
	
	for i := 0; i < 10; i++ {
		next := schedule.Next(now)
		times = append(times, next.Format("2006-01-02 15:04:05"))
		now = next
	}

	return times, nil
}

// generateRandomString 生成指定长度的随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// loadConfig 从配置文件加载配置
func loadConfig() (*Config, error) {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &config, nil
}

// findShortForLong 根据长参数名查找对应的短参数名
func findShortForLong(longName string, paramGroups map[string]string) (string, bool) {
	for short, long := range paramGroups {
		if long == longName {
			return short, true
		}
	}
	return "", false
}

func main() {
	// 自定义 Usage 函数，使 -h 也显示紧凑格式
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of sun:\n")
		
		// 定义参数分组映射（短参数 -> 长参数）
		paramGroups := map[string]string{
			"v":       "version",
		}
		
		// 记录已显示的参数
		shown := make(map[string]bool)
		
		flag.VisitAll(func(f *flag.Flag) {
			// 如果这个参数已经被作为短参数显示过了，跳过
			if shown[f.Name] {
				return
			}
			
			// 检查是否有对应的长参数或短参数需要合并
			var displayName string
			if longName, ok := paramGroups[f.Name]; ok {
				// 这是短参数，需要和长参数一起显示
				displayName = fmt.Sprintf("-%s, --%s", f.Name, longName)
				shown[longName] = true // 标记长参数已显示
			} else if _, exists := findShortForLong(f.Name, paramGroups); exists {
				// 这是长参数，且已经有短参数显示过它了，跳过
				return
			} else {
				// 普通参数，直接显示
				displayName = fmt.Sprintf("--%s", f.Name)
			}
			
			// 根据参数类型添加占位符
			switch f.Value.String() {
			case "0":
				// int 类型
				displayName += " <arg>"
			case "false":
				// bool 类型，不需要占位符
			default:
				// string 类型
				displayName += " <arg>"
			}
			
			fmt.Fprintf(os.Stderr, "  %-30s %s\n", displayName, f.Usage)
		})
	}

	// 1. 加载配置文件
	config, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	// 2. 定义命令行参数
	verbose := flag.Bool("verbose", false, "显示详细信息")
	showVersion := flag.Bool("v", false, "显示版本信息")
	showVersionLong := flag.Bool("version", false, "显示版本信息")
	showTime := flag.Bool("time", false, "显示当前时间")
	showDate := flag.Bool("date", false, "显示当前日期")
	showDateTime := flag.Bool("datetime", false, "显示当前日期和时间")
	showRandom := flag.Int("random", 0, "生成指定长度的随机字符串")
	showUUID := flag.Bool("uuid", false, "生成UUID")
	md5Input := flag.String("md5", "", "计算字符串的MD5值")
	sha1Input := flag.String("sha1", "", "计算字符串的SHA1值")
	sha256Input := flag.String("sha256", "", "计算字符串的SHA256值")
	sha512Input := flag.String("sha512", "", "计算字符串的SHA512值")
	cronExpr := flag.String("cron", "", "显示cron表达式最近10次执行时间")
	uppercaseInput := flag.String("uppercase", "", "将字符串转换为大写")
	lowercaseInput := flag.String("lowercase", "", "将字符串转换为小写")
	jsonFormatInput := flag.String("jsonformat", "", "格式化JSON字符串")
	showTimestamp := flag.Bool("timestamp", false, "显示当前时间戳")
	delimiter := flag.String("delimiter", "", "分隔符/连接符（与--split或--join一起使用）")
	splitInput := flag.String("split", "", "待分割的字符串（需要配合--delimiter使用）")
	joinInput := flag.String("join", "", "待合并的多行字符串（需要配合--delimiter使用）")

	// 3. 解析参数
	flag.Parse()

	// 4. 如果没有提供任何参数，显示帮助信息
	if len(os.Args) == 1 {
		flag.Usage()
		return
	}

	// 5. 检查是否请求版本信息
	if *showVersion || *showVersionLong {
		fmt.Println(config.App.Version)
		return
	}

	// 5. 检查是否请求时间信息
	if *showTime {
		fmt.Println(time.Now().Format("15:04:05"))
		return
	}

	// 6. 检查是否请求日期信息
	if *showDate {
		fmt.Println(time.Now().Format("2006-01-02"))
		return
	}

	// 7. 检查是否请求日期时间信息
	if *showDateTime {
		fmt.Println(time.Now().Format("2006-01-02 15:04:05"))
		return
	}

	// 8. 检查是否请求生成随机字符串
	if *showRandom > 0 {
		fmt.Println(generateRandomString(*showRandom))
		return
	}

	// 9. 检查是否请求生成UUID
	if *showUUID {
		fmt.Println(uuid.New().String())
		return
	}

	// 10. 检查是否请求计算MD5
	if *md5Input != "" {
		fmt.Println(calculateMD5(*md5Input))
		return
	}

	// 11. 检查是否请求计算SHA1
	if *sha1Input != "" {
		fmt.Println(calculateSHA1(*sha1Input))
		return
	}

	// 12. 检查是否请求计算SHA256
	if *sha256Input != "" {
		fmt.Println(calculateSHA256(*sha256Input))
		return
	}

	// 13. 检查是否请求计算SHA512
	if *sha512Input != "" {
		fmt.Println(calculateSHA512(*sha512Input))
		return
	}

	// 14. 检查是否请求cron执行时间
	if *cronExpr != "" {
		times, err := getNextCronTimes(*cronExpr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		for _, t := range times {
			fmt.Println(t)
		}
		return
	}

	// 15. 检查是否请求转换为大写
	if *uppercaseInput != "" {
		fmt.Println(strings.ToUpper(*uppercaseInput))
		return
	}

	// 16. 检查是否请求转换为小写
	if *lowercaseInput != "" {
		fmt.Println(strings.ToLower(*lowercaseInput))
		return
	}

	// 17. 检查是否请求格式化JSON
	if *jsonFormatInput != "" {
		var jsonData interface{}
		err := json.Unmarshal([]byte(*jsonFormatInput), &jsonData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: 无效的JSON格式 - %v\n", err)
			os.Exit(1)
		}
		formatted, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: 格式化JSON失败 - %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(formatted))
		return
	}

	// 18. 检查是否请求显示时间戳
	if *showTimestamp {
		fmt.Println(time.Now().Unix())
		return
	}

	// 19. 检查是否请求分割字符串
	if *splitInput != "" && *delimiter != "" {
		parts := splitString(*splitInput, *delimiter)
		for _, part := range parts {
			fmt.Println(part)
		}
		return
	}

	// 20. 检查是否请求合并字符串
	if *joinInput != "" && *delimiter != "" {
		fmt.Println(joinStrings(*joinInput, *delimiter))
		return
	}

	// 21. 默认业务逻辑（无参数时不会到达这里）
	if *verbose {
		fmt.Println("我是常用工具集")
		return
	}

	fmt.Println("========================")
	fmt.Println("   我的通用 CLI 工具")
	fmt.Println("========================")
	fmt.Println("\n✅ CLI 运行成功！")
}