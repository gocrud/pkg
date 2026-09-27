package errorx

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// kv 是命名参数(有序,后设覆盖先设)。
type kv struct {
	key string
	val any
}

// infoSource 是 Code/Builder/Error 三形态共用的内部视图。
type infoSource interface {
	codeRef() any
	codeKey() string
	templateStr() string
	argList() []kv
	stackFrames() []frame
	causeErr() error
}

// Error 是终结态错误:不可变,携带错误码、命名参数、堆栈(可选)与底层 cause。
// 应用层判定建议使用 errors.Is(err, ErrX)、errorx.CodeOf、errorx.ErrorOf。
type Error struct {
	ref      any
	key      string
	template string
	args     []kv
	stack    []frame
	cause    error
}

// Error 渲染单行消息 "[CODE] 消息",永远不含堆栈。
func (e *Error) Error() string { return oneLine(e.key, e.template, e.args) }

// Unwrap 返回底层 cause,支持标准 errors.Is/As 链式遍历。
func (e *Error) Unwrap() error { return e.cause }

// Is 按错误码定义的指针身份匹配,命中后继续沿 cause 链匹配。
func (e *Error) Is(target error) bool {
	if e.ref != nil && target == e.ref {
		return true
	}
	return e.cause != nil && errors.Is(e.cause, target)
}

// Format 支持 %+v 完整渲染(消息+箭头树堆栈+cause 链),其余 verb 输出单行。
func (e *Error) Format(s fmt.State, verb rune) {
	formatErr(s, verb, e.Error(), e.renderFull())
}

// CodeStr 返回规范串错误码(数字码 strconv 化)。
func (e *Error) CodeStr() string { return e.key }

// Args 返回命名参数(后设覆盖先设)。
func (e *Error) Args() map[string]any { return argsToMap(e.args) }

// Msg 返回渲染后的消息(不含 "[CODE] " 前缀),供协议出口层作为用户提示。
func (e *Error) Msg() string { return renderMessage(e.template, e.args) }

// StackStr 返回箭头树格式的纯堆栈文本,未捕获堆栈时为空串。
func (e *Error) StackStr() string { return renderStack(e.stack) }

func (e *Error) renderFull() string {
	return renderFull(e.key, e.template, e.args, e.stack, e.cause)
}

func (e *Error) codeRef() any         { return e.ref }
func (e *Error) codeKey() string      { return e.key }
func (e *Error) templateStr() string  { return e.template }
func (e *Error) argList() []kv        { return e.args }
func (e *Error) stackFrames() []frame { return e.stack }
func (e *Error) causeErr() error      { return e.cause }

// Builder 是链式中间态,也实现 error,链上任何一步都可直接 return。
// 约定 Builder 为单次链式使用,setter 原地追加参数。
type Builder struct {
	ref      any
	key      string
	template string
	args     []kv
	stack    []frame
}

func newBuilder[T CodeValue](c *Code[T]) *Builder {
	return &Builder{ref: c, key: c.key, template: c.template}
}

// Error 渲染单行消息。
func (b *Builder) Error() string { return oneLine(b.key, b.template, b.args) }

// Format 支持 %+v 完整渲染。
func (b *Builder) Format(s fmt.State, verb rune) {
	formatErr(s, verb, b.Error(), renderFull(b.key, b.template, b.args, b.stack, nil))
}

// Is 按错误码定义匹配。
func (b *Builder) Is(target error) bool { return b.ref != nil && target == b.ref }

// Wrap 终结链式构建并附加底层 cause,返回不可变的 *Error。
func (b *Builder) Wrap(err error) *Error {
	e := b.toError()
	e.cause = err
	return e
}

func (b *Builder) toError() *Error {
	return &Error{ref: b.ref, key: b.key, template: b.template, args: b.args, stack: b.stack}
}

// Str 设置字符串参数。
func (b *Builder) Str(key, val string) *Builder { return b.set(key, val) }

// Int 设置 int 参数。
func (b *Builder) Int(key string, val int) *Builder { return b.set(key, val) }

// Int64 设置 int64 参数。
func (b *Builder) Int64(key string, val int64) *Builder { return b.set(key, val) }

// Any 设置任意类型参数。
func (b *Builder) Any(key string, val any) *Builder { return b.set(key, val) }

func (b *Builder) set(key string, val any) *Builder {
	b.args = append(b.args, kv{key: key, val: val})
	return b
}

func (b *Builder) codeRef() any         { return b.ref }
func (b *Builder) codeKey() string      { return b.key }
func (b *Builder) templateStr() string  { return b.template }
func (b *Builder) argList() []kv        { return b.args }
func (b *Builder) stackFrames() []frame { return b.stack }
func (b *Builder) causeErr() error      { return nil }

// ErrorOf 把 Code/Builder/Error 三形态统一归一为 *Error 视图(沿 Unwrap 链
// 查找),是应用层提取 CodeStr/Args/Msg/StackStr/Unwrap 的统一入口。
func ErrorOf(err error) (*Error, bool) {
	if err == nil {
		return nil, false
	}
	var src infoSource
	if !errors.As(err, &src) {
		return nil, false
	}
	if e, ok := src.(*Error); ok {
		return e, true
	}
	return &Error{
		ref:      src.codeRef(),
		key:      src.codeKey(),
		template: src.templateStr(),
		args:     src.argList(),
		stack:    src.stackFrames(),
		cause:    src.causeErr(),
	}, true
}

// CodeOf 返回规范串错误码(沿 Unwrap 链查找),供值比较/协议翻译层/日志使用。
func CodeOf(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	var src infoSource
	if !errors.As(err, &src) {
		return "", false
	}
	return src.codeKey(), true
}

// Wrap 以指定错误码包装底层原因(非链式兜底,参数请走链式 setter)。
func Wrap[T CodeValue](err error, code *Code[T]) error {
	if err == nil {
		return nil
	}
	return &Error{ref: code, key: code.key, template: code.template, cause: err}
}

// oneLine 渲染单行消息:"[CODE] 消息"(模板缺参时占位符为空并折叠多余空白)。
func oneLine(key, template string, args []kv) string {
	msg := renderMessage(template, args)
	if msg == "" {
		return "[" + key + "]"
	}
	return "[" + key + "] " + msg
}

// renderMessage 渲染模板并折叠多余空白(不含错误码前缀)。
func renderMessage(template string, args []kv) string {
	return strings.Join(strings.Fields(renderNamed(template, args)), " ")
}

// renderNamed 用命名参数填充模板 {{.name}},缺参填空串。
func renderNamed(template string, args []kv) string {
	if template == "" {
		return ""
	}
	var b strings.Builder
	rest := template
	for {
		start := strings.Index(rest, "{{")
		if start < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:start])
		endRel := strings.Index(rest[start+2:], "}}")
		if endRel < 0 {
			b.WriteString(rest[start:])
			break
		}
		end := start + 2 + endRel
		name := strings.TrimPrefix(strings.TrimSpace(rest[start+2:end]), ".")
		b.WriteString(lookupArg(args, name))
		rest = rest[end+2:]
	}
	return b.String()
}

func lookupArg(args []kv, name string) string {
	for i := len(args) - 1; i >= 0; i-- {
		if args[i].key == name {
			return fmt.Sprint(args[i].val)
		}
	}
	return ""
}

func argsToMap(args []kv) map[string]any {
	m := make(map[string]any, len(args))
	for _, a := range args {
		m[a.key] = a.val
	}
	return m
}

// formatErr 是 Code/Builder/Error 共用的 fmt.Formatter 实现。
func formatErr(s fmt.State, verb rune, single, full string) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, full)
			return
		}
		io.WriteString(s, single)
	default:
		fmt.Fprintf(s, fmtVerb(verb), single)
	}
}

func fmtVerb(verb rune) string {
	switch verb {
	case 's':
		return "%s"
	case 'q':
		return "%q"
	case 'x':
		return "%x"
	case 'X':
		return "%X"
	default:
		return "%v"
	}
}
