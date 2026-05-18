// common 包：提供全局初始化、命令行参数解析、全局变量等功能
package common

import (
	"flag"          // 命令行参数解析
	"fmt"           // 格式化输出
	"log"           // 简单日志（用于初始化阶段的致命错误）
	"os"            // 环境变量读取、文件操作
	"path/filepath" // 路径处理：获取绝对路径等

	"github.com/songquanpeng/one-api/common/config" // 全局配置：会话密钥等
	"github.com/songquanpeng/one-api/common/logger" // 日志系统
)

// 命令行参数定义
// 使用 flag 包注册命令行参数，程序启动时可通过以下方式指定：
//
//	./one-api --port 8080 --log-dir /var/log/one-api --version --help
var (
	Port         = flag.Int("port", 3000, "the listening port")                  // 监听端口，默认 3000
	PrintVersion = flag.Bool("version", false, "print version and exit")         // 打印版本号后退出
	PrintHelp    = flag.Bool("help", false, "print help and exit")               // 打印帮助信息后退出
	LogDir       = flag.String("log-dir", "./logs", "specify the log directory") // 日志目录，默认 ./logs
)

// printHelp 打印程序的帮助信息，包括版本号、版权、GitHub 地址和用法说明
func printHelp() {
	fmt.Println("One API " + Version + " - All in one API service for OpenAI API.")
	fmt.Println("Copyright (C) 2023 JustSong. All rights reserved.")
	fmt.Println("GitHub: https://github.com/songquanpeng/one-api")
	fmt.Println("Usage: one-api [--port <port>] [--log-dir <log directory>] [--version] [--help]")
}

// Init 是通用初始化函数，在 main.go 中最先被调用
// 主要完成以下工作：
//  1. 解析命令行参数
//  2. 处理 --version 和 --help 特殊参数
//  3. 从环境变量加载会话密钥（SESSION_SECRET）
//  4. 从环境变量加载 SQLite 数据库路径（SQLITE_PATH）
//  5. 确保日志目录存在
func Init() {
	// 解析命令行参数，将值填充到上面定义的 flag 变量中
	flag.Parse()

	// 如果指定了 --version，打印版本号后退出
	if *PrintVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	// 如果指定了 --help，打印帮助信息后退出
	if *PrintHelp {
		printHelp()
		os.Exit(0)
	}

	// 从环境变量读取会话密钥，用于加密 Cookie 中的会话数据
	// 如果值为默认的示例值 "random_string"，则发出警告（不使用该值，保留代码中的默认密钥）
	// 生产环境务必设置一个随机的强密钥
	if os.Getenv("SESSION_SECRET") != "" {
		if os.Getenv("SESSION_SECRET") == "random_string" {
			logger.SysError("SESSION_SECRET is set to an example value, please change it to a random string.")
		} else {
			config.SessionSecret = os.Getenv("SESSION_SECRET")
		}
	}

	// 从环境变量读取 SQLite 数据库文件路径
	// 默认路径在 model 包中定义，此处可覆盖
	if os.Getenv("SQLITE_PATH") != "" {
		SQLitePath = os.Getenv("SQLITE_PATH")
	}

	// 确保日志目录存在：如果指定了日志目录，则转换为绝对路径并创建目录
	if *LogDir != "" {
		var err error
		// 将相对路径转换为绝对路径，便于日志记录
		*LogDir, err = filepath.Abs(*LogDir)
		if err != nil {
			log.Fatal(err)
		}
		// 如果目录不存在，则创建（权限 0777：所有人可读写执行）
		if _, err := os.Stat(*LogDir); os.IsNotExist(err) {
			err = os.Mkdir(*LogDir, 0777)
			if err != nil {
				log.Fatal(err)
			}
		}
		// 将最终确定的日志目录路径设置到 logger 模块
		logger.LogDir = *LogDir
	}
}
