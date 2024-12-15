package godotenv

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
)

const (
	charComment       = '#'
	prefixSingleQuote = '\''
	prefixDoubleQuote = '"'

	exportPrefix = "export"
)

func hasQuotePrefix(src []byte) (prefix byte, isQuored bool) {
	if len(src) == 0 {
		return 0, false
	}

	switch prefix := src[0]; prefix {
	case prefixSingleQuote, prefixDoubleQuote:
		return prefix, true
	default:
		return 0, false
	}
}

// 找到第一个非空格字符的索引
func indexOfNonSpaceChar(src []byte) int {
	// 在输入的字节切片中查找
	return bytes.IndexFunc(src, func(r rune) bool {
		// 判断字符是否为空白
		return !unicode.IsSpace(r)
	})
}

func isLineEnd(r rune) bool {
	if r == '\n' || r == '\r' {
		return true
	}
	return false
}

// '\t'：水平制表符（Tab）
// '\v'：垂直制表符
// '\f'：换页符
// '\r'：回车符
// ' '：普通空格
// 0x85：下一行(Next Line)字符
// 0xA0：不间断空格(Non-Breaking Space)
func isSpace(r rune) bool {
	switch r {
	case '\t', '\v', '\f', '\r', ' ', 0x85, 0xA0:
		return true
	}
	return false
}

func isCharFunc(char rune) func(rune) bool {
	return func(v rune) bool {
		return v == char
	}
}

var (
	// \\. 匹配的是：一个反斜杠 + 紧跟其后的任意单个字符
	// text := `Hello \$world \{test\}`
	// matches := escapeRegex.FindAllString(text, -1)
	// matches 可能是 ["\\$", "\\{"]
	escapeRegex = regexp.MustCompile(`\\.`)
	// (\\)? 匹配 \
	// (\$)  匹配 $
	// (\()? 匹配 (
	// \{?	 匹配 {
	// ([A-Z0-9_]+)? 匹配只能包含大写字母、数字和下划线
	// \}?   匹配 }
	// + 表示至少一个字符
	// ? 表示是可选的
	expandVarRegex = regexp.MustCompile(`(\\)?(\$)(\()?\{?([A-Z0-9_]+)?\}?`)
	//  \\ 匹配一个反斜杠 \
	// [^$] 匹配除 $ 以外的任意单个字符
	// () 表示创建了一个捕获组
	// 会匹配：
	// `\a`   匹配反斜杠加a
	// `\#`   匹配反斜杠加#
	// `\*`   匹配反斜杠加*
	//
	// 不会匹配：
	// `\$`   因为包含了 $ 符号
	// `\\`   因为没有后续字符
	unescapeCharsRegex = regexp.MustCompile(`\\([^$])`)
)

func expandVariables(v string, m map[string]string) string {
	return expandVarRegex.ReplaceAllStringFunc(v, func(s string) string {
		// ${HOME}
		// [${HOME}, "", $, "", HOME]
		submatch := expandVarRegex.FindStringSubmatch(s)

		//if submatch == nil {
		//	return s
		//}
		if submatch[1] == "\\" || submatch[2] == "(" {
			return submatch[0][1:]
		} else if submatch[4] != "" {
			if val, ok := m[submatch[4]]; ok {
				return val
			}
			if val, ok := os.LookupEnv(submatch[4]); ok {
				fmt.Println(ok, val)
				return val
			}
			return m[submatch[4]]
		}
		return s
	})
}

func expandEscapes(str string) string {
	// ReplaceAllStringFunc 函数会对匹配到的每个子串执行一次指定的函数，整个过程是循环遍历的，会处理字符串中所有匹配的部分
	out := escapeRegex.ReplaceAllStringFunc(str, func(match string) string {
		// 每次进来，先去掉转义符 \
		c := strings.TrimPrefix(match, `\`)
		// \r、\n 不处理，其他的就会把转义符处理掉
		switch c {
		case "n":
			return "\n"
		case "r":
			return "\r"
		default:
			return match
		}
	})
	// 去掉转义符
	return unescapeCharsRegex.ReplaceAllString(out, "$1")
}

const doubleQuoteSpecialChars = "\\\n\r\"!$`"

func doubleQuoteEscape(line string) string {
	for _, c := range doubleQuoteSpecialChars {
		toReplace := "\\" + string(c)
		if c == '\n' {
			toReplace = `\n`
		}
		if c == '\r' {
			toReplace = `\r`
		}
		line = strings.Replace(line, string(c), toReplace, -1)
	}
	return line
}
