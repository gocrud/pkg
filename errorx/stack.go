package errorx

import (
	"errors"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

// frame 是一帧调用信息(函数名 + 文件 + 行号)。
type frame struct {
	fn   string
	file string
	line int
}

var (
	mainModulePath string
	mainModuleElem string
	errorxPrefix   = "github.com/gocrud/pkg/errorx."
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		mainModulePath = info.Main.Path
	}
	if mainModulePath != "" {
		errorxPrefix = mainModulePath + "/errorx."
		if idx := strings.LastIndex(mainModulePath, "/"); idx >= 0 {
			mainModuleElem = mainModulePath[idx+1:]
		} else {
			mainModuleElem = mainModulePath
		}
	}
}

// captureStack 捕获当前 goroutine 完整堆栈并裁剪为业务帧。
func captureStack() []frame {
	buf := make([]byte, 64*1024)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return parseStack(string(buf[:n]))
		}
		buf = make([]byte, len(buf)*2)
	}
}

// parseStack 解析 runtime.Stack 文本并过滤帧:
//   - 跳过 errorx 内部帧(函数名以 "模块路径/errorx." 开头);
//   - 跳过 Go 语言层面帧(文件位于 GOROOT 下或函数包名为 runtime);
//   - 其余(业务+第三方依赖)全部保留,内层优先。
func parseStack(text string) []frame {
	lines := strings.Split(text, "\n")
	goroot := filepath.ToSlash(runtime.GOROOT())
	frames := make([]frame, 0, 16)
	for i := 0; i < len(lines); i++ {
		fnLine := strings.TrimSpace(lines[i])
		if fnLine == "" || strings.HasPrefix(fnLine, "goroutine ") ||
			strings.HasPrefix(fnLine, "created by ") || strings.HasPrefix(fnLine, "[") {
			continue
		}
		if strings.HasPrefix(lines[i], "\t") {
			continue // 文件行与函数行成对,只处理函数行
		}
		if i+1 >= len(lines) {
			break
		}
		fileLine := lines[i+1]
		if !strings.HasPrefix(fileLine, "\t") {
			continue
		}
		fn := fnLine
		if idx := strings.LastIndex(fn, "("); idx > 0 && strings.HasSuffix(fn, ")") {
			fn = fn[:idx]
		}
		if strings.HasPrefix(fn, errorxPrefix) || strings.HasPrefix(fn, "runtime.") {
			i++
			continue
		}
		file, line, ok := parseFileLine(strings.TrimSpace(fileLine), goroot)
		i++
		if !ok {
			continue
		}
		frames = append(frames, frame{fn: fn, file: file, line: line})
	}
	return frames
}

// parseFileLine 解析 "\t/path/file.go:123 +0x56" 形式的文件行,
// 位于 GOROOT 下的文件返回 false。
func parseFileLine(s, goroot string) (string, int, bool) {
	if idx := strings.Index(s, " +0x"); idx >= 0 {
		s = s[:idx]
	}
	idx := strings.LastIndex(s, ":")
	if idx <= 0 {
		return "", 0, false
	}
	line, err := strconv.Atoi(s[idx+1:])
	if err != nil {
		return "", 0, false
	}
	file := filepath.ToSlash(s[:idx])
	if goroot != "" && strings.HasPrefix(file, goroot+"/") {
		return "", 0, false
	}
	return file, line, true
}

// trimFuncName 去掉函数名的模块路径前缀。
func trimFuncName(fn string) string {
	if mainModulePath != "" && strings.HasPrefix(fn, mainModulePath+"/") {
		return fn[len(mainModulePath)+1:]
	}
	return fn
}

// trimFilePath 把文件路径截短为相对模块根的路径。
func trimFilePath(file string) string {
	if mainModuleElem == "" {
		return file
	}
	if idx := strings.Index(file, "/"+mainModuleElem+"/"); idx >= 0 {
		return file[idx+len(mainModuleElem)+2:]
	}
	return file
}

// renderStack 以箭头树格式渲染堆栈:main → 调用点自上而下,
// 位置列按最大函数名宽度右对齐,连续同包帧折叠包名。
func renderStack(frames []frame) string {
	if len(frames) == 0 {
		return ""
	}
	rev := make([]frame, len(frames))
	for i, f := range frames {
		rev[len(frames)-1-i] = f
	}
	type display struct {
		name string
		loc  string
	}
	displays := make([]display, len(rev))
	width := 0
	prevPkg := ""
	for i, f := range rev {
		name := trimFuncName(f.fn)
		pkg := name
		if idx := strings.LastIndex(pkg, "."); idx > 0 {
			pkg = pkg[:idx]
		}
		if i > 0 && pkg == prevPkg {
			if idx := strings.LastIndex(name, "."); idx >= 0 {
				name = name[idx+1:]
			}
		}
		prevPkg = pkg
		loc := ""
		if f.file != "" {
			loc = trimFilePath(f.file)
			if f.line > 0 {
				loc += ":" + strconv.Itoa(f.line)
			}
		}
		if len(name) > width {
			width = len(name)
		}
		displays[i] = display{name: name, loc: loc}
	}
	var b strings.Builder
	for i, d := range displays {
		if i == 0 {
			b.WriteString("├─ ")
		} else {
			b.WriteString("│")
			b.WriteString(strings.Repeat("   ", i-1))
			b.WriteString("  └─ ")
		}
		b.WriteString(d.name)
		if d.loc != "" {
			b.WriteString(strings.Repeat(" ", width-len(d.name)+2))
			b.WriteString(d.loc)
		}
		if i < len(displays)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// renderFull 渲染完整错误信息:单行消息 + 堆栈 + cause 链。
func renderFull(key, template string, args []kv, frames []frame, cause error) string {
	var b strings.Builder
	b.WriteString(oneLine(key, template, args))
	if len(frames) > 0 {
		b.WriteByte('\n')
		b.WriteString(renderStack(frames))
	}
	renderCauseChain(&b, cause)
	return strings.TrimRight(b.String(), "\n")
}

const maxCauseDepth = 8

// renderCauseChain 渲染 cause 链:errorx 错误显示其单行消息,否则显示 Error()。
func renderCauseChain(b *strings.Builder, cause error) {
	indent := 0
	for cur := cause; cur != nil && indent < maxCauseDepth; cur = errors.Unwrap(cur) {
		if indent > 0 {
			b.WriteString(strings.Repeat("   ", indent-1))
			b.WriteString("└─ ")
		}
		b.WriteString("caused by: ")
		b.WriteString(errSingleLine(cur))
		b.WriteByte('\n')
		indent++
	}
}

func errSingleLine(err error) string {
	if src, ok := err.(infoSource); ok {
		return oneLine(src.codeKey(), src.templateStr(), src.argList())
	}
	return err.Error()
}
